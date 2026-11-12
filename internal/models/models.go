// Package models holds the data types shared by the services and exposed to
// the frontend through the generated Wails bindings.
//
// Dates are local wall-clock strings: "2006-01-02T15:04" for schedule times and
// RFC3339 for audit timestamps (created_at/updated_at).
package models

type ProfilePlatform struct {
	Platform string `json:"platform"`
	Handle   string `json:"handle"`
}

type Profile struct {
	ID          int64             `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Niche       string            `json:"niche"`
	Audience    string            `json:"audience"`
	Tone        string            `json:"tone"`
	Language    string            `json:"language"`
	Color       string            `json:"color"`
	Platforms   []ProfilePlatform `json:"platforms"`
	CreatedAt   string            `json:"createdAt"`
	UpdatedAt   string            `json:"updatedAt"`
}

type Theme struct {
	ID          int64  `json:"id"`
	ProfileID   int64  `json:"profileId"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Color       string `json:"color"`
	CreatedAt   string `json:"createdAt"`
}

type PostPlatform struct {
	Platform    string  `json:"platform"`
	Status      string  `json:"status"` // pending | published | skipped
	URL         string  `json:"url"`
	PublishedAt *string `json:"publishedAt"`
}

type Post struct {
	ID            int64          `json:"id"`
	ProfileID     int64          `json:"profileId"`
	ThemeID       *int64         `json:"themeId"`
	Title         string         `json:"title"`
	Description   string         `json:"description"`
	Script        string         `json:"script"`
	Notes         string         `json:"notes"`
	Tags          []string       `json:"tags"`
	Hashtags      []string       `json:"hashtags"`
	Format        string         `json:"format"`
	Status        string         `json:"status"`
	ScheduledAt   *string        `json:"scheduledAt"`
	PublishedAt   *string        `json:"publishedAt"`
	PostponeCount int            `json:"postponeCount"`
	Platforms     []PostPlatform `json:"platforms"`
	CreatedAt     string         `json:"createdAt"`
	UpdatedAt     string         `json:"updatedAt"`
}

type PostFilter struct {
	ProfileID   *int64 `json:"profileId"`
	ThemeID     *int64 `json:"themeId"`
	Status      string `json:"status"`
	Platform    string `json:"platform"`
	From        string `json:"from"` // inclusive, "2006-01-02"
	To          string `json:"to"`   // inclusive, "2006-01-02"
	Search      string `json:"search"`
	Unscheduled bool   `json:"unscheduled"`
}

type Postponement struct {
	ID        int64   `json:"id"`
	PostID    int64   `json:"postId"`
	FromAt    *string `json:"fromAt"`
	ToAt      *string `json:"toAt"`
	Reason    string  `json:"reason"`
	CreatedAt string  `json:"createdAt"`
}

type PostponeRequest struct {
	PostID int64 `json:"postId"`
	// Mode is one of: "days" (shift by Days), "date" (move to Date),
	// "next_free" (next day after the current date without posts of the profile).
	Mode   string `json:"mode"`
	Days   int    `json:"days"`
	Date   string `json:"date"` // "2006-01-02T15:04"
	Reason string `json:"reason"`
	// Cascade shifts every later, unpublished post of the same profile by the
	// same amount, so the queue keeps its spacing.
	Cascade bool `json:"cascade"`
}

type PostponeResult struct {
	Post       Post `json:"post"`
	MovedCount int  `json:"movedCount"` // posts moved, including the target
}

type AIProvider struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	Kind      string `json:"kind"` // openai_compatible | anthropic | google | ollama
	BaseURL   string `json:"baseUrl"`
	APIKey    string `json:"apiKey"`
	Model     string `json:"model"`
	IsDefault bool   `json:"isDefault"`
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
}

type Stats struct {
	Total       int            `json:"total"`
	ByStatus    map[string]int `json:"byStatus"`
	Overdue     int            `json:"overdue"`
	ThisWeek    int            `json:"thisWeek"`
	Unscheduled int            `json:"unscheduled"`
	Postponed   int            `json:"postponed"`
}
