package services

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/zendev-sh/goai"
	"github.com/zendev-sh/goai/provider"

	"github.com/marcuth/social-media-copilot/internal/catalog"
	"github.com/marcuth/social-media-copilot/internal/models"
	"github.com/marcuth/social-media-copilot/internal/skills"
)

type ChatSession struct {
	ID        int64  `json:"id"`
	Title     string `json:"title"`
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
}

type ChatMessage struct {
	ID        int64  `json:"id"`
	SessionID int64  `json:"sessionId"`
	Role      string `json:"role"` // user | assistant
	Content   string `json:"content"`
	Model     string `json:"model"`
	CreatedAt string `json:"createdAt"`
}

// ChatRequest sends a message. SessionID 0 starts a new conversation.
type ChatRequest struct {
	SessionID int64  `json:"sessionId"`
	RequestID string `json:"requestId"`
	Content   string `json:"content"`
	// ProfileID focuses the context on one profile (nil = every profile).
	ProfileID *int64 `json:"profileId"`
	// Post is the post open in the editor, including unsaved edits.
	Post *models.Post `json:"post"`
}

type ChatReply struct {
	Session ChatSession `json:"session"`
	Message ChatMessage `json:"message"`
}

type ChatService struct {
	DB    *sql.DB
	AI    *AIService
	Posts *PostService
}

const maxHistory = 40

const chatSystem = `Você é o Copiloto do app Social Media Copilot e ajuda um criador a planejar e produzir conteúdo para redes sociais: ideias, roteiros, títulos, legendas, hashtags, calendário e estratégia.
- Responda em português do Brasil, em Markdown, de forma direta. Ao escrever conteúdo para publicar, use o idioma do perfil.
- Você não altera dados do app. Quando sugerir títulos, roteiros, legendas ou datas, entregue o texto pronto para o criador copiar.
- Não invente números, métricas, datas ou fatos. Se faltar informação para uma boa resposta, pergunte.`

const chatSkillRules = `# Skills
Você tem a ferramenta load_skill, que carrega manuais de método. Antes de escrever algo coberto por uma skill (por exemplo roteiro de vídeo longo ou de Shorts para o YouTube, legenda, carrossel ou hashtags de Instagram), carregue a skill e siga o método dela. Ao usar uma skill dentro deste app, ignore passos de publicar, agendar, enviar mídia, rodar scripts ou bibliotecas (lib.publish, Publora, Apify, MCP) e de carregar perfis de voz. Não diga ao usuário que carregou uma skill, apenas aplique.`

func (s *ChatService) ListSessions() ([]ChatSession, error) {
	rows, err := s.DB.Query(`SELECT id, title, created_at, updated_at FROM chat_sessions ORDER BY updated_at DESC LIMIT 100`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []ChatSession{}
	for rows.Next() {
		var c ChatSession
		if err := rows.Scan(&c.ID, &c.Title, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (s *ChatService) GetMessages(sessionID int64) ([]ChatMessage, error) {
	rows, err := s.DB.Query(`SELECT id, session_id, role, content, model, created_at FROM chat_messages
		WHERE session_id = ? ORDER BY id`, sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []ChatMessage{}
	for rows.Next() {
		var m ChatMessage
		if err := rows.Scan(&m.ID, &m.SessionID, &m.Role, &m.Content, &m.Model, &m.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (s *ChatService) DeleteSession(id int64) error {
	_, err := s.DB.Exec(`DELETE FROM chat_sessions WHERE id = ?`, id)
	return err
}

func (s *ChatService) getSession(id int64) (ChatSession, error) {
	var c ChatSession
	err := s.DB.QueryRow(`SELECT id, title, created_at, updated_at FROM chat_sessions WHERE id = ?`, id).
		Scan(&c.ID, &c.Title, &c.CreatedAt, &c.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return c, ErrNotFound
	}
	return c, err
}

func (s *ChatService) addMessage(sessionID int64, role, content, model string) (ChatMessage, error) {
	now := nowStamp()
	res, err := s.DB.Exec(`INSERT INTO chat_messages (session_id, role, content, model, created_at) VALUES (?, ?, ?, ?, ?)`,
		sessionID, role, content, model, now)
	if err != nil {
		return ChatMessage{}, err
	}
	id, _ := res.LastInsertId()
	if _, err := s.DB.Exec(`UPDATE chat_sessions SET updated_at = ? WHERE id = ?`, now, sessionID); err != nil {
		return ChatMessage{}, err
	}
	return ChatMessage{ID: id, SessionID: sessionID, Role: role, Content: content, Model: model, CreatedAt: now}, nil
}

func sessionTitle(content string) string {
	title := strings.Join(strings.Fields(content), " ")
	if utf8.RuneCountInString(title) > 60 {
		title = string([]rune(title)[:57]) + "…"
	}
	return title
}

// SendMessage saves the user's message, streams the answer through the
// "ai:delta" event (routed by RequestID) and saves it. Cancelling the call
// stops the generation.
func (s *ChatService) SendMessage(ctx context.Context, r ChatRequest) (ChatReply, error) {
	var reply ChatReply
	content := strings.TrimSpace(r.Content)
	if content == "" {
		return reply, errors.New("escreva uma mensagem")
	}
	providerRow, err := s.AI.Providers.defaultProvider()
	if err != nil {
		return reply, err
	}
	model, err := buildModel(providerRow)
	if err != nil {
		return reply, err
	}

	session := ChatSession{ID: r.SessionID}
	if session.ID == 0 {
		now := nowStamp()
		res, err := s.DB.Exec(`INSERT INTO chat_sessions (title, created_at, updated_at) VALUES (?, ?, ?)`,
			sessionTitle(content), now, now)
		if err != nil {
			return reply, err
		}
		session.ID, _ = res.LastInsertId()
	}
	if _, err := s.addMessage(session.ID, "user", content, ""); err != nil {
		return reply, err
	}
	history, err := s.GetMessages(session.ID)
	if err != nil {
		return reply, err
	}
	if len(history) > maxHistory {
		history = history[len(history)-maxHistory:]
	}
	msgs := make([]provider.Message, 0, len(history))
	for _, m := range history {
		if m.Role == "assistant" {
			msgs = append(msgs, goai.AssistantMessage(m.Content))
		} else {
			msgs = append(msgs, goai.UserMessage(m.Content))
		}
	}

	contextText := s.contextText(r)
	catalogText, tools := s.skillTools()
	text, err := s.stream(ctx, model, r.RequestID, chatSystem+"\n\n"+catalogText+"\n\n"+contextText, msgs, tools)
	if err != nil && text == "" && len(tools) > 0 && ctx.Err() == nil {
		// Some models (often local ones) reject tool definitions: retry without them.
		text, err = s.stream(ctx, model, r.RequestID, chatSystem+"\n\n"+contextText, msgs, nil)
	}
	if err != nil {
		if text != "" {
			s.addMessage(session.ID, "assistant", strings.TrimSpace(text), providerRow.Name+" · "+providerRow.Model)
		}
		return reply, friendlyAIError(err)
	}

	reply.Message, err = s.addMessage(session.ID, "assistant", strings.TrimSpace(text), providerRow.Name+" · "+providerRow.Model)
	if err != nil {
		return reply, err
	}
	reply.Session, err = s.getSession(session.ID)
	return reply, err
}

func (s *ChatService) stream(ctx context.Context, model goaiModel, requestID, system string, msgs []provider.Message, tools []goai.Tool) (string, error) {
	// goai.WithTimeout cancels tool-loop streams as soon as StreamText returns
	// (goai v0.11.0), so the timeout is applied here and released only after
	// the stream has been read.
	ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()
	opts := []goai.Option{
		goai.WithSystem(system),
		goai.WithMessages(msgs...),
		goai.WithMaxOutputTokens(8000),
	}
	if len(tools) > 0 {
		opts = append(opts, goai.WithTools(tools...), goai.WithMaxSteps(4))
	}
	stream, err := goai.StreamText(ctx, model, opts...)
	if err != nil {
		return "", err
	}
	var full strings.Builder
	for chunk := range stream.TextStream() {
		full.WriteString(chunk)
		if s.AI.Emit != nil && requestID != "" {
			s.AI.Emit(EventAIDelta, AIDelta{RequestID: requestID, Delta: chunk})
		}
	}
	return full.String(), stream.Err()
}

func (s *ChatService) skillTools() (string, []goai.Tool) {
	disabled := map[string]bool{}
	if s.AI.Skills != nil {
		if d, err := s.AI.Skills.disabled(); err == nil {
			disabled = d
		}
	}
	var b strings.Builder
	b.WriteString(chatSkillRules)
	b.WriteString("\n\nSkills disponíveis (id: nome, descrição, quando usar):\n")
	count := 0
	for _, sk := range skills.All() {
		if disabled[sk.ID] {
			continue
		}
		count++
		fmt.Fprintf(&b, "- %s: %s. %s %s\n", sk.ID, sk.Name, sk.Description, appliesTo(sk))
	}
	if count == 0 {
		return "", nil
	}
	type input struct {
		ID string `json:"id" jsonschema:"description=Id da skill a carregar"`
	}
	tool := goai.NewTool("load_skill", "Carrega o texto completo de uma skill (manual de método) pelo id.",
		func(_ context.Context, in input) (string, error) {
			if disabled[in.ID] {
				return "", fmt.Errorf("a skill %s está desativada", in.ID)
			}
			return skills.Text(in.ID)
		})
	return b.String(), []goai.Tool{tool}
}

func (s *ChatService) contextText(r ChatRequest) string {
	var b strings.Builder
	now := time.Now()
	weekdays := []string{"domingo", "segunda", "terça", "quarta", "quinta", "sexta", "sábado"}
	fmt.Fprintf(&b, "# Contexto do app\nAgora: %s, %s.\n", weekdays[now.Weekday()], now.Format("02/01/2006 15:04"))

	profiles, _ := s.AI.Profiles.ListProfiles()
	themes, _ := s.AI.Profiles.ListThemes(0)
	names := map[int64]string{}
	b.WriteString("\n## Perfis\n")
	if len(profiles) == 0 {
		b.WriteString("Nenhum perfil cadastrado ainda.\n")
	}
	for _, p := range profiles {
		names[p.ID] = p.Name
		if r.ProfileID != nil && *r.ProfileID != p.ID {
			continue
		}
		fmt.Fprintf(&b, "### %s\n", p.Name)
		writeIf(&b, "Descrição", p.Description)
		writeIf(&b, "Nicho", p.Niche)
		writeIf(&b, "Público", p.Audience)
		writeIf(&b, "Tom de voz", p.Tone)
		writeIf(&b, "Idioma", p.Language)
		var plats []string
		for _, pp := range p.Platforms {
			if c, ok := catalog.PlatformByID(pp.Platform); ok {
				plats = append(plats, strings.TrimSpace(c.Name+" "+pp.Handle))
			}
		}
		writeIf(&b, "Plataformas", strings.Join(plats, ", "))
		var ts []string
		for _, t := range themes {
			if t.ProfileID == p.ID {
				ts = append(ts, t.Name)
			}
		}
		writeIf(&b, "Temas", strings.Join(ts, ", "))
	}
	if r.ProfileID != nil {
		b.WriteString("(O criador está olhando só este perfil.)\n")
	}

	if p := r.Post; p != nil {
		b.WriteString("\n## Post aberto no editor\n")
		writeIf(&b, "Perfil", names[p.ProfileID])
		writeIf(&b, "Título", p.Title)
		writeIf(&b, "Formato", catalog.Label(catalog.Formats, p.Format))
		writeIf(&b, "Etapa", catalog.Label(catalog.Statuses, p.Status))
		if p.ScheduledAt != nil {
			writeIf(&b, "Data", *p.ScheduledAt)
		}
		var plats []string
		for _, pp := range p.Platforms {
			if c, ok := catalog.PlatformByID(pp.Platform); ok {
				plats = append(plats, c.Name)
			}
		}
		writeIf(&b, "Plataformas", strings.Join(plats, ", "))
		writeIf(&b, "Descrição", p.Description)
		writeIf(&b, "Tags", strings.Join(p.Tags, ", "))
		writeIf(&b, "Hashtags", strings.Join(p.Hashtags, " "))
		writeIf(&b, "Anotações", p.Notes)
		if strings.TrimSpace(p.Script) != "" {
			b.WriteString("Roteiro:\n" + p.Script + "\n")
		}
	}

	filter := models.PostFilter{
		ProfileID: r.ProfileID,
		From:      now.AddDate(0, 0, -7).Format(dayLayout),
		To:        now.AddDate(0, 0, 21).Format(dayLayout),
	}
	if posts, err := s.Posts.ListPosts(filter); err == nil && len(posts) > 0 {
		b.WriteString("\n## Agenda (7 dias atrás a 21 dias à frente)\n")
		for _, p := range posts {
			var plats []string
			for _, pp := range p.Platforms {
				if c, ok := catalog.PlatformByID(pp.Platform); ok {
					plats = append(plats, c.Name)
				}
			}
			late := ""
			if p.Status != "published" && p.ScheduledAt != nil && *p.ScheduledAt < now.Format(ScheduleLayout) {
				late = " (atrasado)"
			}
			fmt.Fprintf(&b, "- %s · %s · %s · %s%s · %s\n", *p.ScheduledAt, names[p.ProfileID], p.Title,
				catalog.Label(catalog.Statuses, p.Status), late, strings.Join(plats, ", "))
		}
	}
	return b.String()
}
