package services

import (
	"database/sql"
	"encoding/json"
	"errors"
	"strings"
	"time"
)

const ScheduleLayout = "2006-01-02T15:04"

const dayLayout = "2006-01-02"

var ErrNotFound = errors.New("registro não encontrado")

func nowStamp() string { return time.Now().Format(time.RFC3339) }

func nowSchedule() string { return time.Now().Format(ScheduleLayout) }

func parseSchedule(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	if len(s) == len(dayLayout) {
		s += "T09:00"
	}
	if len(s) > len(ScheduleLayout) {
		s = s[:len(ScheduleLayout)]
	}
	return time.ParseInLocation(ScheduleLayout, s, time.Local)
}

func normalizeSchedule(s *string) (*string, error) {
	if s == nil || strings.TrimSpace(*s) == "" {
		return nil, nil
	}
	t, err := parseSchedule(*s)
	if err != nil {
		return nil, errors.New("data inválida: use o formato AAAA-MM-DDTHH:MM")
	}
	v := t.Format(ScheduleLayout)
	return &v, nil
}

func encodeList(list []string) string {
	clean := cleanList(list)
	b, _ := json.Marshal(clean)
	return string(b)
}

func decodeList(s string) []string {
	out := []string{}
	_ = json.Unmarshal([]byte(s), &out)
	return out
}

func cleanList(list []string) []string {
	out := []string{}
	seen := map[string]bool{}
	for _, v := range list {
		v = strings.TrimSpace(v)
		key := strings.ToLower(v)
		if v == "" || seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, v)
	}
	return out
}

func nullString(s sql.NullString) *string {
	if !s.Valid {
		return nil
	}
	v := s.String
	return &v
}

func nullInt(n sql.NullInt64) *int64 {
	if !n.Valid {
		return nil
	}
	v := n.Int64
	return &v
}

func orDefault(v, def string) string {
	if strings.TrimSpace(v) == "" {
		return def
	}
	return strings.TrimSpace(v)
}
