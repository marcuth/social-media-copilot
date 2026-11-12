package services

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/marcuth/social-media-copilot/internal/db"
	"github.com/marcuth/social-media-copilot/internal/models"
)

func setup(t *testing.T) (*ProfileService, *PostService, models.Profile) {
	t.Helper()
	conn, err := db.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close() })
	profiles := &ProfileService{DB: conn}
	posts := &PostService{DB: conn}
	p, err := profiles.SaveProfile(models.Profile{
		Name: "Canal",
		Platforms: []models.ProfilePlatform{
			{Platform: "youtube", Handle: "@canal"},
			{Platform: "tiktok"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	return profiles, posts, p
}

func str(s string) *string { return &s }

func newPost(t *testing.T, s *PostService, profileID int64, title, at string, platforms ...string) models.Post {
	t.Helper()
	p := models.Post{ProfileID: profileID, Title: title}
	if at != "" {
		p.ScheduledAt = str(at)
	}
	for _, id := range platforms {
		p.Platforms = append(p.Platforms, models.PostPlatform{Platform: id})
	}
	saved, err := s.SavePost(p)
	if err != nil {
		t.Fatal(err)
	}
	return saved
}

func TestSavePostRejectsPlatformOutsideProfile(t *testing.T) {
	_, posts, profile := setup(t)
	_, err := posts.SavePost(models.Post{
		ProfileID: profile.ID,
		Title:     "x",
		Platforms: []models.PostPlatform{{Platform: "instagram"}},
	})
	if err == nil || !strings.Contains(err.Error(), "Instagram") {
		t.Fatalf("want error about Instagram, got %v", err)
	}
}

func TestSavePostKeepsPlatformStateAndTags(t *testing.T) {
	_, posts, profile := setup(t)
	p := newPost(t, posts, profile.ID, "Post", "2026-11-20T10:00", "youtube", "tiktok")
	if _, err := posts.SetPlatformStatus(p.ID, "youtube", "published", "https://y.t/1"); err != nil {
		t.Fatal(err)
	}
	p, _ = posts.GetPost(p.ID)
	p.Tags = []string{" go ", "Go", "", "vue"}
	p.Platforms = []models.PostPlatform{{Platform: "youtube"}}
	p, err := posts.SavePost(p)
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Platforms) != 1 || p.Platforms[0].Status != "published" || p.Platforms[0].URL != "https://y.t/1" {
		t.Fatalf("platform state lost: %+v", p.Platforms)
	}
	if strings.Join(p.Tags, ",") != "go,vue" {
		t.Fatalf("tags not cleaned: %v", p.Tags)
	}
}

func TestPlatformStatusPublishesPost(t *testing.T) {
	_, posts, profile := setup(t)
	p := newPost(t, posts, profile.ID, "Post", "2026-11-20T10:00", "youtube", "tiktok")
	p, _ = posts.SetPlatformStatus(p.ID, "youtube", "published", "")
	if p.Status == "published" {
		t.Fatal("published too early")
	}
	p, _ = posts.SetPlatformStatus(p.ID, "tiktok", "skipped", "")
	if p.Status != "published" || p.PublishedAt == nil {
		t.Fatalf("want published, got %s", p.Status)
	}
}

func TestPostponeByDays(t *testing.T) {
	_, posts, profile := setup(t)
	p := newPost(t, posts, profile.ID, "A", "2026-11-20T10:00")
	res, err := posts.Postpone(models.PostponeRequest{PostID: p.ID, Mode: "days", Days: 3, Reason: "gripe"})
	if err != nil {
		t.Fatal(err)
	}
	if *res.Post.ScheduledAt != "2026-11-23T10:00" || res.Post.PostponeCount != 1 || res.MovedCount != 1 {
		t.Fatalf("unexpected result: %+v", res)
	}
	hist, _ := posts.ListPostponements(p.ID)
	if len(hist) != 1 || *hist[0].FromAt != "2026-11-20T10:00" || hist[0].Reason != "gripe" {
		t.Fatalf("history: %+v", hist)
	}
}

func TestPostponeCascadeShiftsLaterPostsOnly(t *testing.T) {
	profiles, posts, profile := setup(t)
	before := newPost(t, posts, profile.ID, "antes", "2026-11-19T10:00")
	target := newPost(t, posts, profile.ID, "alvo", "2026-11-20T10:00")
	sameDay := newPost(t, posts, profile.ID, "mesmo dia", "2026-11-20T18:00")
	after := newPost(t, posts, profile.ID, "depois", "2026-11-22T09:00")
	done := newPost(t, posts, profile.ID, "publicado", "2026-11-25T09:00")
	if _, err := posts.UpdateStatus(done.ID, "published"); err != nil {
		t.Fatal(err)
	}
	other, _ := profiles.SaveProfile(models.Profile{Name: "Outro"})
	otherPost := newPost(t, posts, other.ID, "outro perfil", "2026-11-21T10:00")

	res, err := posts.Postpone(models.PostponeRequest{PostID: target.ID, Mode: "days", Days: 2, Cascade: true})
	if err != nil {
		t.Fatal(err)
	}
	if res.MovedCount != 3 {
		t.Fatalf("moved %d, want 3", res.MovedCount)
	}
	want := map[int64]string{
		before.ID:    "2026-11-19T10:00",
		target.ID:    "2026-11-22T10:00",
		sameDay.ID:   "2026-11-22T18:00",
		after.ID:     "2026-11-24T09:00",
		done.ID:      "2026-11-25T09:00",
		otherPost.ID: "2026-11-21T10:00",
	}
	for id, at := range want {
		p, _ := posts.GetPost(id)
		if *p.ScheduledAt != at {
			t.Errorf("post %q at %s, want %s", p.Title, *p.ScheduledAt, at)
		}
	}
}

func TestPostponeNextFreeDay(t *testing.T) {
	_, posts, profile := setup(t)
	target := newPost(t, posts, profile.ID, "alvo", "2026-11-20T10:00")
	newPost(t, posts, profile.ID, "ocupa 21", "2026-11-21T08:00")
	newPost(t, posts, profile.ID, "ocupa 22", "2026-11-22T20:00")
	res, err := posts.Postpone(models.PostponeRequest{PostID: target.ID, Mode: "next_free"})
	if err != nil {
		t.Fatal(err)
	}
	if *res.Post.ScheduledAt != "2026-11-23T10:00" {
		t.Fatalf("got %s", *res.Post.ScheduledAt)
	}
}

func TestPostponeDateRequiresLaterDate(t *testing.T) {
	_, posts, profile := setup(t)
	p := newPost(t, posts, profile.ID, "A", "2026-11-20T10:00")
	if _, err := posts.Postpone(models.PostponeRequest{PostID: p.ID, Mode: "date", Date: "2026-11-19T10:00"}); err == nil {
		t.Fatal("want error for earlier date")
	}
	unscheduled := newPost(t, posts, profile.ID, "B", "")
	if _, err := posts.Postpone(models.PostponeRequest{PostID: unscheduled.ID, Mode: "days", Days: 1}); err == nil {
		t.Fatal("want error postponing unscheduled post by days")
	}
}

func TestRescheduleLaterCountsAsPostponement(t *testing.T) {
	_, posts, profile := setup(t)
	p := newPost(t, posts, profile.ID, "A", "2026-11-20T10:00")
	p, _ = posts.Reschedule(p.ID, str("2026-11-18T10:00"))
	if p.PostponeCount != 0 {
		t.Fatal("moving earlier must not count")
	}
	p, _ = posts.Reschedule(p.ID, str("2026-11-21T10:00"))
	if p.PostponeCount != 1 {
		t.Fatal("moving later must count")
	}
}

func TestListPostsFilters(t *testing.T) {
	_, posts, profile := setup(t)
	newPost(t, posts, profile.ID, "Vídeo sobre Go", "2026-11-20T10:00", "youtube")
	newPost(t, posts, profile.ID, "Dança", "2026-12-01T10:00", "tiktok")
	newPost(t, posts, profile.ID, "Rascunho", "")

	check := func(f models.PostFilter, want int) {
		t.Helper()
		list, err := posts.ListPosts(f)
		if err != nil {
			t.Fatal(err)
		}
		if len(list) != want {
			t.Errorf("filter %+v: got %d, want %d", f, len(list), want)
		}
	}
	check(models.PostFilter{}, 3)
	check(models.PostFilter{From: "2026-11-01", To: "2026-11-30"}, 1)
	check(models.PostFilter{Platform: "tiktok"}, 1)
	check(models.PostFilter{Unscheduled: true}, 1)
	check(models.PostFilter{Search: "go"}, 1)
}
