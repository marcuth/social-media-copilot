package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

// migrations are applied in order. The index+1 of each entry is the schema
// version stored in PRAGMA user_version after it runs. Never edit an entry that
// has already shipped: append a new one instead.
var migrations = []string{
	`
	CREATE TABLE profiles (
		id          INTEGER PRIMARY KEY AUTOINCREMENT,
		name        TEXT NOT NULL,
		description TEXT NOT NULL DEFAULT '',
		niche       TEXT NOT NULL DEFAULT '',
		audience    TEXT NOT NULL DEFAULT '',
		tone        TEXT NOT NULL DEFAULT '',
		language    TEXT NOT NULL DEFAULT 'pt-BR',
		color       TEXT NOT NULL DEFAULT '#3F51D9',
		created_at  TEXT NOT NULL,
		updated_at  TEXT NOT NULL
	);

	CREATE TABLE profile_platforms (
		profile_id INTEGER NOT NULL REFERENCES profiles(id) ON DELETE CASCADE,
		platform   TEXT NOT NULL,
		handle     TEXT NOT NULL DEFAULT '',
		PRIMARY KEY (profile_id, platform)
	);

	CREATE TABLE themes (
		id          INTEGER PRIMARY KEY AUTOINCREMENT,
		profile_id  INTEGER NOT NULL REFERENCES profiles(id) ON DELETE CASCADE,
		name        TEXT NOT NULL,
		description TEXT NOT NULL DEFAULT '',
		color       TEXT NOT NULL DEFAULT '#8A88A0',
		created_at  TEXT NOT NULL
	);
	CREATE INDEX idx_themes_profile ON themes(profile_id);

	CREATE TABLE posts (
		id             INTEGER PRIMARY KEY AUTOINCREMENT,
		profile_id     INTEGER NOT NULL REFERENCES profiles(id) ON DELETE CASCADE,
		theme_id       INTEGER REFERENCES themes(id) ON DELETE SET NULL,
		title          TEXT NOT NULL DEFAULT '',
		description    TEXT NOT NULL DEFAULT '',
		script         TEXT NOT NULL DEFAULT '',
		notes          TEXT NOT NULL DEFAULT '',
		tags           TEXT NOT NULL DEFAULT '[]',
		hashtags       TEXT NOT NULL DEFAULT '[]',
		format         TEXT NOT NULL DEFAULT 'short',
		status         TEXT NOT NULL DEFAULT 'idea',
		scheduled_at   TEXT,
		published_at   TEXT,
		postpone_count INTEGER NOT NULL DEFAULT 0,
		created_at     TEXT NOT NULL,
		updated_at     TEXT NOT NULL
	);
	CREATE INDEX idx_posts_profile ON posts(profile_id);
	CREATE INDEX idx_posts_scheduled ON posts(scheduled_at);

	CREATE TABLE post_platforms (
		post_id      INTEGER NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
		platform     TEXT NOT NULL,
		status       TEXT NOT NULL DEFAULT 'pending',
		url          TEXT NOT NULL DEFAULT '',
		published_at TEXT,
		PRIMARY KEY (post_id, platform)
	);

	CREATE TABLE postponements (
		id         INTEGER PRIMARY KEY AUTOINCREMENT,
		post_id    INTEGER NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
		from_at    TEXT,
		to_at      TEXT,
		reason     TEXT NOT NULL DEFAULT '',
		created_at TEXT NOT NULL
	);
	CREATE INDEX idx_postponements_post ON postponements(post_id);

	CREATE TABLE ai_providers (
		id         INTEGER PRIMARY KEY AUTOINCREMENT,
		name       TEXT NOT NULL,
		kind       TEXT NOT NULL,
		base_url   TEXT NOT NULL DEFAULT '',
		api_key    TEXT NOT NULL DEFAULT '',
		model      TEXT NOT NULL,
		is_default INTEGER NOT NULL DEFAULT 0,
		created_at TEXT NOT NULL,
		updated_at TEXT NOT NULL
	);
	`,
	// YouTube Shorts stopped being a separate platform: it is the "short"
	// format published on YouTube.
	`
	INSERT OR IGNORE INTO profile_platforms (profile_id, platform, handle)
		SELECT profile_id, 'youtube', handle FROM profile_platforms WHERE platform = 'youtube_shorts';
	DELETE FROM profile_platforms WHERE platform = 'youtube_shorts';
	INSERT OR IGNORE INTO post_platforms (post_id, platform, status, url, published_at)
		SELECT post_id, 'youtube', status, url, published_at FROM post_platforms WHERE platform = 'youtube_shorts';
	DELETE FROM post_platforms WHERE platform = 'youtube_shorts';
	`,
	// Key/value settings (e.g. disabled skills) and the free chat.
	`
	CREATE TABLE app_settings (
		key   TEXT PRIMARY KEY,
		value TEXT NOT NULL
	);

	CREATE TABLE chat_sessions (
		id         INTEGER PRIMARY KEY AUTOINCREMENT,
		title      TEXT NOT NULL,
		created_at TEXT NOT NULL,
		updated_at TEXT NOT NULL
	);

	CREATE TABLE chat_messages (
		id         INTEGER PRIMARY KEY AUTOINCREMENT,
		session_id INTEGER NOT NULL REFERENCES chat_sessions(id) ON DELETE CASCADE,
		role       TEXT NOT NULL,
		content    TEXT NOT NULL,
		model      TEXT NOT NULL DEFAULT '',
		created_at TEXT NOT NULL
	);
	CREATE INDEX idx_chat_messages_session ON chat_messages(session_id);
	`,
}

func DefaultPath() (string, error) {
	if p := os.Getenv("SMC_DB_PATH"); p != "" {
		return p, nil
	}
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "social-media-copilot", "social-media-copilot.db"), nil
}

func Open(path string) (*sql.DB, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("criar pasta do banco: %w", err)
	}
	dsn := "file:" + filepath.ToSlash(path) +
		"?_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)"
	conn, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	// A single connection keeps SQLite writes serialized and pragmas consistent.
	conn.SetMaxOpenConns(1)
	if err := migrate(conn); err != nil {
		conn.Close()
		return nil, err
	}
	return conn, nil
}

func migrate(conn *sql.DB) error {
	var version int
	if err := conn.QueryRow(`PRAGMA user_version`).Scan(&version); err != nil {
		return fmt.Errorf("ler versão do schema: %w", err)
	}
	for i := version; i < len(migrations); i++ {
		tx, err := conn.Begin()
		if err != nil {
			return err
		}
		if _, err := tx.Exec(migrations[i]); err != nil {
			tx.Rollback()
			return fmt.Errorf("migração %d: %w", i+1, err)
		}
		if _, err := tx.Exec(fmt.Sprintf(`PRAGMA user_version = %d`, i+1)); err != nil {
			tx.Rollback()
			return err
		}
		if err := tx.Commit(); err != nil {
			return err
		}
	}
	return nil
}

// LatestVersion is the schema version this build migrates to.
func LatestVersion() int { return len(migrations) }

// ReadVersion reads the schema version of a database file without migrating
// it, failing if the file isn't an app database.
func ReadVersion(path string) (int, error) {
	conn, err := sql.Open("sqlite", "file:"+filepath.ToSlash(path)+"?mode=ro")
	if err != nil {
		return 0, err
	}
	defer conn.Close()
	var version, tables int
	if err := conn.QueryRow(`PRAGMA user_version`).Scan(&version); err != nil {
		return 0, err
	}
	err = conn.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name IN ('profiles', 'posts')`).Scan(&tables)
	if err != nil {
		return 0, err
	}
	if tables != 2 {
		return 0, fmt.Errorf("o arquivo não é um banco do Social Media Copilot")
	}
	return version, nil
}
