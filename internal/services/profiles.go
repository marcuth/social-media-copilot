package services

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/marcuth/social-media-copilot/internal/catalog"
	"github.com/marcuth/social-media-copilot/internal/models"
)

// ProfileService manages profiles, their platforms and their themes.
type ProfileService struct {
	DB *sql.DB
}

const profileColumns = `id, name, description, niche, audience, tone, language, color, created_at, updated_at`

func scanProfile(row interface{ Scan(...any) error }) (models.Profile, error) {
	var p models.Profile
	err := row.Scan(&p.ID, &p.Name, &p.Description, &p.Niche, &p.Audience, &p.Tone,
		&p.Language, &p.Color, &p.CreatedAt, &p.UpdatedAt)
	p.Platforms = []models.ProfilePlatform{}
	return p, err
}

func (s *ProfileService) loadPlatforms(profiles []models.Profile) error {
	if len(profiles) == 0 {
		return nil
	}
	index := map[int64]int{}
	for i, p := range profiles {
		index[p.ID] = i
	}
	rows, err := s.DB.Query(`SELECT profile_id, platform, handle FROM profile_platforms`)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var id int64
		var pp models.ProfilePlatform
		if err := rows.Scan(&id, &pp.Platform, &pp.Handle); err != nil {
			return err
		}
		if i, ok := index[id]; ok {
			profiles[i].Platforms = append(profiles[i].Platforms, pp)
		}
	}
	// Keep the catalog order so every screen lists platforms the same way.
	for i := range profiles {
		sortPlatforms(profiles[i].Platforms)
	}
	return rows.Err()
}

func sortPlatforms(list []models.ProfilePlatform) {
	order := map[string]int{}
	for i, p := range catalog.Platforms {
		order[p.ID] = i
	}
	for i := 1; i < len(list); i++ {
		for j := i; j > 0 && order[list[j].Platform] < order[list[j-1].Platform]; j-- {
			list[j], list[j-1] = list[j-1], list[j]
		}
	}
}

// ListProfiles returns every profile with its platforms, alphabetically.
func (s *ProfileService) ListProfiles() ([]models.Profile, error) {
	rows, err := s.DB.Query(`SELECT ` + profileColumns + ` FROM profiles ORDER BY name COLLATE NOCASE`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []models.Profile{}
	for rows.Next() {
		p, err := scanProfile(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, s.loadPlatforms(out)
}

// GetProfile returns one profile.
func (s *ProfileService) GetProfile(id int64) (models.Profile, error) {
	p, err := scanProfile(s.DB.QueryRow(`SELECT `+profileColumns+` FROM profiles WHERE id = ?`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return p, ErrNotFound
	}
	if err != nil {
		return p, err
	}
	list := []models.Profile{p}
	if err := s.loadPlatforms(list); err != nil {
		return p, err
	}
	return list[0], nil
}

// SaveProfile creates (ID 0) or updates a profile and replaces its platforms.
func (s *ProfileService) SaveProfile(p models.Profile) (models.Profile, error) {
	p.Name = strings.TrimSpace(p.Name)
	if p.Name == "" {
		return p, errors.New("dê um nome ao perfil")
	}
	seen := map[string]bool{}
	for _, pp := range p.Platforms {
		if _, ok := catalog.PlatformByID(pp.Platform); !ok {
			return p, fmt.Errorf("plataforma desconhecida: %s", pp.Platform)
		}
		if seen[pp.Platform] {
			return p, fmt.Errorf("plataforma repetida: %s", pp.Platform)
		}
		seen[pp.Platform] = true
	}
	p.Language = orDefault(p.Language, "pt-BR")
	p.Color = orDefault(p.Color, "#3F51D9")
	now := nowStamp()

	tx, err := s.DB.Begin()
	if err != nil {
		return p, err
	}
	defer tx.Rollback()

	if p.ID == 0 {
		res, err := tx.Exec(`INSERT INTO profiles (name, description, niche, audience, tone, language, color, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			p.Name, p.Description, p.Niche, p.Audience, p.Tone, p.Language, p.Color, now, now)
		if err != nil {
			return p, err
		}
		p.ID, _ = res.LastInsertId()
	} else {
		res, err := tx.Exec(`UPDATE profiles SET name = ?, description = ?, niche = ?, audience = ?, tone = ?,
			language = ?, color = ?, updated_at = ? WHERE id = ?`,
			p.Name, p.Description, p.Niche, p.Audience, p.Tone, p.Language, p.Color, now, p.ID)
		if err != nil {
			return p, err
		}
		if n, _ := res.RowsAffected(); n == 0 {
			return p, ErrNotFound
		}
	}

	if _, err := tx.Exec(`DELETE FROM profile_platforms WHERE profile_id = ?`, p.ID); err != nil {
		return p, err
	}
	for _, pp := range p.Platforms {
		if _, err := tx.Exec(`INSERT INTO profile_platforms (profile_id, platform, handle) VALUES (?, ?, ?)`,
			p.ID, pp.Platform, strings.TrimSpace(pp.Handle)); err != nil {
			return p, err
		}
	}
	if err := tx.Commit(); err != nil {
		return p, err
	}
	return s.GetProfile(p.ID)
}

// DeleteProfile removes a profile together with its themes and posts.
func (s *ProfileService) DeleteProfile(id int64) error {
	_, err := s.DB.Exec(`DELETE FROM profiles WHERE id = ?`, id)
	return err
}

// ListThemes returns the themes of a profile, or of every profile when
// profileID is 0.
func (s *ProfileService) ListThemes(profileID int64) ([]models.Theme, error) {
	q := `SELECT id, profile_id, name, description, color, created_at FROM themes`
	args := []any{}
	if profileID != 0 {
		q += ` WHERE profile_id = ?`
		args = append(args, profileID)
	}
	q += ` ORDER BY name COLLATE NOCASE`
	rows, err := s.DB.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []models.Theme{}
	for rows.Next() {
		var t models.Theme
		if err := rows.Scan(&t.ID, &t.ProfileID, &t.Name, &t.Description, &t.Color, &t.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// SaveTheme creates (ID 0) or updates a theme.
func (s *ProfileService) SaveTheme(t models.Theme) (models.Theme, error) {
	t.Name = strings.TrimSpace(t.Name)
	if t.Name == "" {
		return t, errors.New("dê um nome ao tema")
	}
	if t.ProfileID == 0 {
		return t, errors.New("escolha o perfil do tema")
	}
	t.Color = orDefault(t.Color, "#8A88A0")
	if t.ID == 0 {
		t.CreatedAt = nowStamp()
		res, err := s.DB.Exec(`INSERT INTO themes (profile_id, name, description, color, created_at) VALUES (?, ?, ?, ?, ?)`,
			t.ProfileID, t.Name, t.Description, t.Color, t.CreatedAt)
		if err != nil {
			return t, err
		}
		t.ID, _ = res.LastInsertId()
		return t, nil
	}
	_, err := s.DB.Exec(`UPDATE themes SET name = ?, description = ?, color = ? WHERE id = ?`,
		t.Name, t.Description, t.Color, t.ID)
	return t, err
}

// DeleteTheme removes a theme; its posts keep existing without a theme.
func (s *ProfileService) DeleteTheme(id int64) error {
	_, err := s.DB.Exec(`DELETE FROM themes WHERE id = ?`, id)
	return err
}
