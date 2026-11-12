package services

import (
	"database/sql"
	"encoding/json"
	"errors"
	"strings"

	"github.com/marcuth/social-media-copilot/internal/catalog"
	"github.com/marcuth/social-media-copilot/internal/skills"
)

const disabledSkillsKey = "skills.disabled"

// SkillInfo describes a bundled skill for the settings screen.
type SkillInfo struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Source      string `json:"source"`
	License     string `json:"license"`
	// AppliesTo explains in plain words when the skill is used automatically.
	AppliesTo string `json:"appliesTo"`
	Enabled   bool   `json:"enabled"`
}

// SkillService lists the bundled skills and lets the user turn them off.
type SkillService struct {
	DB *sql.DB
}

func (s *SkillService) disabled() (map[string]bool, error) {
	var raw string
	err := s.DB.QueryRow(`SELECT value FROM app_settings WHERE key = ?`, disabledSkillsKey).Scan(&raw)
	out := map[string]bool{}
	if errors.Is(err, sql.ErrNoRows) {
		return out, nil
	}
	if err != nil {
		return out, err
	}
	var ids []string
	_ = json.Unmarshal([]byte(raw), &ids)
	for _, id := range ids {
		out[id] = true
	}
	return out, nil
}

// ListSkills returns every bundled skill with its on/off state.
func (s *SkillService) ListSkills() ([]SkillInfo, error) {
	disabled, err := s.disabled()
	if err != nil {
		return nil, err
	}
	out := []SkillInfo{}
	for _, sk := range skills.All() {
		out = append(out, SkillInfo{
			ID:          sk.ID,
			Name:        sk.Name,
			Description: sk.Description,
			Source:      sk.Source,
			License:     sk.License,
			AppliesTo:   appliesTo(sk),
			Enabled:     !disabled[sk.ID],
		})
	}
	return out, nil
}

// SetSkillEnabled turns a skill on or off for the AI features.
func (s *SkillService) SetSkillEnabled(id string, enabled bool) error {
	if _, ok := skills.ByID(id); !ok {
		return errors.New("skill desconhecida")
	}
	disabled, err := s.disabled()
	if err != nil {
		return err
	}
	if enabled {
		delete(disabled, id)
	} else {
		disabled[id] = true
	}
	ids := []string{}
	for k := range disabled {
		ids = append(ids, k)
	}
	raw, _ := json.Marshal(ids)
	_, err = s.DB.Exec(`INSERT INTO app_settings (key, value) VALUES (?, ?)
		ON CONFLICT(key) DO UPDATE SET value = excluded.value`, disabledSkillsKey, string(raw))
	return err
}

// GetSkillText returns the full text of a skill, to read it in the app.
func (s *SkillService) GetSkillText(id string) (string, error) {
	return skills.Text(id)
}

var taskLabels = map[skills.Task]string{
	skills.TaskScript:   "roteiros",
	skills.TaskMetadata: "metadados",
	skills.TaskIdeas:    "ideias",
}

func appliesTo(sk skills.Skill) string {
	var tasks []string
	for _, t := range sk.Tasks {
		if l, ok := taskLabels[t]; ok {
			tasks = append(tasks, l)
		}
	}
	if len(tasks) == 0 {
		return "No chat, quando o assunto pedir."
	}
	text := "Em " + strings.Join(tasks, " e ")
	if len(sk.Platforms) > 0 {
		var names []string
		for _, id := range sk.Platforms {
			if p, ok := catalog.PlatformByID(id); ok {
				names = append(names, p.Name)
			}
		}
		text += " para " + strings.Join(names, ", ")
	}
	if len(sk.Formats) > 0 {
		var names []string
		for _, f := range sk.Formats {
			names = append(names, strings.ToLower(catalog.Label(catalog.Formats, f)))
		}
		text += " (" + strings.Join(names, ", ") + ")"
	}
	return text + ", e no chat."
}
