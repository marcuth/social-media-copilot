// Package catalog defines the fixed vocabularies of the app: platforms,
// content formats and pipeline statuses.
package catalog

// Platform describes a social network the app knows how to plan for.
type Platform struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Color string `json:"color"`
	// Guidance is fed to the AI so metadata respects each platform's norms.
	Guidance string `json:"guidance"`
}

// Option is a generic id/label pair.
type Option struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}

var Platforms = []Platform{
	{"youtube", "YouTube", "#FF0033", "Vídeo longo: título até 100 caracteres (ideal 50-70) com a palavra-chave no início, descrição longa com resumo, palavras-chave e capítulos quando fizer sentido, tags relevantes e 3 a 5 hashtags. Shorts (vídeo curto): título curto e direto (até 60 caracteres), descrição de 1-2 frases e #shorts entre as hashtags."},
	{"tiktok", "TikTok", "#25F4EE", "Legenda curta com gancho na primeira linha (até 150 caracteres é o ideal); 3 a 6 hashtags misturando nicho e alcance amplo."},
	{"instagram", "Instagram", "#E1306C", "Legenda com gancho na primeira linha, corpo escaneável com quebras de linha e chamada para ação; 5 a 15 hashtags de nicho."},
	{"facebook", "Facebook", "#1877F2", "Texto conversacional, pergunta para engajamento; poucas hashtags (1 a 3)."},
	{"x", "X (Twitter)", "#1D1D1F", "Texto de até 280 caracteres; no máximo 1 ou 2 hashtags."},
	{"threads", "Threads", "#5B5B66", "Tom conversacional, até 500 caracteres; no máximo 1 hashtag (tópico)."},
	{"linkedin", "LinkedIn", "#0A66C2", "Gancho forte nas duas primeiras linhas, tom profissional, parágrafos curtos; 3 a 5 hashtags."},
	{"pinterest", "Pinterest", "#E60023", "Título descritivo com palavras-chave (até 100 caracteres); descrição de até 500 caracteres otimizada para busca."},
	{"kwai", "Kwai", "#FF7A00", "Legenda curta e popular, com gancho; 3 a 5 hashtags."},
}

var Formats = []Option{
	{"short", "Vídeo curto"},
	{"video", "Vídeo longo"},
	{"live", "Live"},
	{"carousel", "Carrossel"},
	{"image", "Imagem"},
	{"story", "Story"},
	{"text", "Texto"},
}

// Statuses are ordered as the production pipeline flows.
var Statuses = []Option{
	{"idea", "Ideia"},
	{"scripting", "Roteiro"},
	{"production", "Gravação"},
	{"editing", "Edição"},
	{"ready", "Pronto"},
	{"published", "Publicado"},
}

func has(list []Option, id string) bool {
	for _, o := range list {
		if o.ID == id {
			return true
		}
	}
	return false
}

// PlatformByID returns the platform and whether it exists.
func PlatformByID(id string) (Platform, bool) {
	for _, p := range Platforms {
		if p.ID == id {
			return p, true
		}
	}
	return Platform{}, false
}

func IsFormat(id string) bool { return has(Formats, id) }
func IsStatus(id string) bool { return has(Statuses, id) }

// Label returns the label of id inside list, or id itself.
func Label(list []Option, id string) string {
	for _, o := range list {
		if o.ID == id {
			return o.Label
		}
	}
	return id
}
