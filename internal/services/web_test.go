package services

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/marcuth/social-media-copilot/internal/models"
)

type fakeFirecrawl struct {
	mu    sync.Mutex
	paths []string
}

func (f *fakeFirecrawl) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	f.paths = append(f.paths, r.URL.Path)
	f.mu.Unlock()
	var body map[string]any
	_ = json.NewDecoder(r.Body).Decode(&body)
	switch r.URL.Path {
	case "/v2/search":
		w.Write([]byte(`{"success":true,"data":{"web":[{"url":"https://receitas.com/tofu","title":"Tofu crocante no forno",
			"description":"Amido de milho deixa crocante","markdown":"Asse a 220 graus por 25 minutos."}]}}`))
	case "/v2/scrape":
		w.Write([]byte(`{"success":true,"data":{"markdown":"# Artigo\nConteúdo do artigo.","metadata":{"title":"Artigo","sourceURL":"` + body["url"].(string) + `"}}}`))
	default:
		http.NotFound(w, r)
	}
}

func withFirecrawl(t *testing.T, chat *ChatService) *fakeFirecrawl {
	t.Helper()
	fake := &fakeFirecrawl{}
	srv := httptest.NewServer(fake)
	t.Cleanup(srv.Close)
	web := &WebService{DB: chat.DB, Providers: chat.AI.Providers, Emit: chat.AI.Emit}
	if err := web.SaveWebSettings(WebSettings{FirecrawlBaseURL: srv.URL, NativeSearch: true}); err != nil {
		t.Fatal(err)
	}
	chat.AI.Web = web
	return fake
}

func TestWebSettingsRoundTrip(t *testing.T) {
	chat, _, _ := setupChat(t, &fakeOpenAI{})
	web := &WebService{DB: chat.DB, Providers: chat.AI.Providers}
	s, err := web.GetWebSettings()
	if err != nil || !s.NativeSearch || s.FirecrawlAPIKey != "" {
		t.Fatalf("defaults: %+v %v", s, err)
	}
	if err := web.SaveWebSettings(WebSettings{FirecrawlBaseURL: "ftp://x"}); err == nil {
		t.Fatal("want error for a non-http URL")
	}
	if err := web.SaveWebSettings(WebSettings{FirecrawlAPIKey: " fc-1 ", FirecrawlBaseURL: "http://localhost:3002/", NativeSearch: false}); err != nil {
		t.Fatal(err)
	}
	s, _ = web.GetWebSettings()
	if s.FirecrawlAPIKey != "fc-1" || s.FirecrawlBaseURL != "http://localhost:3002" || s.NativeSearch {
		t.Fatalf("saved: %+v", s)
	}
}

func TestWebUnavailableWithoutConfig(t *testing.T) {
	chat, _, _ := setupChat(t, &fakeOpenAI{})
	web := &WebService{DB: chat.DB, Providers: chat.AI.Providers}
	if st := web.GetWebStatus(); st.SearchVia != "" || st.FetchVia != "" {
		t.Fatalf("status: %+v", st)
	}
	if _, err := web.Search(context.Background(), "x", SearchOpts{}); err == nil || !strings.Contains(err.Error(), "Firecrawl") {
		t.Fatalf("err: %v", err)
	}
	if tools := web.chatTools("r", ""); len(tools) != 0 {
		t.Fatal("no web tools without a backend")
	}
}

func TestFirecrawlSearchAndFetch(t *testing.T) {
	chat, _, _ := setupChat(t, &fakeOpenAI{})
	withFirecrawl(t, chat)
	web := chat.AI.Web
	if st := web.GetWebStatus(); st.SearchVia != "Firecrawl" || st.FetchVia != "Firecrawl" {
		t.Fatalf("status: %+v", st)
	}
	text, err := web.Search(context.Background(), "tofu crocante", SearchOpts{Country: "BR"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(text, "Fonte: https://receitas.com/tofu") || !strings.Contains(text, "220 graus") {
		t.Fatalf("search text: %s", text)
	}
	page, err := web.Fetch(context.Background(), "https://blog.com/post", "")
	if err != nil || !strings.Contains(page, "Conteúdo do artigo.") || !strings.Contains(page, "Fonte: https://blog.com/post") {
		t.Fatalf("page: %q %v", page, err)
	}
	if _, err := web.Fetch(context.Background(), "blog.com/sem-esquema", ""); err == nil {
		t.Fatal("want error for a URL without scheme")
	}
}

func TestNativeToolsPerProvider(t *testing.T) {
	cases := []struct {
		p           models.AIProvider
		label       string
		search, get bool
	}{
		{models.AIProvider{Kind: KindAnthropic, APIKey: "k", Model: "m"}, "Anthropic", true, true},
		{models.AIProvider{Kind: KindGoogle, APIKey: "k", Model: "m"}, "Google", true, true},
		{models.AIProvider{Kind: KindOpenAICompatible, BaseURL: "https://api.openai.com/v1", APIKey: "k", Model: "m"}, "OpenAI", true, false},
		{models.AIProvider{Kind: KindOpenAICompatible, BaseURL: "https://api.deepseek.com/v1", Model: "m"}, "", false, false},
		{models.AIProvider{Kind: KindOllama, Model: "m"}, "", false, false},
	}
	for _, c := range cases {
		kit := native(c.p)
		if c.label == "" {
			if kit != nil {
				t.Errorf("%s %s: want no native tools", c.p.Kind, c.p.BaseURL)
			}
			continue
		}
		if kit == nil || kit.label != c.label || (kit.search != nil) != c.search || (kit.fetch != nil) != c.get {
			t.Errorf("%s: got %+v", c.label, kit)
		}
	}
}

func TestScriptResearchUsesWebResults(t *testing.T) {
	fake := &fakeOpenAI{}
	chat, profile, _ := setupChat(t, fake)
	fc := withFirecrawl(t, chat)
	var statuses []string
	chat.AI.Web.Emit = func(name string, data any) {
		if st, ok := data.(AIStatus); ok && name == EventAIStatus {
			statuses = append(statuses, st.Text)
		}
	}
	_, err := chat.AI.GenerateScript(context.Background(), AIRequest{
		RequestID: "r1", ProfileID: profile.ID, Format: "short", Platforms: []string{"youtube"},
		Title: "Tofu crocante", Research: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(fake.bodies[0])
	if !strings.Contains(string(raw), "Pesquisa na web") || !strings.Contains(string(raw), "receitas.com/tofu") {
		t.Fatal("script prompt should carry the search results")
	}
	if len(fc.paths) != 1 || fc.paths[0] != "/v2/search" {
		t.Fatalf("firecrawl calls: %v", fc.paths)
	}
	if len(statuses) == 0 || !strings.HasPrefix(statuses[0], "Pesquisando na web") {
		t.Fatalf("statuses: %v", statuses)
	}
}

func TestChatCallsWebSearchTool(t *testing.T) {
	fake := &fakeOpenAI{callTool: "web_search", callArgs: map[string]string{"query": "tofu crocante"}}
	chat, _, _ := setupChat(t, fake)
	withFirecrawl(t, chat)
	if _, err := chat.SendMessage(context.Background(), ChatRequest{RequestID: "r1", Content: "como deixar tofu crocante?"}); err != nil {
		t.Fatal(err)
	}
	first, _ := json.Marshal(fake.bodies[0])
	for _, want := range []string{"web_search", "fetch_url", "load_skill"} {
		if !strings.Contains(string(first), want) {
			t.Errorf("tool %s not offered", want)
		}
	}
	second, _ := json.Marshal(fake.bodies[1]["messages"])
	if !strings.Contains(string(second), "receitas.com/tofu") {
		t.Fatal("search results were not sent back to the model")
	}
}

// fakeAnthropic answers the Messages API; failing=true returns 500.
type fakeAnthropic struct {
	failing bool
	bodies  []string
}

func (f *fakeAnthropic) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var raw json.RawMessage
	_ = json.NewDecoder(r.Body).Decode(&raw)
	f.bodies = append(f.bodies, r.URL.Path+" "+string(raw))
	if f.failing {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"type":"error","error":{"type":"api_error","message":"boom"}}`))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"id":"msg_1","type":"message","role":"assistant","model":"m","content":[{"type":"text","text":"Tofu fica crocante com amido (https://nativo.com)."}],"stop_reason":"end_turn","usage":{"input_tokens":1,"output_tokens":1}}`))
}

func useAnthropic(t *testing.T, chat *ChatService, fake *fakeAnthropic) {
	t.Helper()
	srv := httptest.NewServer(fake)
	t.Cleanup(srv.Close)
	p, _ := chat.AI.Providers.defaultProvider()
	p.Kind, p.BaseURL, p.APIKey, p.Model = KindAnthropic, srv.URL, "sk-test", "claude-test"
	if _, err := chat.AI.Providers.SaveProvider(p); err != nil {
		t.Fatal(err)
	}
}

func TestNativeSearchPreferredOverFirecrawl(t *testing.T) {
	chat, _, _ := setupChat(t, &fakeOpenAI{})
	fc := withFirecrawl(t, chat)
	anth := &fakeAnthropic{}
	useAnthropic(t, chat, anth)
	if st := chat.AI.Web.GetWebStatus(); st.SearchVia != "busca nativa (Anthropic)" || st.FetchVia != "Firecrawl" {
		t.Fatalf("status: %+v", st)
	}
	text, err := chat.AI.Web.Search(context.Background(), "tofu", SearchOpts{})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(text, "nativo.com") || len(fc.paths) != 0 {
		t.Fatalf("want native answer without Firecrawl, got %q (firecrawl %v)", text, fc.paths)
	}
	if len(anth.bodies) != 1 || !strings.Contains(anth.bodies[0], "web_search") {
		t.Fatalf("native request should carry the web_search tool: %v", anth.bodies)
	}
}

func TestFirecrawlTakesOverWhenNativeFails(t *testing.T) {
	chat, _, _ := setupChat(t, &fakeOpenAI{})
	fc := withFirecrawl(t, chat)
	useAnthropic(t, chat, &fakeAnthropic{failing: true})
	text, err := chat.AI.Web.Search(context.Background(), "tofu", SearchOpts{})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(text, "receitas.com/tofu") || len(fc.paths) != 1 {
		t.Fatalf("want Firecrawl fallback, got %q", text)
	}
}
