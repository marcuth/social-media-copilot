<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, reactive, ref, watch } from "vue"
import { onBeforeRouteLeave, useRoute, useRouter } from "vue-router"
import { Events } from "@wailsio/runtime"
import { ArrowLeft, CalendarClock, Copy, Eye, Pencil, Sparkles, Square, Trash2, Wand2 } from "lucide-vue-next"

import MarkdownView from "../components/MarkdownView.vue"
import PlatformIcon from "../components/PlatformIcon.vue"
import PlatformPicker from "../components/PlatformPicker.vue"
import PostponeDialog from "../components/PostponeDialog.vue"
import TagInput from "../components/TagInput.vue"
import {
    AIService,
    AiDelta,
    PostService,
    isCancel,
    normPost,
    type AIRequest,
    type Metadata,
    type PostN,
    type Postponement,
} from "../lib/api"
import { formatSchedule } from "../lib/dates"
import { STATUS_COLORS, platformInfo, profileById, store, themesOf } from "../lib/store"
import { toast, toastError } from "../lib/toast"

const props = defineProps<{ id: string }>()
const route = useRoute()
const router = useRouter()

const isNew = computed(() => props.id === "new")

function blank(): PostN {
    const profileId = Number(route.query.profile) || store.activeProfileId || store.profiles[0]?.id || 0
    const profile = profileById(profileId)
    return {
        id: 0,
        profileId,
        themeId: null,
        title: "",
        description: "",
        script: "",
        notes: "",
        tags: [],
        hashtags: [],
        format: "short",
        status: (route.query.status as string) || "idea",
        scheduledAt: (route.query.date as string) || null,
        publishedAt: null,
        postponeCount: 0,
        platforms: (profile?.platforms ?? []).map((p) => ({
            platform: p.platform,
            status: "pending",
            url: "",
            publishedAt: null,
        })),
        createdAt: "",
        updatedAt: "",
    }
}

const post = ref<PostN>(blank())
const savedJSON = ref("")
const history = ref<Postponement[]>([])
const saving = ref(false)
const postponing = ref(false)
const scriptPreview = ref(false)

const profile = computed(() => profileById(post.value.profileId))
const themes = computed(() => themesOf(post.value.profileId))
const availablePlatforms = computed(() => profile.value?.platforms.map((p) => p.platform) ?? [])
const selectedPlatforms = computed({
    get: () => post.value.platforms.map((p) => p.platform),
    set: (ids: string[]) => {
        const old = new Map(post.value.platforms.map((p) => [p.platform, p]))
        post.value.platforms = ids.map(
            (id) => old.get(id) ?? { platform: id, status: "pending", url: "", publishedAt: null },
        )
    },
})
const dirty = computed(() => JSON.stringify(editable(post.value)) !== savedJSON.value)

function editable(p: PostN): unknown {
    const { platforms, postponeCount: _c, updatedAt: _u, createdAt: _cr, publishedAt: _p, ...rest } = p
    return { ...rest, platforms: platforms.map((x) => x.platform) }
}

function markSaved(): void {
    savedJSON.value = JSON.stringify(editable(post.value))
}

async function load(): Promise<void> {
    if (isNew.value) {
        post.value = blank()
        markSaved()
        return
    }
    try {
        post.value = normPost(await PostService.GetPost(Number(props.id)))
        markSaved()
        history.value = (await PostService.ListPostponements(post.value.id)) ?? []
    } catch (e) {
        toastError(e)
        router.replace("/")
    }
}

// Switching profile keeps only the platforms the new profile has.
watch(
    () => post.value.profileId,
    (id, old) => {
        if (!old || id === old) return
        const allowed = new Set(availablePlatforms.value)
        selectedPlatforms.value = selectedPlatforms.value.filter((p) => allowed.has(p))
        if (!selectedPlatforms.value.length) selectedPlatforms.value = [...allowed]
        if (post.value.themeId && !themes.value.some((t) => t.id === post.value.themeId)) post.value.themeId = null
    },
)

async function save(): Promise<void> {
    saving.value = true
    try {
        const wasNew = isNew.value
        const saved = normPost(await PostService.SavePost(post.value))
        post.value = saved
        markSaved()
        toast("Post salvo.")
        if (wasNew) router.replace({ name: "post", params: { id: saved.id } })
    } catch (e) {
        toastError(e)
    } finally {
        saving.value = false
    }
}

async function remove(): Promise<void> {
    if (!confirm(`Excluir "${post.value.title}"? Isso não pode ser desfeito.`)) return
    try {
        await PostService.DeletePost(post.value.id)
        markSaved()
        toast("Post excluído.")
        router.push("/")
    } catch (e) {
        toastError(e)
    }
}

async function duplicate(): Promise<void> {
    try {
        const copy = await PostService.DuplicatePost(post.value.id)
        toast("Cópia criada sem data.")
        router.push({ name: "post", params: { id: copy.id } })
    } catch (e) {
        toastError(e)
    }
}

async function onPostponed(updated: PostN): Promise<void> {
    postponing.value = false
    // Keep unsaved edits; only take the new schedule fields.
    post.value.scheduledAt = updated.scheduledAt
    post.value.postponeCount = updated.postponeCount
    const snapshot = JSON.parse(savedJSON.value)
    snapshot.scheduledAt = updated.scheduledAt
    savedJSON.value = JSON.stringify(snapshot)
    history.value = (await PostService.ListPostponements(post.value.id)) ?? []
}

async function setPlatformStatus(platform: string, status: string, url: string): Promise<void> {
    try {
        const updated = normPost(await PostService.SetPlatformStatus(post.value.id, platform, status, url))
        post.value.platforms = updated.platforms
        if (updated.status !== post.value.status) {
            post.value.status = updated.status
            post.value.publishedAt = updated.publishedAt
            const snapshot = JSON.parse(savedJSON.value)
            snapshot.status = updated.status
            savedJSON.value = JSON.stringify(snapshot)
            if (updated.status === "published") toast("Publicado em todas as plataformas.")
        }
    } catch (e) {
        toastError(e)
    }
}

// ── AI ────────────────────────────────────────────────────────────────────
const aiTab = ref<"script" | "metadata">("script")
const ai = reactive({
    duration: 60,
    scriptInstructions: "",
    metaInstructions: "",
    scriptDraft: "",
    generatingScript: false,
    generatingMeta: false,
    meta: null as Metadata | null,
})
let scriptCall: { cancel(): void } | null = null
let requestId = ""

const offDelta = Events.On(AiDelta, (ev: { data: { requestId: string; delta: string } }) => {
    if (ev.data.requestId === requestId) ai.scriptDraft += ev.data.delta
})
onBeforeUnmount(() => {
    offDelta()
    scriptCall?.cancel()
})

function aiRequest(instructions: string): AIRequest {
    const p = post.value
    return {
        requestId,
        profileId: p.profileId,
        themeId: p.themeId,
        format: p.format,
        platforms: selectedPlatforms.value,
        title: p.title,
        description: p.description,
        script: p.script,
        notes: p.notes,
        durationSeconds: ai.duration,
        instructions,
    }
}

async function generateScript(): Promise<void> {
    if (!store.providers.length) return needProvider()
    requestId = crypto.randomUUID()
    ai.scriptDraft = ""
    ai.generatingScript = true
    const call = AIService.GenerateScript(aiRequest(ai.scriptInstructions))
    scriptCall = call
    try {
        ai.scriptDraft = await call
    } catch (e) {
        if (!isCancel(e)) toastError(e)
    } finally {
        ai.generatingScript = false
        scriptCall = null
    }
}

function stopScript(): void {
    scriptCall?.cancel()
}

function useScript(): void {
    post.value.script = ai.scriptDraft
    ai.scriptDraft = ""
    scriptPreview.value = true
    toast("Roteiro aplicado. Salve para guardar.")
}

async function generateMetadata(): Promise<void> {
    if (!store.providers.length) return needProvider()
    ai.generatingMeta = true
    try {
        const m = await AIService.GenerateMetadata(aiRequest(ai.metaInstructions))
        ai.meta = { ...m, tags: m.tags ?? [], hashtags: m.hashtags ?? [] }
    } catch (e) {
        if (!isCancel(e)) toastError(e)
    } finally {
        ai.generatingMeta = false
    }
}

function applyMeta(field: keyof Metadata | "all"): void {
    const m = ai.meta
    if (!m) return
    if (field === "title" || field === "all") post.value.title = m.title
    if (field === "description" || field === "all") post.value.description = m.description
    if (field === "tags" || field === "all") post.value.tags = [...(m.tags ?? [])]
    if (field === "hashtags" || field === "all") post.value.hashtags = [...(m.hashtags ?? [])]
}

function needProvider(): void {
    toast("Configure um provedor de IA em Configurações para usar esta função.", "error")
}

// ── Keyboard & leave guard ───────────────────────────────────────────────
function onKey(e: KeyboardEvent): void {
    if ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === "s") {
        e.preventDefault()
        save()
    }
}
onMounted(() => {
    load()
    document.addEventListener("keydown", onKey)
})
onBeforeUnmount(() => document.removeEventListener("keydown", onKey))
onBeforeRouteLeave(() => !dirty.value || confirm("Há alterações não salvas. Sair mesmo assim?"))

const durations = [
    { v: 15, l: "15 s" },
    { v: 30, l: "30 s" },
    { v: 60, l: "1 min" },
    { v: 90, l: "1min30" },
    { v: 180, l: "3 min" },
    { v: 480, l: "8 min" },
    { v: 900, l: "15 min" },
]
const PLATFORM_STATUS = [
    { id: "pending", label: "Pendente" },
    { id: "published", label: "Publicado" },
    { id: "skipped", label: "Não vai sair" },
]
</script>

<template>
    <div class="page editor">
        <header class="page-head">
            <div class="row">
                <button class="btn ghost icon" aria-label="Voltar" @click="router.back()"><ArrowLeft /></button>
                <div>
                    <h1>{{ isNew ? "Novo post" : post.title || "Sem título" }}</h1>
                    <p v-if="!isNew">
                        {{ formatSchedule(post.scheduledAt) }}
                        <template v-if="post.postponeCount"> · adiado {{ post.postponeCount }}x</template>
                    </p>
                </div>
            </div>
            <div class="row">
                <template v-if="!isNew">
                    <button class="btn ghost icon" aria-label="Duplicar" title="Duplicar" @click="duplicate">
                        <Copy />
                    </button>
                    <button class="btn ghost icon danger" aria-label="Excluir" title="Excluir" @click="remove">
                        <Trash2 />
                    </button>
                    <button
                        class="btn"
                        :disabled="post.status === 'published'"
                        @click="postponing = true"
                    >
                        <CalendarClock /> Adiar
                    </button>
                </template>
                <button class="btn primary" :disabled="saving || (!dirty && !isNew)" @click="save">
                    {{ dirty || isNew ? "Salvar" : "Salvo" }}
                </button>
            </div>
        </header>

        <div v-if="!store.profiles.length" class="panel empty">
            <h3>Crie um perfil primeiro</h3>
            <p>Todo post pertence a um perfil, e é nele que você diz em quais plataformas publica.</p>
            <RouterLink to="/profiles" class="btn primary">Criar perfil</RouterLink>
        </div>

        <div v-else class="cols">
            <form class="form" @submit.prevent="save">
                <section class="panel block">
                    <div class="grid-3">
                        <label class="field">
                            <span>Perfil</span>
                            <select v-model="post.profileId" class="select">
                                <option v-for="p in store.profiles" :key="p.id" :value="p.id">{{ p.name }}</option>
                            </select>
                        </label>
                        <label class="field">
                            <span>Tema</span>
                            <select v-model="post.themeId" class="select">
                                <option :value="null">Sem tema</option>
                                <option v-for="t in themes" :key="t.id" :value="t.id">{{ t.name }}</option>
                            </select>
                        </label>
                        <label class="field">
                            <span>Formato</span>
                            <select v-model="post.format" class="select">
                                <option v-for="f in store.catalog.formats" :key="f.id" :value="f.id">
                                    {{ f.label }}
                                </option>
                            </select>
                        </label>
                    </div>

                    <div class="grid-2">
                        <label class="field">
                            <span>Etapa</span>
                            <select v-model="post.status" class="select">
                                <option v-for="s in store.catalog.statuses" :key="s.id" :value="s.id">
                                    {{ s.label }}
                                </option>
                            </select>
                        </label>
                        <label class="field">
                            <span>Data de publicação</span>
                            <input
                                class="input"
                                type="datetime-local"
                                :value="post.scheduledAt ?? ''"
                                @input="post.scheduledAt = ($event.target as HTMLInputElement).value || null"
                            />
                        </label>
                    </div>

                    <div class="stages" aria-hidden="true">
                        <span
                            v-for="s in store.catalog.statuses"
                            :key="s.id"
                            :style="{ background: STATUS_COLORS[s.id] }"
                            :class="{
                                reached:
                                    store.catalog.statuses.findIndex((x) => x.id === s.id) <=
                                    store.catalog.statuses.findIndex((x) => x.id === post.status),
                            }"
                        ></span>
                    </div>

                    <div class="field">
                        <span>Publicar em</span>
                        <PlatformPicker
                            v-if="availablePlatforms.length"
                            v-model="selectedPlatforms"
                            :available="availablePlatforms"
                        />
                        <p v-else class="muted">
                            Este perfil ainda não tem plataformas.
                            <RouterLink to="/profiles">Adicionar plataformas ao perfil</RouterLink>
                        </p>
                    </div>
                </section>

                <section class="panel block">
                    <label class="field">
                        <span>Título</span>
                        <input v-model="post.title" class="input title-input" placeholder="Sobre o que é este conteúdo?" />
                    </label>
                    <label class="field">
                        <span>Descrição / legenda</span>
                        <textarea v-model="post.description" class="textarea" rows="5"></textarea>
                    </label>
                    <div class="grid-2">
                        <div class="field">
                            <span>Tags</span>
                            <TagInput v-model="post.tags" label="Tags" placeholder="palavra-chave, Enter para adicionar" />
                        </div>
                        <div class="field">
                            <span>Hashtags</span>
                            <TagInput v-model="post.hashtags" label="Hashtags" prefix="#" placeholder="#hashtag" />
                        </div>
                    </div>
                </section>

                <section class="panel block">
                    <div class="field">
                        <div class="row between">
                            <span class="label">Roteiro</span>
                            <button type="button" class="btn ghost sm" @click="scriptPreview = !scriptPreview">
                                <component :is="scriptPreview ? Pencil : Eye" />
                                {{ scriptPreview ? "Editar" : "Visualizar" }}
                            </button>
                        </div>
                        <MarkdownView v-if="scriptPreview" :source="post.script || '_Sem roteiro ainda._'" class="script-view" />
                        <textarea
                            v-else
                            v-model="post.script"
                            class="textarea script"
                            rows="14"
                            placeholder="Escreva o roteiro ou gere um com a IA ao lado. Markdown é aceito."
                        ></textarea>
                    </div>
                    <label class="field">
                        <span>Anotações</span>
                        <textarea
                            v-model="post.notes"
                            class="textarea"
                            rows="3"
                            placeholder="Referências, locação, cenas, links…"
                        ></textarea>
                    </label>
                </section>
            </form>

            <aside class="side">
                <section class="panel ai">
                    <div class="ai-head">
                        <Sparkles :size="17" />
                        <h2>Copiloto</h2>
                    </div>
                    <div class="tabs" role="tablist">
                        <button role="tab" :aria-selected="aiTab === 'script'" @click="aiTab = 'script'">Roteiro</button>
                        <button role="tab" :aria-selected="aiTab === 'metadata'" @click="aiTab = 'metadata'">
                            Metadados
                        </button>
                    </div>

                    <div v-if="aiTab === 'script'" class="ai-body">
                        <label class="field">
                            <span>Duração</span>
                            <select v-model="ai.duration" class="select">
                                <option v-for="d in durations" :key="d.v" :value="d.v">{{ d.l }}</option>
                            </select>
                        </label>
                        <label class="field">
                            <span>Instruções (opcional)</span>
                            <textarea
                                v-model="ai.scriptInstructions"
                                class="textarea"
                                rows="2"
                                placeholder="Ex.: comece com uma pergunta polêmica"
                            ></textarea>
                        </label>
                        <button v-if="!ai.generatingScript" class="btn accent" @click="generateScript">
                            <Wand2 /> {{ post.script ? "Reescrever roteiro" : "Gerar roteiro" }}
                        </button>
                        <button v-else class="btn" @click="stopScript"><Square /> Parar</button>

                        <div v-if="ai.scriptDraft || ai.generatingScript" class="draft" :class="{ live: ai.generatingScript }">
                            <MarkdownView :source="ai.scriptDraft || 'Escrevendo…'" />
                        </div>
                        <div v-if="ai.scriptDraft && !ai.generatingScript" class="row">
                            <button class="btn primary sm" @click="useScript">Usar no post</button>
                            <button class="btn ghost sm" @click="ai.scriptDraft = ''">Descartar</button>
                        </div>
                    </div>

                    <div v-else class="ai-body">
                        <p class="muted small">
                            Usa o título, o roteiro e as plataformas selecionadas para sugerir título, descrição, tags e
                            hashtags.
                        </p>
                        <label class="field">
                            <span>Instruções (opcional)</span>
                            <textarea
                                v-model="ai.metaInstructions"
                                class="textarea"
                                rows="2"
                                placeholder="Ex.: foque em SEO para iniciantes"
                            ></textarea>
                        </label>
                        <button class="btn accent" :disabled="ai.generatingMeta" @click="generateMetadata">
                            <Wand2 /> {{ ai.generatingMeta ? "Gerando…" : "Gerar metadados" }}
                        </button>

                        <div v-if="ai.meta" class="meta-result">
                            <div class="meta-item">
                                <div class="row between">
                                    <strong>Título</strong>
                                    <button class="btn ghost sm" @click="applyMeta('title')">Usar</button>
                                </div>
                                <p>{{ ai.meta.title }}</p>
                            </div>
                            <div class="meta-item">
                                <div class="row between">
                                    <strong>Descrição</strong>
                                    <button class="btn ghost sm" @click="applyMeta('description')">Usar</button>
                                </div>
                                <p class="pre">{{ ai.meta.description }}</p>
                            </div>
                            <div class="meta-item">
                                <div class="row between">
                                    <strong>Tags</strong>
                                    <button class="btn ghost sm" @click="applyMeta('tags')">Usar</button>
                                </div>
                                <div class="chips">
                                    <span v-for="t in ai.meta.tags" :key="t" class="chip">{{ t }}</span>
                                </div>
                            </div>
                            <div class="meta-item">
                                <div class="row between">
                                    <strong>Hashtags</strong>
                                    <button class="btn ghost sm" @click="applyMeta('hashtags')">Usar</button>
                                </div>
                                <div class="chips">
                                    <span v-for="t in ai.meta.hashtags" :key="t" class="chip">{{ t }}</span>
                                </div>
                            </div>
                            <button class="btn primary sm" @click="applyMeta('all')">Usar tudo</button>
                        </div>
                    </div>
                </section>

                <section v-if="!isNew && post.platforms.length" class="panel block">
                    <h3>Publicação</h3>
                    <div v-for="pp in post.platforms" :key="pp.platform" class="pub">
                        <div class="row">
                            <PlatformIcon :id="pp.platform" />
                            <span class="grow">{{ platformInfo(pp.platform).name }}</span>
                            <select
                                class="select sm-select"
                                :value="pp.status"
                                :aria-label="`Situação em ${platformInfo(pp.platform).name}`"
                                @change="setPlatformStatus(pp.platform, ($event.target as HTMLSelectElement).value, pp.url)"
                            >
                                <option v-for="s in PLATFORM_STATUS" :key="s.id" :value="s.id">{{ s.label }}</option>
                            </select>
                        </div>
                        <input
                            v-if="pp.status === 'published'"
                            class="input"
                            :value="pp.url"
                            placeholder="Link da publicação"
                            :aria-label="`Link em ${platformInfo(pp.platform).name}`"
                            @change="setPlatformStatus(pp.platform, pp.status, ($event.target as HTMLInputElement).value)"
                        />
                    </div>
                </section>

                <section v-if="history.length" class="panel block">
                    <h3>Histórico de datas</h3>
                    <ol class="history">
                        <li v-for="h in history" :key="h.id">
                            <span>{{ formatSchedule(h.fromAt) }} → <strong>{{ formatSchedule(h.toAt) }}</strong></span>
                            <small v-if="h.reason" class="muted">{{ h.reason }}</small>
                        </li>
                    </ol>
                </section>
            </aside>
        </div>

        <PostponeDialog v-if="postponing" :post="post" @close="postponing = false" @done="onPostponed" />
    </div>
</template>

<style scoped>
.editor {
    max-width: 1360px;
}

.editor .page-head h1 {
    font-size: 26px;
    max-width: 640px;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
}

.cols {
    display: grid;
    grid-template-columns: minmax(0, 1fr) 380px;
    gap: 20px;
    align-items: start;
}

.form {
    display: flex;
    flex-direction: column;
    gap: 16px;
}

.block {
    display: flex;
    flex-direction: column;
    gap: 14px;
    padding: 18px;
}

.grid-2,
.grid-3 {
    display: grid;
    gap: 12px;
}

.grid-2 {
    grid-template-columns: 1fr 1fr;
}

.grid-3 {
    grid-template-columns: 1fr 1fr 1fr;
}

.stages {
    display: grid;
    grid-template-columns: repeat(6, 1fr);
    gap: 3px;
    margin-top: -4px;
}

.stages span {
    height: 4px;
    border-radius: 2px;
    opacity: 0.18;
}

.stages span.reached {
    opacity: 1;
}

.title-input {
    height: 42px;
    font-size: 17px;
    font-weight: 600;
}

.between {
    justify-content: space-between;
}

.label {
    font-size: 12.5px;
    font-weight: 600;
    color: var(--ink-2);
}

.script {
    font-size: 14.5px;
    min-height: 260px;
}

.script-view {
    min-height: 120px;
    padding: 12px 14px;
    border: 1px solid var(--line);
    border-radius: var(--r-sm);
}

.side {
    display: flex;
    flex-direction: column;
    gap: 16px;
    position: sticky;
    top: 0;
}

.ai {
    overflow: hidden;
}

.ai-head {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 16px 18px 10px;
    color: var(--cobalt);
}

.ai-head h2 {
    color: var(--ink);
    font-size: 17px;
}

.tabs {
    display: flex;
    gap: 2px;
    padding: 0 18px;
    border-bottom: 1px solid var(--line);
}

.tabs button {
    padding: 8px 10px;
    border: 0;
    border-bottom: 2px solid transparent;
    background: transparent;
    color: var(--muted);
    font: inherit;
    font-weight: 600;
    cursor: pointer;
    margin-bottom: -1px;
}

.tabs button[aria-selected="true"] {
    color: var(--ink);
    border-bottom-color: var(--ink);
}

.ai-body {
    display: flex;
    flex-direction: column;
    gap: 12px;
    padding: 16px 18px 18px;
}

.small {
    font-size: 12.5px;
}

.draft {
    max-height: 420px;
    overflow-y: auto;
    padding: 12px 14px;
    border-radius: var(--r-sm);
    background: var(--surface-2);
    font-size: 13.5px;
}

/* While the model writes, the draft edge shows the REC light. */
.draft.live {
    box-shadow: inset 3px 0 0 var(--rec);
}

.meta-result {
    display: flex;
    flex-direction: column;
    gap: 12px;
}

.meta-item {
    display: flex;
    flex-direction: column;
    gap: 4px;
    padding-bottom: 10px;
    border-bottom: 1px solid var(--line);
}

.pre {
    white-space: pre-wrap;
    font-size: 13px;
    max-height: 180px;
    overflow-y: auto;
}

.chips {
    display: flex;
    flex-wrap: wrap;
    gap: 4px;
}

.pub {
    display: flex;
    flex-direction: column;
    gap: 6px;
}

.sm-select {
    width: auto;
    height: 30px;
    padding: 2px 8px;
    font-size: 13px;
}

.history {
    margin: 0;
    padding: 0;
    list-style: none;
    display: flex;
    flex-direction: column;
    gap: 8px;
    font-size: 13px;
}

.history li {
    display: flex;
    flex-direction: column;
}

@media (max-width: 1100px) {
    .cols {
        grid-template-columns: 1fr;
    }

    .side {
        position: static;
    }
}
</style>
