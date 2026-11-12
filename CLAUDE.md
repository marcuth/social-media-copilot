# CLAUDE.md

Social Media Copilot is a local-first desktop app (Wails v3 + Go + Vue 3/TypeScript + SQLite) for planning social media content across several profiles, with AI help for scripts and metadata. The UI is in Brazilian Portuguese; keep new user-facing strings in pt-BR.

## Commands

```bash
wails3 dev                 # the real app with hot reload (Vite on port 9245)
wails3 build               # bin/social-media-copilot(.exe); also regenerates bindings
wails3 generate bindings -clean=true -ts -i   # after changing any exported service method/model
go test ./...              # backend tests (internal/services/posts_test.go)
go vet ./...
cd frontend && npm run ci             # eslint + prettier --check + vue-tsc
cd frontend && npm run lint:fix && npm run format
```

CI (`.github/workflows/ci.yml`, on push/PR to `main`) runs `lint:fix`, `format`, `gofmt` and `go mod tidy`, then pushes a `chore: auto format and lint fix [skip ci]` commit as github-actions[bot] when something changed, so the remote is often one commit ahead: `git pull --rebase` before pushing. It then runs `npm run lint`, `npm run build`, `go vet ./internal/...` and `go test ./internal/...` (the `main` package needs CGO + GTK on Linux, so CI doesn't build it). Dependabot updates Go modules, npm (`frontend/`) and actions weekly.

`SMC_DB_PATH` overrides the database location (default: `os.UserConfigDir()/social-media-copilot/social-media-copilot.db`). Use it to try things against a throwaway database.

## Architecture

- `main.go` opens the DB, builds the services and registers them with Wails. `AIService.Emit` is wired to `app.Event.Emit` after the app exists.
- `internal/db`: SQLite through `modernc.org/sqlite` (pure Go, so builds use `CGO_ENABLED=0`). One connection (`SetMaxOpenConns(1)`): never call `s.DB` while a transaction is open in the same goroutine, it deadlocks. Schema lives in the `migrations` slice, versioned by `PRAGMA user_version`; append new entries, never edit shipped ones.
- `internal/models`: types shared with the frontend. Schedule strings are local wall-clock `"2006-01-02T15:04"`; audit timestamps are RFC3339.
- `internal/catalog`: fixed vocabularies (platforms with AI guidance, formats, statuses). Adding a platform also needs an icon in `frontend/src/components/PlatformIcon.vue`.
- `internal/services`: every exported method becomes a TS function in `frontend/bindings` (generated, do not edit). Methods taking `context.Context` are cancellable from JS via the returned `CancellablePromise`.
  - `PostService`: CRUD, platform publication state, `Postpone` (days / date / next free day, optional cascade over later unpublished posts of the profile), `Reschedule` (drag on calendar; moving later counts as a postponement).
  - `ProviderService` + `AIService`: AI providers are stored rows (`openai_compatible`, `anthropic`, `google`, `ollama`) turned into GoAI models by `buildModel`. Scripts stream through the `ai:delta` event, routed by a `requestId` chosen in the frontend. Metadata/ideas use `goai.GenerateObject` and fall back to plain text + JSON extraction for models without structured output.
- `internal/skills`: method guides ("skills") embedded from `content/` and a registry mapping each one to tasks (`script`, `metadata`, `ideas`, `chat`), platforms and formats. `content/marcuth/` holds the user's own skills; `content/instagram-skills/` is vendored unchanged from sergebulaev/instagram-skills (MIT, keep its LICENSE; see its README for how to update). Never edit vendored files: adapt behaviour through the registry or the `Preamble`.
  - Generations call `AIService.system(task, platforms, format)`, which appends the matching, enabled skills to the system prompt. `SkillService` lists them and stores disabled ids in `app_settings` (`skills.disabled`).
  - The chat (`ChatService`) does not inline skills: it lists them in the prompt and exposes a `load_skill` tool; if the model rejects tools it retries without them.
- `ChatService`: conversations in `chat_sessions` / `chat_messages`; the prompt carries the profiles (or the focused one), the post open in the editor (sent by the frontend with unsaved edits) and the schedule from 7 days ago to 21 days ahead. Don't use `goai.WithTimeout` on tool-calling streams: in goai v0.11.0 it cancels the stream as soon as `StreamText` returns; wrap the context yourself (see `ChatService.stream`).
- Frontend (`frontend/src`): `lib/api.ts` is the only place importing bindings (and normalises Go nil slices); `lib/store.ts` holds catalog, profiles, themes, providers and the global profile filter; `lib/confirm.ts` + `components/ConfirmDialog.vue` replace `window.confirm`; `lib/theme.ts` toggles the `.dark` class (light / dark / system); `lib/chat.ts` holds the copilot chat state (the panel is `CopilotChat.vue`: a right-side Sheet that expands into a Dialog, both rendering `ChatPanel.vue`). Views in `views/`, app components in `components/`, shadcn-vue components in `components/ui/`.

## Conventions

- Go errors returned to the UI are written in pt-BR, in plain language.
- Frontend: Prettier (4 spaces, double quotes, no semicolons, imports sorted by length) and ESLint (`eslint.config.js`); `interface` for object shapes; no `any`.
- Comments only for what the code can't say (why, gotchas, non-obvious behaviour); don't restate names.
- UI: Tailwind CSS 4 + shadcn-vue (reka-ui), with the same design system as Indie Copilot (Nunito Sans body, Merriweather `font-heading` titles, `--radius: 1rem`, cards `rounded-xl border border-border/60 bg-card shadow-xs`, pages = `PageHeader` + `px-8 py-6` content). Only the primary colour differs (`#e11d62`). Tokens live in `src/styles/main.css`.
- Add shadcn components with `npx shadcn-vue@latest add <name>` (run in `frontend/`), never by hand; then align variants with Indie Copilot's (`button` has `cursor-pointer`, translucent `outline`, `lg` = h-12; `badge` is `rounded-md`).
- Use `AppSelect` instead of raw `Select` when values can be numbers or null (reka-ui items need string values).
- Icons: `@lucide/vue`. Platform marks: `components/PlatformIcon.vue` (simple-icons).
