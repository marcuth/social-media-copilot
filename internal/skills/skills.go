// Package skills bundles method guides ("skills") written as Markdown and
// decides which ones the AI should follow for a given task.
//
// Skills live in content/ and are embedded in the binary unchanged, so they
// can be refreshed by copying newer versions over the files. Each skill is
// declared in the registry below with the files that make it up and the
// situations it applies to.
package skills

import (
	"embed"
	"fmt"
	"slices"
	"strings"
)

//go:embed content
var content embed.FS

// Task is the kind of work the AI is doing.
type Task string

const (
	TaskScript   Task = "script"
	TaskMetadata Task = "metadata"
	TaskIdeas    Task = "ideas"
	// TaskChat skills are only offered in the free chat, where the model
	// loads them on demand.
	TaskChat Task = "chat"
)

// Skill is one method guide and where it applies.
type Skill struct {
	ID          string
	Name        string
	Description string // pt-BR, shown in the UI and in the chat catalog
	Source      string
	License     string
	// Files are embedded paths under content/; the first is the main file,
	// the rest are references it cites.
	Files []string
	// Applies to these tasks when the post targets any of Platforms (empty =
	// any) with one of Formats (empty = any). Every skill is also available
	// in the chat.
	Tasks     []Task
	Platforms []string
	Formats   []string
}

const (
	sourceMarcuth   = "Suas skills"
	sourceInstagram = "instagram-skills (Sergey Bulaev)"
	licenseMIT      = "MIT"
	igDir           = "content/instagram-skills/"
)

var registry = []Skill{
	{
		ID:          "roteiro-video-longo",
		Name:        "Roteiro de vídeo longo (método Jovens de Negócios)",
		Description: "Roteiros de vídeos longos educativos: tema, perguntas, conceitos, introdução com gancho, desenvolvimento em modelos e conclusão.",
		Source:      sourceMarcuth,
		Files:       []string{"content/marcuth/roteiro-video-longo.md"},
		Tasks:       []Task{TaskScript},
		Platforms:   []string{"youtube"},
		Formats:     []string{"video"},
	},
	{
		ID:          "roteiro-video-curto",
		Name:        "Roteiro de YouTube Shorts (método GPC)",
		Description: "Roteiros de Shorts com Gancho, Progresso e Clímax, com indicações de edição e de horário de postagem.",
		Source:      sourceMarcuth,
		Files:       []string{"content/marcuth/roteiro-video-curto.md"},
		Tasks:       []Task{TaskScript},
		Platforms:   []string{"youtube"},
		Formats:     []string{"short"},
	},
	{
		ID:          "ig-caption-writer",
		Name:        "Legenda de Instagram",
		Description: "Legendas com gancho nos primeiros 125 caracteres, corpo escaneável e uma chamada para ação, escolhendo a fórmula pelo objetivo.",
		Source:      sourceInstagram,
		License:     licenseMIT,
		Files: []string{
			igDir + "skills/ig-caption-writer/SKILL.md",
			igDir + "skills/ig-caption-writer/references/caption-checklist.md",
			igDir + "references/hook-formulas.md",
			igDir + "references/voice-rules.md",
		},
		Tasks:     []Task{TaskMetadata},
		Platforms: []string{"instagram"},
		Formats:   []string{"short", "video", "image", "story"},
	},
	{
		ID:          "ig-carousel-planner",
		Name:        "Carrossel de Instagram",
		Description: "Planeja carrosséis slide a slide (até 10), com slide de gancho, slides de valor e slide final que gera salvamentos.",
		Source:      sourceInstagram,
		License:     licenseMIT,
		Files: []string{
			igDir + "skills/ig-carousel-planner/SKILL.md",
			igDir + "skills/ig-carousel-planner/references/slide-architecture.md",
			igDir + "references/hook-formulas.md",
			igDir + "references/voice-rules.md",
		},
		Tasks:     []Task{TaskScript, TaskMetadata},
		Platforms: []string{"instagram"},
		Formats:   []string{"carousel"},
	},
	{
		ID:          "ig-hashtag-strategist",
		Name:        "Hashtags de Instagram",
		Description: "Monta 3 a 5 hashtags dimensionadas (nicho, médias e amplas) em vez de dezenas aleatórias.",
		Source:      sourceInstagram,
		License:     licenseMIT,
		Files: []string{
			igDir + "skills/ig-hashtag-strategist/SKILL.md",
			igDir + "references/hashtag-strategy.md",
		},
		Tasks:     []Task{TaskMetadata},
		Platforms: []string{"instagram"},
	},
	{
		ID:          "ig-humanizer",
		Name:        "Humanizar texto de Instagram",
		Description: "Remove marcas de texto gerado por IA (vocabulário, travessões, tríades, sinceridade encenada) de legendas e slides.",
		Source:      sourceInstagram,
		License:     licenseMIT,
		Files: []string{
			igDir + "skills/ig-humanizer/SKILL.md",
			igDir + "references/voice-rules.md",
		},
		Tasks:     []Task{TaskMetadata},
		Platforms: []string{"instagram"},
	},
	{
		ID:          "ig-content-planner",
		Name:        "Plano de conteúdo de Instagram",
		Description: "Plano semanal com mistura de Reels, carrosséis, imagens e stories, cada um com fórmula de gancho, ângulo e objetivo.",
		Source:      sourceInstagram,
		License:     licenseMIT,
		Files: []string{
			igDir + "skills/ig-content-planner/SKILL.md",
			igDir + "skills/ig-content-planner/references/pillars-framework.md",
			igDir + "references/hook-formulas.md",
			igDir + "references/algorithm-heuristics.md",
		},
		Tasks:     []Task{TaskIdeas},
		Platforms: []string{"instagram"},
	},
	{
		ID:          "ig-repurposer",
		Name:        "Adaptar conteúdo para o Instagram",
		Description: "Transforma um roteiro de YouTube, post de LinkedIn, blog ou thread em carrossel ou legenda nativos do Instagram.",
		Source:      sourceInstagram,
		License:     licenseMIT,
		Files: []string{
			igDir + "skills/ig-repurposer/SKILL.md",
			igDir + "references/hook-formulas.md",
			igDir + "references/voice-rules.md",
		},
		Tasks: []Task{TaskChat},
	},
	{
		ID:          "ig-profile-optimizer",
		Name:        "Otimizar perfil do Instagram",
		Description: "Revisa e reescreve nome, bio, link, destaques, grade inicial e posts fixados do perfil.",
		Source:      sourceInstagram,
		License:     licenseMIT,
		Files:       []string{igDir + "skills/ig-profile-optimizer/SKILL.md"},
		Tasks:       []Task{TaskChat},
	},
	{
		ID:          "ig-hook-extractor",
		Name:        "Analisar gancho de um post",
		Description: "Descobre qual fórmula de gancho um post colado usa, por que funcionou, e devolve um modelo para o seu tema.",
		Source:      sourceInstagram,
		License:     licenseMIT,
		Files: []string{
			igDir + "skills/ig-hook-extractor/SKILL.md",
			igDir + "skills/ig-hook-extractor/references/classification-rules.md",
			igDir + "references/hook-formulas.md",
		},
		Tasks: []Task{TaskChat},
	},
}

// All returns every registered skill.
func All() []Skill { return slices.Clone(registry) }

// ByID returns a skill and whether it exists.
func ByID(id string) (Skill, bool) {
	for _, s := range registry {
		if s.ID == id {
			return s, true
		}
	}
	return Skill{}, false
}

// Match returns the skills for a task, given the post's platforms and format,
// skipping the disabled ones.
func Match(task Task, platforms []string, format string, disabled map[string]bool) []Skill {
	var out []Skill
	for _, s := range registry {
		if disabled[s.ID] || !slices.Contains(s.Tasks, task) {
			continue
		}
		if len(s.Platforms) > 0 && !slices.ContainsFunc(platforms, func(p string) bool { return slices.Contains(s.Platforms, p) }) {
			continue
		}
		if len(s.Formats) > 0 && !slices.Contains(s.Formats, format) {
			continue
		}
		out = append(out, s)
	}
	return out
}

// Preamble tells the model how to use skills inside the app, which has no
// publishing tools and no way to ask follow-up questions mid-task.
const Preamble = `# Skills
As skills abaixo são manuais de método escritos para este tipo de tarefa. Siga os métodos, as regras e os checklists delas.
- Você está dentro do app Social Media Copilot e não tem ferramentas de publicação: ignore passos de publicar, agendar, enviar mídia, rodar scripts ou bibliotecas (lib.publish, Publora, Apify, MCP), carregar perfis de voz e pedir aprovação.
- Quando uma skill citar arquivos de referência, use os que vierem incluídos abaixo; ignore os que não vierem.
- Não faça perguntas de volta: use o contexto do perfil e do post; se faltar um dado concreto (número, data, resultado), não invente, deixe de fora.
- Se a skill pedir um formato de entrega diferente do pedido na tarefa, vale o formato da tarefa.
- Escreva no idioma do perfil, mesmo que a skill esteja em outro idioma.`

// Render returns the text of the skills, each referenced file included once.
func Render(list []Skill) (string, error) {
	if len(list) == 0 {
		return "", nil
	}
	body, err := renderBody(list)
	if err != nil {
		return "", err
	}
	return Preamble + body, nil
}

func renderBody(list []Skill) (string, error) {
	var b strings.Builder
	seen := map[string]bool{}
	for _, s := range list {
		fmt.Fprintf(&b, "\n\n## Skill: %s\n", s.Name)
		for _, f := range s.Files {
			if seen[f] {
				fmt.Fprintf(&b, "\n(Referência %s já incluída acima.)\n", strings.TrimPrefix(f, "content/"))
				continue
			}
			seen[f] = true
			text, err := content.ReadFile(f)
			if err != nil {
				return "", fmt.Errorf("skill %s: %w", s.ID, err)
			}
			fmt.Fprintf(&b, "\n<arquivo caminho=%q>\n%s\n</arquivo>\n", strings.TrimPrefix(f, "content/"), strings.TrimSpace(string(text)))
		}
	}
	return b.String(), nil
}

// Text returns the full text of one skill (without the preamble), for
// display or for the chat's load_skill tool.
func Text(id string) (string, error) {
	s, ok := ByID(id)
	if !ok {
		return "", fmt.Errorf("skill desconhecida: %s", id)
	}
	body, err := renderBody([]Skill{s})
	return strings.TrimSpace(body), err
}
