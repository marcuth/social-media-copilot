package db

import (
	"database/sql"
	"path/filepath"
	"testing"
)

func TestShortsMigrationMergesIntoYouTube(t *testing.T) {
	path := filepath.Join(t.TempDir(), "v1.db")
	raw, err := sql.Open("sqlite", "file:"+filepath.ToSlash(path))
	if err != nil {
		t.Fatal(err)
	}
	raw.SetMaxOpenConns(1)
	now := "2026-11-12T00:00:00Z"
	for _, q := range []string{
		migrations[0],
		`PRAGMA user_version = 1`,
		`INSERT INTO profiles (id, name, created_at, updated_at) VALUES (1, 'a', '` + now + `', '` + now + `'), (2, 'b', '` + now + `', '` + now + `')`,
		// Profile 1 had both; profile 2 only Shorts.
		`INSERT INTO profile_platforms VALUES (1, 'youtube', '@long'), (1, 'youtube_shorts', '@short'), (2, 'youtube_shorts', '@b')`,
		`INSERT INTO posts (id, profile_id, title, created_at, updated_at) VALUES (1, 2, 'p', '` + now + `', '` + now + `')`,
		`INSERT INTO post_platforms (post_id, platform, status, url) VALUES (1, 'youtube_shorts', 'published', 'https://yt/s')`,
	} {
		if _, err := raw.Exec(q); err != nil {
			t.Fatal(err)
		}
	}
	raw.Close()

	conn, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	var n int
	conn.QueryRow(`SELECT COUNT(*) FROM profile_platforms WHERE platform = 'youtube_shorts'`).Scan(&n)
	if n != 0 {
		t.Fatalf("shorts rows left: %d", n)
	}
	var handle string
	conn.QueryRow(`SELECT handle FROM profile_platforms WHERE profile_id = 1 AND platform = 'youtube'`).Scan(&handle)
	if handle != "@long" {
		t.Fatalf("existing YouTube handle overwritten: %q", handle)
	}
	conn.QueryRow(`SELECT handle FROM profile_platforms WHERE profile_id = 2 AND platform = 'youtube'`).Scan(&handle)
	if handle != "@b" {
		t.Fatalf("shorts-only profile not converted: %q", handle)
	}
	var status, url string
	conn.QueryRow(`SELECT status, url FROM post_platforms WHERE post_id = 1 AND platform = 'youtube'`).Scan(&status, &url)
	if status != "published" || url != "https://yt/s" {
		t.Fatalf("post platform not converted: %s %s", status, url)
	}
}
