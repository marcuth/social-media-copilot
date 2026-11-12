package services

import (
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/marcuth/social-media-copilot/internal/db"
	"github.com/marcuth/social-media-copilot/internal/models"
)

func setupBackup(t *testing.T) (*BackupService, *ProfileService, *PostService) {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "app.db")
	conn, err := db.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close() })
	return &BackupService{DB: conn, Path: path}, &ProfileService{DB: conn}, &PostService{DB: conn}
}

func TestBackupRoundTrip(t *testing.T) {
	backup, profiles, posts := setupBackup(t)
	p, _ := profiles.SaveProfile(models.Profile{Name: "Canal", Platforms: []models.ProfilePlatform{{Platform: "youtube"}}})
	post := newPost(t, posts, p.ID, "Antes do backup", "2026-11-20T10:00", "youtube")

	file := filepath.Join(t.TempDir(), "copia.db")
	if err := backup.ExportTo(file); err != nil {
		t.Fatal(err)
	}
	info, _ := backup.GetBackupInfo()
	if info.LastBackupAt == "" || info.SizeBytes == 0 {
		t.Fatalf("info: %+v", info)
	}

	// Changes after the backup must disappear on restore.
	if err := posts.DeletePost(post.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := profiles.SaveProfile(models.Profile{Name: "Depois"}); err != nil {
		t.Fatal(err)
	}

	res, err := backup.RestoreFrom(file)
	if err != nil {
		t.Fatal(err)
	}
	if res.Profiles != 1 || res.Posts != 1 {
		t.Fatalf("restore counts: %+v", res)
	}
	if _, err := os.Stat(res.SafetyCopy); err != nil {
		t.Fatalf("safety copy missing: %v", err)
	}
	got, err := posts.GetPost(post.ID)
	if err != nil || got.Title != "Antes do backup" || len(got.Platforms) != 1 {
		t.Fatalf("post after restore: %+v %v", got, err)
	}
	// AUTOINCREMENT keeps counting from the backup's sequence.
	next := newPost(t, posts, p.ID, "Novo", "")
	if next.ID <= post.ID {
		t.Fatalf("new id %d should follow %d", next.ID, post.ID)
	}
	// The safety copy holds the state from just before the restore.
	v, err := db.ReadVersion(res.SafetyCopy)
	if err != nil || v != db.LatestVersion() {
		t.Fatalf("safety copy version %d %v", v, err)
	}
}

func TestRestoreMigratesOldBackup(t *testing.T) {
	backup, _, posts := setupBackup(t)
	old := filepath.Join(t.TempDir(), "v1.db")
	raw, err := sql.Open("sqlite", "file:"+filepath.ToSlash(old))
	if err != nil {
		t.Fatal(err)
	}
	now := "2026-11-12T00:00:00Z"
	for _, q := range []string{
		readMigration(t, 0),
		`PRAGMA user_version = 1`,
		`INSERT INTO profiles (id, name, created_at, updated_at) VALUES (1, 'Antigo', '` + now + `', '` + now + `')`,
		`INSERT INTO profile_platforms VALUES (1, 'youtube_shorts', '@antigo')`,
		`INSERT INTO posts (id, profile_id, title, created_at, updated_at) VALUES (7, 1, 'Short antigo', '` + now + `', '` + now + `')`,
		`INSERT INTO post_platforms (post_id, platform) VALUES (7, 'youtube_shorts')`,
	} {
		if _, err := raw.Exec(q); err != nil {
			t.Fatal(err)
		}
	}
	raw.Close()

	if _, err := backup.RestoreFrom(old); err != nil {
		t.Fatal(err)
	}
	p, err := posts.GetPost(7)
	if err != nil || p.Title != "Short antigo" || len(p.Platforms) != 1 || p.Platforms[0].Platform != "youtube" {
		t.Fatalf("migrated post: %+v %v", p, err)
	}
}

func TestRestoreRejectsBadFiles(t *testing.T) {
	backup, _, _ := setupBackup(t)
	dir := t.TempDir()

	text := filepath.Join(dir, "nota.txt")
	os.WriteFile(text, []byte("não sou um banco"), 0o644)
	if _, err := backup.RestoreFrom(text); err == nil {
		t.Fatal("want error for a non-database file")
	}

	other := filepath.Join(dir, "outro.db")
	raw, _ := sql.Open("sqlite", "file:"+filepath.ToSlash(other))
	raw.Exec(`CREATE TABLE coisas (id INTEGER)`)
	raw.Close()
	if _, err := backup.RestoreFrom(other); err == nil || !strings.Contains(err.Error(), "Social Media Copilot") {
		t.Fatalf("want error for a foreign database, got %v", err)
	}

	newer := filepath.Join(dir, "novo.db")
	if err := backup.ExportTo(newer); err != nil {
		t.Fatal(err)
	}
	raw, _ = sql.Open("sqlite", "file:"+filepath.ToSlash(newer))
	raw.Exec(`PRAGMA user_version = 999`)
	raw.Close()
	if _, err := backup.RestoreFrom(newer); err == nil || !strings.Contains(err.Error(), "mais nova") {
		t.Fatalf("want error for a newer schema, got %v", err)
	}

	if err := backup.ExportTo(backup.Path); err == nil {
		t.Fatal("exporting over the live database must fail")
	}
}

// readMigration is a copy of the first shipped schema, to build a v1 backup.
func readMigration(t *testing.T, i int) string {
	t.Helper()
	if i != 0 {
		t.Fatal("only the first migration is needed")
	}
	return `
	CREATE TABLE profiles (id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT NOT NULL, description TEXT NOT NULL DEFAULT '',
		niche TEXT NOT NULL DEFAULT '', audience TEXT NOT NULL DEFAULT '', tone TEXT NOT NULL DEFAULT '',
		language TEXT NOT NULL DEFAULT 'pt-BR', color TEXT NOT NULL DEFAULT '#3F51D9', created_at TEXT NOT NULL, updated_at TEXT NOT NULL);
	CREATE TABLE profile_platforms (profile_id INTEGER NOT NULL REFERENCES profiles(id) ON DELETE CASCADE, platform TEXT NOT NULL,
		handle TEXT NOT NULL DEFAULT '', PRIMARY KEY (profile_id, platform));
	CREATE TABLE themes (id INTEGER PRIMARY KEY AUTOINCREMENT, profile_id INTEGER NOT NULL REFERENCES profiles(id) ON DELETE CASCADE,
		name TEXT NOT NULL, description TEXT NOT NULL DEFAULT '', color TEXT NOT NULL DEFAULT '#8A88A0', created_at TEXT NOT NULL);
	CREATE TABLE posts (id INTEGER PRIMARY KEY AUTOINCREMENT, profile_id INTEGER NOT NULL REFERENCES profiles(id) ON DELETE CASCADE,
		theme_id INTEGER REFERENCES themes(id) ON DELETE SET NULL, title TEXT NOT NULL DEFAULT '', description TEXT NOT NULL DEFAULT '',
		script TEXT NOT NULL DEFAULT '', notes TEXT NOT NULL DEFAULT '', tags TEXT NOT NULL DEFAULT '[]', hashtags TEXT NOT NULL DEFAULT '[]',
		format TEXT NOT NULL DEFAULT 'short', status TEXT NOT NULL DEFAULT 'idea', scheduled_at TEXT, published_at TEXT,
		postpone_count INTEGER NOT NULL DEFAULT 0, created_at TEXT NOT NULL, updated_at TEXT NOT NULL);
	CREATE TABLE post_platforms (post_id INTEGER NOT NULL REFERENCES posts(id) ON DELETE CASCADE, platform TEXT NOT NULL,
		status TEXT NOT NULL DEFAULT 'pending', url TEXT NOT NULL DEFAULT '', published_at TEXT, PRIMARY KEY (post_id, platform));
	CREATE TABLE postponements (id INTEGER PRIMARY KEY AUTOINCREMENT, post_id INTEGER NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
		from_at TEXT, to_at TEXT, reason TEXT NOT NULL DEFAULT '', created_at TEXT NOT NULL);
	CREATE TABLE ai_providers (id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT NOT NULL, kind TEXT NOT NULL, base_url TEXT NOT NULL DEFAULT '',
		api_key TEXT NOT NULL DEFAULT '', model TEXT NOT NULL, is_default INTEGER NOT NULL DEFAULT 0, created_at TEXT NOT NULL, updated_at TEXT NOT NULL);`
}
