package services

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/marcuth/social-media-copilot/internal/models"
)

// fakeOpenAI streams a fixed answer and records the request bodies.
type fakeOpenAI struct {
	mu        sync.Mutex
	bodies    []map[string]any
	rejectAll bool   // reject requests that carry tools
	callSkill string // first answer is a load_skill call for this id
}

func (f *fakeOpenAI) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	raw, _ := io.ReadAll(r.Body)
	var body map[string]any
	_ = json.Unmarshal(raw, &body)
	f.mu.Lock()
	f.bodies = append(f.bodies, body)
	f.mu.Unlock()
	if _, hasTools := body["tools"]; hasTools && f.rejectAll {
		http.Error(w, `{"error":{"message":"tools not supported"}}`, http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	if f.callSkill != "" && !strings.Contains(string(raw), `"role":"tool"`) {
		args, _ := json.Marshal(map[string]string{"id": f.callSkill})
		fmt.Fprintf(w, "data: {\"id\":\"1\",\"object\":\"chat.completion.chunk\",\"choices\":[{\"index\":0,\"delta\":{\"tool_calls\":[{\"index\":0,\"id\":\"call_1\",\"type\":\"function\",\"function\":{\"name\":\"load_skill\",\"arguments\":%q}}]},\"finish_reason\":null}]}\n\n", string(args))
		fmt.Fprint(w, "data: {\"id\":\"1\",\"object\":\"chat.completion.chunk\",\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":\"tool_calls\"}]}\n\n")
		fmt.Fprint(w, "data: [DONE]\n\n")
		return
	}
	for _, part := range []string{"Olá, ", "criador!"} {
		fmt.Fprintf(w, "data: {\"id\":\"1\",\"object\":\"chat.completion.chunk\",\"choices\":[{\"index\":0,\"delta\":{\"content\":%q},\"finish_reason\":null}]}\n\n", part)
	}
	fmt.Fprint(w, "data: {\"id\":\"1\",\"object\":\"chat.completion.chunk\",\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":\"stop\"}]}\n\n")
	fmt.Fprint(w, "data: [DONE]\n\n")
}

func setupChat(t *testing.T, fake *fakeOpenAI) (*ChatService, models.Profile, *[]AIDelta) {
	t.Helper()
	profiles, posts, profile := setup(t)
	srv := httptest.NewServer(fake)
	t.Cleanup(srv.Close)
	providers := &ProviderService{DB: profiles.DB}
	if _, err := providers.SaveProvider(models.AIProvider{
		Name: "Fake", Kind: KindOpenAICompatible, BaseURL: srv.URL, Model: "fake-model",
	}); err != nil {
		t.Fatal(err)
	}
	var deltas []AIDelta
	var mu sync.Mutex
	ai := &AIService{Profiles: profiles, Providers: providers, Skills: &SkillService{DB: profiles.DB},
		Emit: func(_ string, data any) {
			mu.Lock()
			deltas = append(deltas, data.(AIDelta))
			mu.Unlock()
		}}
	return &ChatService{DB: profiles.DB, AI: ai, Posts: posts}, profile, &deltas
}

func TestChatStreamsAndPersists(t *testing.T) {
	fake := &fakeOpenAI{}
	chat, profile, deltas := setupChat(t, fake)

	reply, err := chat.SendMessage(context.Background(), ChatRequest{
		RequestID: "r1",
		Content:   "Me dá uma ideia de vídeo sobre marmitas para a semana inteira",
		ProfileID: &profile.ID,
		Post:      &models.Post{ProfileID: profile.ID, Title: "Tofu crocante", Format: "short"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if reply.Message.Content != "Olá, criador!" || reply.Message.Model != "Fake · fake-model" {
		t.Fatalf("unexpected reply: %+v", reply.Message)
	}
	// One call, with tools: the retry without tools must not have kicked in.
	if len(fake.bodies) != 1 {
		t.Fatalf("want a single model call, got %d", len(fake.bodies))
	}
	if len(*deltas) != 2 || (*deltas)[0].RequestID != "r1" {
		t.Fatalf("deltas: %+v", *deltas)
	}
	if !strings.HasPrefix(reply.Session.Title, "Me dá uma ideia") {
		t.Fatalf("title: %q", reply.Session.Title)
	}

	// The prompt carries the context, the skill catalog and the tool.
	body := fake.bodies[0]
	raw, _ := json.Marshal(body)
	for _, want := range []string{"Post aberto no editor", "Tofu crocante", "roteiro-video-curto", "load_skill", "Canal"} {
		if !strings.Contains(string(raw), want) {
			t.Errorf("request is missing %q", want)
		}
	}

	// A second message continues the session with the full history.
	if _, err := chat.SendMessage(context.Background(), ChatRequest{SessionID: reply.Session.ID, Content: "E outra?"}); err != nil {
		t.Fatal(err)
	}
	msgs, _ := chat.GetMessages(reply.Session.ID)
	if len(msgs) != 4 || msgs[2].Content != "E outra?" {
		t.Fatalf("messages: %+v", msgs)
	}
	last, _ := json.Marshal(fake.bodies[len(fake.bodies)-1]["messages"])
	if !strings.Contains(string(last), "Olá, criador!") {
		t.Fatal("history not sent to the model")
	}
}

func TestChatRetriesWithoutToolsWhenRejected(t *testing.T) {
	fake := &fakeOpenAI{rejectAll: true}
	chat, _, _ := setupChat(t, fake)
	reply, err := chat.SendMessage(context.Background(), ChatRequest{Content: "oi"})
	if err != nil {
		t.Fatal(err)
	}
	if reply.Message.Content != "Olá, criador!" {
		t.Fatalf("reply: %q", reply.Message.Content)
	}
	if _, hasTools := fake.bodies[len(fake.bodies)-1]["tools"]; hasTools {
		t.Fatal("retry still sent tools")
	}
}

func TestDisabledSkillLeavesChatCatalog(t *testing.T) {
	fake := &fakeOpenAI{}
	chat, _, _ := setupChat(t, fake)
	if err := chat.AI.Skills.SetSkillEnabled("ig-humanizer", false); err != nil {
		t.Fatal(err)
	}
	catalogText, _ := chat.skillTools()
	if strings.Contains(catalogText, "ig-humanizer") || !strings.Contains(catalogText, "ig-caption-writer") {
		t.Fatal("disabled skill should be hidden from the catalog")
	}
	list, _ := chat.AI.Skills.ListSkills()
	for _, s := range list {
		if s.ID == "ig-humanizer" && s.Enabled {
			t.Fatal("ListSkills should report the skill as disabled")
		}
	}
}

func TestChatLoadsSkillThroughTool(t *testing.T) {
	fake := &fakeOpenAI{callSkill: "roteiro-video-curto"}
	chat, _, _ := setupChat(t, fake)
	reply, err := chat.SendMessage(context.Background(), ChatRequest{Content: "escreve um roteiro de short"})
	if err != nil {
		t.Fatal(err)
	}
	if reply.Message.Content != "Olá, criador!" {
		t.Fatalf("reply: %q", reply.Message.Content)
	}
	if len(fake.bodies) != 2 {
		t.Fatalf("want 2 model calls (tool + answer), got %d", len(fake.bodies))
	}
	second, _ := json.Marshal(fake.bodies[1]["messages"])
	if !strings.Contains(string(second), "GPC") {
		t.Fatal("the skill text was not sent back to the model as the tool result")
	}
}

func TestScriptGenerationUsesMatchingSkill(t *testing.T) {
	fake := &fakeOpenAI{}
	chat, profile, _ := setupChat(t, fake)
	_, err := chat.AI.GenerateScript(context.Background(), AIRequest{
		ProfileID: profile.ID, Format: "short", Platforms: []string{"youtube"}, Title: "Tofu crocante",
	})
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(fake.bodies[0])
	if !strings.Contains(string(raw), "Método GPC") {
		t.Fatal("Shorts script should carry the GPC skill")
	}
	if strings.Contains(string(raw), "Jovens de Negócios") {
		t.Fatal("long-video skill must not be used for a short")
	}
	matched, _ := chat.AI.MatchingSkills("script", profile.ID, []string{"youtube"}, "video")
	if len(matched) != 1 || matched[0].ID != "roteiro-video-longo" {
		t.Fatalf("matching skills for long video: %+v", matched)
	}
}
