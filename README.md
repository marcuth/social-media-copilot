# Social Media Copilot

App desktop para planejar e produzir conteúdo de redes sociais: vários perfis, cada um com suas plataformas e temas, um calendário de publicação com opções de adiamento e um copiloto de IA que escreve roteiros e metadados (título, descrição, tags e hashtags).

Tudo fica no seu computador, num banco SQLite local. A IA funciona com o provedor que você escolher: OpenAI, Anthropic, Google Gemini, DeepSeek, Groq, OpenRouter, LM Studio, Ollama ou qualquer API compatível com OpenAI.

## O que dá para fazer

- **Perfis**: cada perfil é um canal ou marca. Você escolhe em quais plataformas ele publica (YouTube, TikTok, Instagram, Facebook, X, Threads, LinkedIn, Pinterest, Kwai), o @ em cada uma, o nicho, o público e o tom de voz.
- **Temas**: os pilares de conteúdo de cada perfil, usados como filtro e como contexto para a IA.
- **Posts**: título, descrição, tags, hashtags, roteiro (Markdown), anotações, formato, etapa de produção e data. Cada post escolhe em quais plataformas do perfil vai sair, e você marca a publicação plataforma por plataforma (com link).
- **Calendário**: visão mensal com arrastar e soltar, lista de atrasados e de posts sem data.
- **Adiar**: +1 dia, +3 dias, +1 semana, próximo dia livre do perfil ou uma data escolhida, com motivo. A opção "empurrar a fila" move também os próximos posts do perfil, mantendo o espaçamento. Todo adiamento fica no histórico do post.
- **Produção**: quadro kanban por etapa (Ideia, Roteiro, Gravação, Edição, Pronto, Publicado).
- **IA**: gera roteiros em tempo real (com duração alvo e instruções), sugere metadados respeitando as regras de cada plataforma e propõe ideias novas que entram direto na produção.

## Desenvolvimento

Requisitos: Go 1.25+, Node 20+ e o CLI do Wails v3.

```bash
go install github.com/wailsapp/wails/v3/cmd/wails3@v3.1.0

wails3 dev      # app com hot reload
wails3 build    # gera bin/social-media-copilot(.exe)
wails3 package  # instalador (Windows: NSIS; Linux: AppImage, deb, rpm)
go test ./...   # testes do backend
```

O banco fica em `<pasta de configuração do usuário>/social-media-copilot/social-media-copilot.db` (no Windows, `%APPDATA%\social-media-copilot`). Para usar outro arquivo, defina `SMC_DB_PATH`.

## Stack

- [Wails v3](https://v3.wails.io) (Go + WebView nativo) e Go 1.26
- Vue 3 + TypeScript + Vite, Tailwind CSS 4 e [shadcn-vue](https://www.shadcn-vue.com)
- SQLite via [modernc.org/sqlite](https://pkg.go.dev/modernc.org/sqlite) (sem CGO)
- [GoAI](https://github.com/zendev-sh/goai) para falar com os provedores de IA

## Licença

[MIT](LICENSE)
