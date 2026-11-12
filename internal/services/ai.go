package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/zendev-sh/goai"

	"github.com/marcuth/social-media-copilot/internal/catalog"
)

// EventAIDelta carries streamed text chunks of a generation to the frontend.
const EventAIDelta = "ai:delta"

// AIDelta is one streamed chunk, routed by the request id chosen by the caller.
type AIDelta struct {
	RequestID string `json:"requestId"`
	Delta     string `json:"delta"`
}

// AIRequest is the context of a post the AI should work on.
type AIRequest struct {
	RequestID       string   `json:"requestId"`
	ProfileID       int64    `json:"profileId"`
	ThemeID         *int64   `json:"themeId"`
	Format          string   `json:"format"`
	Platforms       []string `json:"platforms"`
	Title           string   `json:"title"`
	Description     string   `json:"description"`
	Script          string   `json:"script"`
	Notes           string   `json:"notes"`
	DurationSeconds int      `json:"durationSeconds"`
	Instructions    string   `json:"instructions"`
}

// Metadata is the publishing metadata suggested for a post.
type Metadata struct {
	Title       string   `json:"title" jsonschema:"description=Título do conteúdo"`
	Description string   `json:"description" jsonschema:"description=Descrição ou legenda completa"`
	Tags        []string `json:"tags" jsonschema:"description=Palavras-chave sem o símbolo #"`
	Hashtags    []string `json:"hashtags" jsonschema:"description=Hashtags começando com #"`
}

// IdeasRequest asks for new content ideas for a profile.
type IdeasRequest struct {
	ProfileID    int64    `json:"profileId"`
	ThemeID      *int64   `json:"themeId"`
	Count        int      `json:"count"`
	Format       string   `json:"format"`
	Platforms    []string `json:"platforms"`
	Instructions string   `json:"instructions"`
}

// Idea is one content suggestion.
type Idea struct {
	Title   string `json:"title" jsonschema:"description=Título provisório"`
	Hook    string `json:"hook" jsonschema:"description=Gancho dos primeiros segundos ou da primeira linha"`
	Summary string `json:"summary" jsonschema:"description=Resumo do conteúdo em 1 ou 2 frases"`
	Format  string `json:"format" jsonschema:"enum=short|video|live|carousel|image|story|text"`
}

type ideasEnvelope struct {
	Ideas []Idea `json:"ideas"`
}

// AIService generates scripts, metadata and ideas with the default provider.
type AIService struct {
	Profiles  *ProfileService
	Providers *ProviderService
	// Emit sends an event to the frontend; set by main once the app exists.
	Emit func(name string, data any)
}

const systemBase = `Você é o copiloto de um criador de conteúdo para redes sociais. Você escreve roteiros e metadados prontos para usar, específicos para o perfil e as plataformas indicadas, sem frases genéricas nem clichês de marketing. Responda sempre no idioma do perfil.`

func (s *AIService) contextBlock(profileID int64, themeID *int64, format string, platforms []string) (string, string, error) {
	profile, err := s.Profiles.GetProfile(profileID)
	if err != nil {
		return "", "", errors.New("escolha um perfil para a IA ter contexto")
	}
	var b strings.Builder
	fmt.Fprintf(&b, "## Perfil\nNome: %s\nIdioma: %s\n", profile.Name, profile.Language)
	writeIf(&b, "Descrição", profile.Description)
	writeIf(&b, "Nicho", profile.Niche)
	writeIf(&b, "Público", profile.Audience)
	writeIf(&b, "Tom de voz", profile.Tone)

	if themeID != nil {
		themes, _ := s.Profiles.ListThemes(profileID)
		for _, t := range themes {
			if t.ID == *themeID {
				fmt.Fprintf(&b, "\n## Tema\n%s\n", t.Name)
				if t.Description != "" {
					b.WriteString(t.Description + "\n")
				}
			}
		}
	}
	if format != "" {
		fmt.Fprintf(&b, "\n## Formato\n%s\n", catalog.Label(catalog.Formats, format))
	}
	if len(platforms) == 0 {
		for _, pp := range profile.Platforms {
			platforms = append(platforms, pp.Platform)
		}
	}
	if len(platforms) > 0 {
		b.WriteString("\n## Plataformas de publicação\n")
		for _, id := range platforms {
			if p, ok := catalog.PlatformByID(id); ok {
				fmt.Fprintf(&b, "- %s: %s\n", p.Name, p.Guidance)
			}
		}
	}
	return b.String(), profile.Language, nil
}

func writeIf(b *strings.Builder, label, value string) {
	if v := strings.TrimSpace(value); v != "" {
		fmt.Fprintf(b, "%s: %s\n", label, v)
	}
}

func postBlock(r AIRequest) string {
	var b strings.Builder
	b.WriteString("\n## Conteúdo\n")
	writeIf(&b, "Título/ideia", r.Title)
	writeIf(&b, "Descrição atual", r.Description)
	writeIf(&b, "Anotações", r.Notes)
	if strings.TrimSpace(r.Script) != "" {
		b.WriteString("Roteiro atual:\n" + r.Script + "\n")
	}
	return b.String()
}

func (s *AIService) model() (goaiModel, error) {
	p, err := s.Providers.defaultProvider()
	if err != nil {
		return nil, err
	}
	return buildModel(p)
}

// GenerateScript writes a script for the post, streaming chunks through the
// "ai:delta" event, and returns the full text. Cancelling the call from the
// frontend stops the generation.
func (s *AIService) GenerateScript(ctx context.Context, r AIRequest) (string, error) {
	model, err := s.model()
	if err != nil {
		return "", err
	}
	ctxText, _, err := s.contextBlock(r.ProfileID, r.ThemeID, r.Format, r.Platforms)
	if err != nil {
		return "", err
	}
	var task strings.Builder
	task.WriteString(ctxText)
	task.WriteString(postBlock(r))
	task.WriteString("\n## Tarefa\n")
	if strings.TrimSpace(r.Script) != "" {
		task.WriteString("Reescreva e melhore o roteiro atual mantendo a ideia central.\n")
	} else {
		task.WriteString("Escreva o roteiro deste conteúdo.\n")
	}
	if r.DurationSeconds > 0 {
		fmt.Fprintf(&task, "Duração alvo: cerca de %d segundos de fala.\n", r.DurationSeconds)
	}
	task.WriteString(`Estruture em Markdown com: **Gancho** (primeiros 3 segundos), **Desenvolvimento** em blocos curtos com indicações de cena/corte entre colchetes, e **Chamada para ação** no final. Escreva as falas como serão ditas, em frases curtas. Não inclua título, hashtags nem explicações fora do roteiro.`)
	if v := strings.TrimSpace(r.Instructions); v != "" {
		task.WriteString("\nInstruções extras do criador: " + v)
	}

	stream, err := goai.StreamText(ctx, model,
		goai.WithSystem(systemBase),
		goai.WithPrompt(task.String()),
		goai.WithMaxOutputTokens(8000),
		goai.WithTimeout(5*time.Minute),
	)
	if err != nil {
		return "", friendlyAIError(err)
	}
	var full strings.Builder
	for chunk := range stream.TextStream() {
		full.WriteString(chunk)
		if s.Emit != nil && r.RequestID != "" {
			s.Emit(EventAIDelta, AIDelta{RequestID: r.RequestID, Delta: chunk})
		}
	}
	if err := stream.Err(); err != nil {
		return full.String(), friendlyAIError(err)
	}
	return strings.TrimSpace(full.String()), nil
}

// GenerateMetadata suggests title, description, tags and hashtags.
func (s *AIService) GenerateMetadata(ctx context.Context, r AIRequest) (Metadata, error) {
	var out Metadata
	model, err := s.model()
	if err != nil {
		return out, err
	}
	ctxText, _, err := s.contextBlock(r.ProfileID, r.ThemeID, r.Format, r.Platforms)
	if err != nil {
		return out, err
	}
	prompt := ctxText + postBlock(r) + `
## Tarefa
Crie os metadados de publicação deste conteúdo, respeitando as orientações de cada plataforma listada (quando houver várias, use a mais restritiva para o título e escreva uma descrição que funcione em todas).
- title: título chamativo e específico, sem aspas.
- description: descrição/legenda completa, com gancho na primeira linha e chamada para ação.
- tags: 8 a 15 palavras-chave de busca, sem #.
- hashtags: 3 a 10 hashtags começando com #, sem espaços.`
	if v := strings.TrimSpace(r.Instructions); v != "" {
		prompt += "\nInstruções extras do criador: " + v
	}

	err = generateJSON(ctx, model, prompt, &out)
	if err != nil {
		return out, err
	}
	out.Title = strings.Trim(strings.TrimSpace(out.Title), `"`)
	out.Tags = cleanList(stripHash(out.Tags))
	out.Hashtags = cleanList(addHash(out.Hashtags))
	return out, nil
}

// GenerateIdeas suggests new content ideas for a profile.
func (s *AIService) GenerateIdeas(ctx context.Context, r IdeasRequest) ([]Idea, error) {
	model, err := s.model()
	if err != nil {
		return nil, err
	}
	ctxText, _, err := s.contextBlock(r.ProfileID, r.ThemeID, r.Format, r.Platforms)
	if err != nil {
		return nil, err
	}
	count := r.Count
	if count < 1 || count > 20 {
		count = 5
	}
	prompt := ctxText + fmt.Sprintf(`
## Tarefa
Sugira %d ideias de conteúdo novas e diferentes entre si para este perfil. Cada ideia precisa de um ângulo concreto (nada de "dicas sobre X"). Responda com um objeto {"ideas": [...]} em que cada item tem title, hook, summary e format (um de: short, video, live, carousel, image, story, text).`, count)
	if v := strings.TrimSpace(r.Instructions); v != "" {
		prompt += "\nInstruções extras do criador: " + v
	}
	var env ideasEnvelope
	if err := generateJSON(ctx, model, prompt, &env); err != nil {
		return nil, err
	}
	out := []Idea{}
	for _, idea := range env.Ideas {
		if strings.TrimSpace(idea.Title) == "" {
			continue
		}
		if !catalog.IsFormat(idea.Format) {
			idea.Format = orDefault(r.Format, "short")
		}
		out = append(out, idea)
	}
	return out, nil
}

// generateJSON asks for structured output; providers without JSON-schema
// support (many local models) fall back to plain text parsed as JSON.
func generateJSON[T any](ctx context.Context, model goaiModel, prompt string, out *T) error {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Minute)
	defer cancel()
	res, err := goai.GenerateObject[T](ctx, model,
		goai.WithSystem(systemBase),
		goai.WithPrompt(prompt),
		goai.WithMaxOutputTokens(8000),
	)
	if err == nil {
		*out = res.Object
		return nil
	}
	if ctx.Err() != nil {
		return friendlyAIError(ctx.Err())
	}
	text, err2 := goai.GenerateText(ctx, model,
		goai.WithSystem(systemBase+" Responda somente com um objeto JSON válido, sem blocos de código nem texto fora do JSON."),
		goai.WithPrompt(prompt),
		goai.WithMaxOutputTokens(8000),
	)
	if err2 != nil {
		return friendlyAIError(err2)
	}
	raw := extractJSON(text.Text)
	if err := json.Unmarshal([]byte(raw), out); err != nil {
		return fmt.Errorf("a resposta da IA não veio no formato esperado; tente de novo ou use outro modelo")
	}
	return nil
}

// extractJSON returns the outermost JSON object in s.
func extractJSON(s string) string {
	start := strings.Index(s, "{")
	end := strings.LastIndex(s, "}")
	if start < 0 || end <= start {
		return s
	}
	return s[start : end+1]
}

func stripHash(list []string) []string {
	out := make([]string, 0, len(list))
	for _, v := range list {
		out = append(out, strings.TrimLeft(strings.TrimSpace(v), "#"))
	}
	return out
}

func addHash(list []string) []string {
	out := make([]string, 0, len(list))
	for _, v := range list {
		v = strings.ReplaceAll(strings.TrimSpace(v), " ", "")
		if v == "" {
			continue
		}
		if !strings.HasPrefix(v, "#") {
			v = "#" + v
		}
		out = append(out, v)
	}
	return out
}
