package skills

import (
	"strings"
	"testing"
)

func TestEveryFileIsEmbedded(t *testing.T) {
	for _, s := range All() {
		if len(s.Files) == 0 {
			t.Errorf("%s has no files", s.ID)
		}
		for _, f := range s.Files {
			if _, err := content.ReadFile(f); err != nil {
				t.Errorf("%s: %v", s.ID, err)
			}
		}
	}
}

func ids(list []Skill) string {
	var out []string
	for _, s := range list {
		out = append(out, s.ID)
	}
	return strings.Join(out, ",")
}

func TestMatch(t *testing.T) {
	cases := []struct {
		task      Task
		platforms []string
		format    string
		want      string
	}{
		{TaskScript, []string{"youtube"}, "video", "roteiro-video-longo"},
		{TaskScript, []string{"youtube", "tiktok"}, "short", "roteiro-video-curto"},
		{TaskScript, []string{"tiktok"}, "short", ""},
		{TaskScript, []string{"instagram"}, "carousel", "ig-carousel-planner"},
		{TaskMetadata, []string{"instagram"}, "short", "ig-caption-writer,ig-hashtag-strategist,ig-humanizer"},
		{TaskMetadata, []string{"youtube"}, "video", ""},
		{TaskIdeas, []string{"instagram", "youtube"}, "", "ig-content-planner"},
	}
	for _, c := range cases {
		if got := ids(Match(c.task, c.platforms, c.format, nil)); got != c.want {
			t.Errorf("%s %v %s: got %q, want %q", c.task, c.platforms, c.format, got, c.want)
		}
	}
	if got := Match(TaskScript, []string{"youtube"}, "video", map[string]bool{"roteiro-video-longo": true}); len(got) != 0 {
		t.Errorf("disabled skill still matched: %s", ids(got))
	}
}

func TestRenderIncludesSharedReferenceOnce(t *testing.T) {
	text, err := Render(Match(TaskMetadata, []string{"instagram"}, "short", nil))
	if err != nil {
		t.Fatal(err)
	}
	if n := strings.Count(text, `caminho="instagram-skills/references/voice-rules.md"`); n != 1 {
		t.Fatalf("voice-rules included %d times", n)
	}
	if !strings.HasPrefix(text, Preamble) {
		t.Fatal("missing preamble")
	}
	chat, _ := Text("roteiro-video-curto")
	if strings.Contains(chat, Preamble) || !strings.Contains(chat, "GPC") {
		t.Fatal("Text should return the skill body without the preamble")
	}
}
