package services

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/marcuth/social-media-copilot/internal/catalog"
	"github.com/marcuth/social-media-copilot/internal/models"
)

type PostService struct {
	DB *sql.DB
}

const postColumns = `id, profile_id, theme_id, title, description, script, notes, tags, hashtags,
	format, status, scheduled_at, published_at, postpone_count, created_at, updated_at`

type scanner interface{ Scan(...any) error }

func scanPost(row scanner) (models.Post, error) {
	var p models.Post
	var theme sql.NullInt64
	var tags, hashtags string
	var scheduled, published sql.NullString
	err := row.Scan(&p.ID, &p.ProfileID, &theme, &p.Title, &p.Description, &p.Script, &p.Notes,
		&tags, &hashtags, &p.Format, &p.Status, &scheduled, &published, &p.PostponeCount,
		&p.CreatedAt, &p.UpdatedAt)
	p.ThemeID = nullInt(theme)
	p.Tags = decodeList(tags)
	p.Hashtags = decodeList(hashtags)
	p.ScheduledAt = nullString(scheduled)
	p.PublishedAt = nullString(published)
	p.Platforms = []models.PostPlatform{}
	return p, err
}

func (s *PostService) loadPlatforms(q queryer, posts []models.Post) error {
	if len(posts) == 0 {
		return nil
	}
	index := map[int64]int{}
	ids := make([]any, len(posts))
	for i, p := range posts {
		index[p.ID] = i
		ids[i] = p.ID
	}
	placeholders := strings.TrimSuffix(strings.Repeat("?,", len(ids)), ",")
	rows, err := q.Query(`SELECT post_id, platform, status, url, published_at FROM post_platforms
		WHERE post_id IN (`+placeholders+`)`, ids...)
	if err != nil {
		return err
	}
	defer rows.Close()
	order := map[string]int{}
	for i, p := range catalog.Platforms {
		order[p.ID] = i
	}
	for rows.Next() {
		var id int64
		var pp models.PostPlatform
		var published sql.NullString
		if err := rows.Scan(&id, &pp.Platform, &pp.Status, &pp.URL, &published); err != nil {
			return err
		}
		pp.PublishedAt = nullString(published)
		list := &posts[index[id]].Platforms
		*list = append(*list, pp)
		for j := len(*list) - 1; j > 0 && order[(*list)[j].Platform] < order[(*list)[j-1].Platform]; j-- {
			(*list)[j], (*list)[j-1] = (*list)[j-1], (*list)[j]
		}
	}
	return rows.Err()
}

type queryer interface {
	Query(string, ...any) (*sql.Rows, error)
	QueryRow(string, ...any) *sql.Row
}

// ListPosts returns posts matching the filter. Scheduled posts come first in
// date order, followed by unscheduled ones (newest first).
func (s *PostService) ListPosts(f models.PostFilter) ([]models.Post, error) {
	where := []string{"1 = 1"}
	args := []any{}
	if f.ProfileID != nil {
		where = append(where, "profile_id = ?")
		args = append(args, *f.ProfileID)
	}
	if f.ThemeID != nil {
		where = append(where, "theme_id = ?")
		args = append(args, *f.ThemeID)
	}
	if f.Status != "" {
		where = append(where, "status = ?")
		args = append(args, f.Status)
	}
	if f.Platform != "" {
		where = append(where, "EXISTS (SELECT 1 FROM post_platforms pp WHERE pp.post_id = posts.id AND pp.platform = ?)")
		args = append(args, f.Platform)
	}
	if f.Unscheduled {
		where = append(where, "scheduled_at IS NULL")
	}
	if f.From != "" {
		where = append(where, "substr(scheduled_at, 1, 10) >= ?")
		args = append(args, f.From)
	}
	if f.To != "" {
		where = append(where, "substr(scheduled_at, 1, 10) <= ?")
		args = append(args, f.To)
	}
	if q := strings.TrimSpace(f.Search); q != "" {
		where = append(where, "(title LIKE ? OR description LIKE ? OR script LIKE ? OR notes LIKE ?)")
		like := "%" + q + "%"
		args = append(args, like, like, like, like)
	}
	rows, err := s.DB.Query(`SELECT `+postColumns+` FROM posts WHERE `+strings.Join(where, " AND ")+`
		ORDER BY scheduled_at IS NULL, scheduled_at, created_at DESC`, args...)
	if err != nil {
		return nil, err
	}
	out := []models.Post{}
	for rows.Next() {
		p, err := scanPost(rows)
		if err != nil {
			rows.Close()
			return nil, err
		}
		out = append(out, p)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, s.loadPlatforms(s.DB, out)
}

func (s *PostService) getPost(q queryer, id int64) (models.Post, error) {
	p, err := scanPost(q.QueryRow(`SELECT `+postColumns+` FROM posts WHERE id = ?`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return p, ErrNotFound
	}
	if err != nil {
		return p, err
	}
	list := []models.Post{p}
	if err := s.loadPlatforms(q, list); err != nil {
		return p, err
	}
	return list[0], nil
}

func (s *PostService) GetPost(id int64) (models.Post, error) {
	return s.getPost(s.DB, id)
}

func (s *PostService) profilePlatforms(q queryer, profileID int64) (map[string]bool, error) {
	var exists int
	if err := q.QueryRow(`SELECT COUNT(*) FROM profiles WHERE id = ?`, profileID).Scan(&exists); err != nil {
		return nil, err
	}
	if exists == 0 {
		return nil, errors.New("escolha um perfil válido")
	}
	rows, err := q.Query(`SELECT platform FROM profile_platforms WHERE profile_id = ?`, profileID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]bool{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out[id] = true
	}
	return out, rows.Err()
}

// SavePost creates (ID 0) or updates a post. The platforms must belong to the
// post's profile. Publication state of platforms that stay selected is kept.
func (s *PostService) SavePost(p models.Post) (models.Post, error) {
	p.Title = strings.TrimSpace(p.Title)
	if p.Title == "" {
		return p, errors.New("dê um título ao post")
	}
	p.Format = orDefault(p.Format, "short")
	p.Status = orDefault(p.Status, "idea")
	if !catalog.IsFormat(p.Format) {
		return p, fmt.Errorf("formato desconhecido: %s", p.Format)
	}
	if !catalog.IsStatus(p.Status) {
		return p, fmt.Errorf("status desconhecido: %s", p.Status)
	}
	scheduled, err := normalizeSchedule(p.ScheduledAt)
	if err != nil {
		return p, err
	}
	p.ScheduledAt = scheduled

	tx, err := s.DB.Begin()
	if err != nil {
		return p, err
	}
	defer tx.Rollback()

	allowed, err := s.profilePlatforms(tx, p.ProfileID)
	if err != nil {
		return p, err
	}
	selected := map[string]bool{}
	for _, pp := range p.Platforms {
		if !allowed[pp.Platform] {
			name := pp.Platform
			if c, ok := catalog.PlatformByID(pp.Platform); ok {
				name = c.Name
			}
			return p, fmt.Errorf("o perfil não tem %s configurado", name)
		}
		selected[pp.Platform] = true
	}
	if p.ThemeID != nil {
		var owner int64
		err := tx.QueryRow(`SELECT profile_id FROM themes WHERE id = ?`, *p.ThemeID).Scan(&owner)
		if err != nil || owner != p.ProfileID {
			return p, errors.New("o tema escolhido não pertence a este perfil")
		}
	}

	now := nowStamp()
	if p.Status == "published" && p.PublishedAt == nil {
		v := nowSchedule()
		p.PublishedAt = &v
	}
	if p.Status != "published" {
		p.PublishedAt = nil
	}

	if p.ID == 0 {
		res, err := tx.Exec(`INSERT INTO posts (profile_id, theme_id, title, description, script, notes, tags, hashtags,
			format, status, scheduled_at, published_at, postpone_count, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 0, ?, ?)`,
			p.ProfileID, p.ThemeID, p.Title, p.Description, p.Script, p.Notes, encodeList(p.Tags),
			encodeList(p.Hashtags), p.Format, p.Status, p.ScheduledAt, p.PublishedAt, now, now)
		if err != nil {
			return p, err
		}
		p.ID, _ = res.LastInsertId()
	} else {
		res, err := tx.Exec(`UPDATE posts SET profile_id = ?, theme_id = ?, title = ?, description = ?, script = ?,
			notes = ?, tags = ?, hashtags = ?, format = ?, status = ?, scheduled_at = ?, published_at = ?, updated_at = ?
			WHERE id = ?`,
			p.ProfileID, p.ThemeID, p.Title, p.Description, p.Script, p.Notes, encodeList(p.Tags),
			encodeList(p.Hashtags), p.Format, p.Status, p.ScheduledAt, p.PublishedAt, now, p.ID)
		if err != nil {
			return p, err
		}
		if n, _ := res.RowsAffected(); n == 0 {
			return p, ErrNotFound
		}
	}

	rows, err := tx.Query(`SELECT platform FROM post_platforms WHERE post_id = ?`, p.ID)
	if err != nil {
		return p, err
	}
	existing := map[string]bool{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return p, err
		}
		existing[id] = true
	}
	rows.Close()
	for id := range existing {
		if !selected[id] {
			if _, err := tx.Exec(`DELETE FROM post_platforms WHERE post_id = ? AND platform = ?`, p.ID, id); err != nil {
				return p, err
			}
		}
	}
	for id := range selected {
		if !existing[id] {
			if _, err := tx.Exec(`INSERT INTO post_platforms (post_id, platform) VALUES (?, ?)`, p.ID, id); err != nil {
				return p, err
			}
		}
	}
	if err := tx.Commit(); err != nil {
		return p, err
	}
	return s.GetPost(p.ID)
}

func (s *PostService) DeletePost(id int64) error {
	_, err := s.DB.Exec(`DELETE FROM posts WHERE id = ?`, id)
	return err
}

func (s *PostService) DuplicatePost(id int64) (models.Post, error) {
	p, err := s.GetPost(id)
	if err != nil {
		return p, err
	}
	p.ID = 0
	p.Title += " (cópia)"
	p.ScheduledAt = nil
	p.PublishedAt = nil
	if p.Status == "published" {
		p.Status = "ready"
	}
	return s.SavePost(p)
}

func (s *PostService) UpdateStatus(id int64, status string) (models.Post, error) {
	if !catalog.IsStatus(status) {
		return models.Post{}, fmt.Errorf("status desconhecido: %s", status)
	}
	var published any
	if status == "published" {
		published = nowSchedule()
	}
	_, err := s.DB.Exec(`UPDATE posts SET status = ?, published_at = CASE WHEN ? = 'published'
		THEN COALESCE(published_at, ?) ELSE NULL END, updated_at = ? WHERE id = ?`,
		status, status, published, nowStamp(), id)
	if err != nil {
		return models.Post{}, err
	}
	return s.GetPost(id)
}

// SetPlatformStatus records the publication of a post on one platform. When
// every platform is published, the post itself becomes published.
func (s *PostService) SetPlatformStatus(postID int64, platform, status, url string) (models.Post, error) {
	if status != "pending" && status != "published" && status != "skipped" {
		return models.Post{}, fmt.Errorf("status de plataforma desconhecido: %s", status)
	}
	var published any
	if status == "published" {
		published = nowSchedule()
	}
	tx, err := s.DB.Begin()
	if err != nil {
		return models.Post{}, err
	}
	defer tx.Rollback()
	res, err := tx.Exec(`UPDATE post_platforms SET status = ?, url = ?, published_at = CASE WHEN ? = 'published'
		THEN COALESCE(published_at, ?) ELSE NULL END WHERE post_id = ? AND platform = ?`,
		status, strings.TrimSpace(url), status, published, postID, platform)
	if err != nil {
		return models.Post{}, err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return models.Post{}, errors.New("esta plataforma não está selecionada no post")
	}
	var pending, done int
	if err := tx.QueryRow(`SELECT
		COALESCE(SUM(status = 'pending'), 0), COALESCE(SUM(status = 'published'), 0)
		FROM post_platforms WHERE post_id = ?`, postID).Scan(&pending, &done); err != nil {
		return models.Post{}, err
	}
	if pending == 0 && done > 0 {
		if _, err := tx.Exec(`UPDATE posts SET status = 'published', published_at = COALESCE(published_at, ?),
			updated_at = ? WHERE id = ?`, nowSchedule(), nowStamp(), postID); err != nil {
			return models.Post{}, err
		}
	}
	if err := tx.Commit(); err != nil {
		return models.Post{}, err
	}
	return s.GetPost(postID)
}

// Reschedule sets a new date (or clears it with nil), for example after a
// drag on the calendar. Moving a post later counts as a postponement.
func (s *PostService) Reschedule(id int64, scheduledAt *string) (models.Post, error) {
	next, err := normalizeSchedule(scheduledAt)
	if err != nil {
		return models.Post{}, err
	}
	tx, err := s.DB.Begin()
	if err != nil {
		return models.Post{}, err
	}
	defer tx.Rollback()
	p, err := s.getPost(tx, id)
	if err != nil {
		return p, err
	}
	later := p.ScheduledAt != nil && next != nil && *next > *p.ScheduledAt
	if err := s.moveTo(tx, p, next, "", later); err != nil {
		return p, err
	}
	if err := tx.Commit(); err != nil {
		return p, err
	}
	return s.GetPost(id)
}

func (s *PostService) moveTo(tx *sql.Tx, p models.Post, next *string, reason string, record bool) error {
	inc := 0
	if record {
		inc = 1
		if _, err := tx.Exec(`INSERT INTO postponements (post_id, from_at, to_at, reason, created_at) VALUES (?, ?, ?, ?, ?)`,
			p.ID, p.ScheduledAt, next, strings.TrimSpace(reason), nowStamp()); err != nil {
			return err
		}
	}
	_, err := tx.Exec(`UPDATE posts SET scheduled_at = ?, postpone_count = postpone_count + ?, updated_at = ? WHERE id = ?`,
		next, inc, nowStamp(), p.ID)
	return err
}

// Postpone moves a post to a later date. With Cascade, every later unpublished
// post of the same profile moves by the same amount so the queue keeps its
// spacing.
func (s *PostService) Postpone(req models.PostponeRequest) (models.PostponeResult, error) {
	var result models.PostponeResult
	tx, err := s.DB.Begin()
	if err != nil {
		return result, err
	}
	defer tx.Rollback()

	p, err := s.getPost(tx, req.PostID)
	if err != nil {
		return result, err
	}
	if p.Status == "published" {
		return result, errors.New("este post já foi publicado")
	}

	var from time.Time
	if p.ScheduledAt != nil {
		if from, err = parseSchedule(*p.ScheduledAt); err != nil {
			return result, err
		}
	} else if req.Mode != "date" {
		return result, errors.New("agende o post antes de adiá-lo, ou escolha uma data")
	}

	var shift func(time.Time) time.Time
	switch req.Mode {
	case "days":
		if req.Days < 1 {
			return result, errors.New("adie por pelo menos 1 dia")
		}
		shift = func(t time.Time) time.Time { return t.AddDate(0, 0, req.Days) }
	case "date":
		to, err := parseSchedule(req.Date)
		if err != nil {
			return result, errors.New("data inválida")
		}
		if p.ScheduledAt != nil && !to.After(from) {
			return result, errors.New("escolha uma data depois da atual")
		}
		delta := to.Sub(from)
		shift = func(t time.Time) time.Time {
			if p.ScheduledAt == nil {
				return to
			}
			return t.Add(delta)
		}
	case "next_free":
		to, err := s.nextFreeDay(tx, p.ProfileID, p.ID, from)
		if err != nil {
			return result, err
		}
		delta := to.Sub(from)
		shift = func(t time.Time) time.Time { return t.Add(delta) }
	default:
		return result, fmt.Errorf("modo de adiamento desconhecido: %s", req.Mode)
	}

	target := shift(from).Format(ScheduleLayout)
	if err := s.moveTo(tx, p, &target, req.Reason, true); err != nil {
		return result, err
	}
	result.MovedCount = 1

	if req.Cascade && p.ScheduledAt != nil {
		rows, err := tx.Query(`SELECT `+postColumns+` FROM posts WHERE profile_id = ? AND id != ?
			AND status != 'published' AND scheduled_at >= ? ORDER BY scheduled_at`,
			p.ProfileID, p.ID, *p.ScheduledAt)
		if err != nil {
			return result, err
		}
		var later []models.Post
		for rows.Next() {
			lp, err := scanPost(rows)
			if err != nil {
				rows.Close()
				return result, err
			}
			later = append(later, lp)
		}
		rows.Close()
		reason := fmt.Sprintf("Empurrado pelo adiamento de \"%s\"", p.Title)
		for _, lp := range later {
			t, err := parseSchedule(*lp.ScheduledAt)
			if err != nil {
				continue
			}
			next := shift(t).Format(ScheduleLayout)
			if err := s.moveTo(tx, lp, &next, reason, true); err != nil {
				return result, err
			}
			result.MovedCount++
		}
	}

	if err := tx.Commit(); err != nil {
		return result, err
	}
	result.Post, err = s.GetPost(p.ID)
	return result, err
}

func (s *PostService) nextFreeDay(tx *sql.Tx, profileID, postID int64, from time.Time) (time.Time, error) {
	rows, err := tx.Query(`SELECT DISTINCT substr(scheduled_at, 1, 10) FROM posts
		WHERE profile_id = ? AND id != ? AND scheduled_at > ?`, profileID, postID, from.Format(dayLayout))
	if err != nil {
		return from, err
	}
	busy := map[string]bool{}
	for rows.Next() {
		var d string
		if err := rows.Scan(&d); err != nil {
			rows.Close()
			return from, err
		}
		busy[d] = true
	}
	rows.Close()
	for i := 1; i < 3650; i++ {
		d := from.AddDate(0, 0, i)
		if !busy[d.Format(dayLayout)] {
			return d, nil
		}
	}
	return from, errors.New("nenhum dia livre encontrado")
}

func (s *PostService) ListPostponements(postID int64) ([]models.Postponement, error) {
	rows, err := s.DB.Query(`SELECT id, post_id, from_at, to_at, reason, created_at FROM postponements
		WHERE post_id = ? ORDER BY id DESC`, postID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []models.Postponement{}
	for rows.Next() {
		var m models.Postponement
		var from, to sql.NullString
		if err := rows.Scan(&m.ID, &m.PostID, &from, &to, &m.Reason, &m.CreatedAt); err != nil {
			return nil, err
		}
		m.FromAt, m.ToAt = nullString(from), nullString(to)
		out = append(out, m)
	}
	return out, rows.Err()
}

func (s *PostService) GetStats(profileID *int64) (models.Stats, error) {
	st := models.Stats{ByStatus: map[string]int{}}
	for _, o := range catalog.Statuses {
		st.ByStatus[o.ID] = 0
	}
	where, args := "1 = 1", []any{}
	if profileID != nil {
		where, args = "profile_id = ?", []any{*profileID}
	}
	rows, err := s.DB.Query(`SELECT status, COUNT(*) FROM posts WHERE `+where+` GROUP BY status`, args...)
	if err != nil {
		return st, err
	}
	for rows.Next() {
		var status string
		var n int
		if err := rows.Scan(&status, &n); err != nil {
			rows.Close()
			return st, err
		}
		st.ByStatus[status] = n
		st.Total += n
	}
	rows.Close()

	now := time.Now()
	weekday := (int(now.Weekday()) + 6) % 7 // Monday = 0
	monday := now.AddDate(0, 0, -weekday).Format(dayLayout)
	nextMonday := now.AddDate(0, 0, 7-weekday).Format(dayLayout)
	err = s.DB.QueryRow(`SELECT
		COALESCE(SUM(status != 'published' AND scheduled_at < ?), 0),
		COALESCE(SUM(substr(scheduled_at, 1, 10) >= ? AND substr(scheduled_at, 1, 10) < ?), 0),
		COALESCE(SUM(scheduled_at IS NULL AND status != 'published'), 0),
		COALESCE(SUM(postpone_count > 0 AND status != 'published'), 0)
		FROM posts WHERE `+where, append([]any{now.Format(ScheduleLayout), monday, nextMonday}, args...)...).
		Scan(&st.Overdue, &st.ThisWeek, &st.Unscheduled, &st.Postponed)
	return st, err
}
