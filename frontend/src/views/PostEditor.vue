<script setup lang="ts">
import {
    ArrowLeft,
    BookOpen,
    CalendarClock,
    Copy,
    Eye,
    History,
    Pencil,
    Send,
    Sparkles,
    Square,
    Trash2,
    Wand2,
} from "@lucide/vue"
import {
    computed,
    onBeforeUnmount,
    onMounted,
    reactive,
    ref,
    watch,
    watchEffect,
} from "vue"
import { onBeforeRouteLeave, useRoute, useRouter } from "vue-router"
import { Events } from "@wailsio/runtime"

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
    type SkillInfo,
} from "@/lib/api"
import {
    STATUS_COLORS,
    platformInfo,
    profileById,
    store,
    themesOf,
} from "@/lib/store"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import PlatformPicker from "@/components/PlatformPicker.vue"
import PostponeDialog from "@/components/PostponeDialog.vue"
import MarkdownView from "@/components/MarkdownView.vue"
import PlatformIcon from "@/components/PlatformIcon.vue"
import PageHeader from "@/components/PageHeader.vue"
import { Textarea } from "@/components/ui/textarea"
import AppSelect from "@/components/AppSelect.vue"
import TagInput from "@/components/TagInput.vue"
import { Button } from "@/components/ui/button"
import { toast, toastError } from "@/lib/toast"
import { Badge } from "@/components/ui/badge"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { confirmDialog } from "@/lib/confirm"
import { formatSchedule } from "@/lib/dates"
import { chat } from "@/lib/chat"
import { cn } from "@/lib/utils"

const props = defineProps<{ id: string }>()
const route = useRoute()
const router = useRouter()

const isNew = computed(() => props.id === "new")

function blank(): PostN {
    const profileId =
        Number(route.query.profile) ||
        store.activeProfileId ||
        store.profiles[0]?.id ||
        0
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
const savedJSON = ref(JSON.stringify(editable(post.value)))
const history = ref<Postponement[]>([])
const saving = ref(false)
const postponing = ref(false)
const scriptPreview = ref(false)
let leaving = false

const profile = computed(() => profileById(post.value.profileId))
const availablePlatforms = computed(
    () => profile.value?.platforms.map((p) => p.platform) ?? [],
)
const selectedPlatforms = computed({
    get: () => post.value.platforms.map((p) => p.platform),
    set: (ids: string[]) => {
        const old = new Map(post.value.platforms.map((p) => [p.platform, p]))
        post.value.platforms = ids.map(
            (id) =>
                old.get(id) ?? {
                    platform: id,
                    status: "pending",
                    url: "",
                    publishedAt: null,
                },
        )
    },
})
const dirty = computed(
    () => JSON.stringify(editable(post.value)) !== savedJSON.value,
)
const statusIndex = computed(() =>
    store.catalog.statuses.findIndex((s) => s.id === post.value.status),
)

const profileOptions = computed(() =>
    store.profiles.map((p) => ({ value: p.id, label: p.name })),
)
const themeOptions = computed(() => [
    { value: null, label: "Sem tema" },
    ...themesOf(post.value.profileId).map((t) => ({
        value: t.id,
        label: t.name,
    })),
])
const formatOptions = computed(() =>
    store.catalog.formats.map((f) => ({ value: f.id, label: f.label })),
)
const statusOptions = computed(() =>
    store.catalog.statuses.map((s) => ({ value: s.id, label: s.label })),
)

function editable(p: PostN): unknown {
    const {
        platforms,
        postponeCount: _c,
        updatedAt: _u,
        createdAt: _cr,
        publishedAt: _p,
        ...rest
    } = p
    return { ...rest, platforms: platforms.map((x) => x.platform) }
}

function markSaved(): void {
    savedJSON.value = JSON.stringify(editable(post.value))
}

/** Updates the saved snapshot for fields changed directly on the server. */
function patchSaved(patch: Partial<PostN>): void {
    savedJSON.value = JSON.stringify({
        ...JSON.parse(savedJSON.value),
        ...patch,
    })
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
        history.value =
            (await PostService.ListPostponements(post.value.id)) ?? []
    } catch (e) {
        toastError(e)
        leaving = true
        router.replace("/")
    }
}

// Switching profile keeps only the platforms the new profile has.
watch(
    () => post.value.profileId,
    (id, old) => {
        if (!old || id === old) return
        const allowed = new Set(availablePlatforms.value)
        selectedPlatforms.value = selectedPlatforms.value.filter((p) =>
            allowed.has(p),
        )
        if (!selectedPlatforms.value.length)
            selectedPlatforms.value = [...allowed]
        if (
            post.value.themeId &&
            !themesOf(id).some((t) => t.id === post.value.themeId)
        )
            post.value.themeId = null
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
    const ok = await confirmDialog({
        title: "Excluir post",
        description: `"${post.value.title}" será excluído com o roteiro, os metadados e o histórico. Isso não pode ser desfeito.`,
        confirmText: "Excluir post",
        destructive: true,
    })
    if (!ok) return
    try {
        await PostService.DeletePost(post.value.id)
        leaving = true
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
    patchSaved({ scheduledAt: updated.scheduledAt })
    history.value = (await PostService.ListPostponements(post.value.id)) ?? []
}

async function setPlatformStatus(
    platform: string,
    status: string,
    url: string,
): Promise<void> {
    try {
        const updated = normPost(
            await PostService.SetPlatformStatus(
                post.value.id,
                platform,
                status,
                url,
            ),
        )
        post.value.platforms = updated.platforms
        if (updated.status !== post.value.status) {
            post.value.status = updated.status
            post.value.publishedAt = updated.publishedAt
            patchSaved({ status: updated.status })
            if (updated.status === "published")
                toast("Publicado em todas as plataformas.")
        }
    } catch (e) {
        toastError(e)
    }
}

// The chat sees the post being edited, unsaved changes included.
watchEffect(() => {
    chat.post = post.value
})
onBeforeUnmount(() => {
    chat.post = null
})

// Skills each generation will follow, shown before generating.
const skillsFor = reactive({
    script: [] as SkillInfo[],
    metadata: [] as SkillInfo[],
})
watch(
    () => [
        post.value.profileId,
        post.value.format,
        selectedPlatforms.value.join(","),
    ],
    async () => {
        const p = post.value
        try {
            const [script, metadata] = await Promise.all([
                AIService.MatchingSkills(
                    "script",
                    p.profileId,
                    selectedPlatforms.value,
                    p.format,
                ),
                AIService.MatchingSkills(
                    "metadata",
                    p.profileId,
                    selectedPlatforms.value,
                    p.format,
                ),
            ])
            skillsFor.script = script ?? []
            skillsFor.metadata = metadata ?? []
        } catch {
            skillsFor.script = []
            skillsFor.metadata = []
        }
    },
    { immediate: true },
)

// ── AI ────────────────────────────────────────────────────────────────────
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

const offDelta = Events.On(
    AiDelta,
    (ev: { data: { requestId: string; delta: string } }) => {
        if (ev.data.requestId === requestId) ai.scriptDraft += ev.data.delta
    },
)
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

function needProvider(): boolean {
    if (store.providers.length) return false
    toast(
        "Configure um provedor de IA em Configurações para usar esta função.",
        "error",
    )
    return true
}

async function generateScript(): Promise<void> {
    if (needProvider()) return
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

function useScript(): void {
    post.value.script = ai.scriptDraft
    ai.scriptDraft = ""
    scriptPreview.value = true
    toast("Roteiro aplicado. Salve para guardar.")
}

async function generateMetadata(): Promise<void> {
    if (needProvider()) return
    ai.generatingMeta = true
    try {
        const m = await AIService.GenerateMetadata(
            aiRequest(ai.metaInstructions),
        )
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
    if (field === "description" || field === "all")
        post.value.description = m.description
    if (field === "tags" || field === "all")
        post.value.tags = [...(m.tags ?? [])]
    if (field === "hashtags" || field === "all")
        post.value.hashtags = [...(m.hashtags ?? [])]
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
onBeforeRouteLeave(async () => {
    if (leaving || !dirty.value) return true
    return confirmDialog({
        title: "Sair sem salvar?",
        description:
            "As alterações deste post ainda não foram salvas e serão perdidas.",
        confirmText: "Sair sem salvar",
        destructive: true,
    })
})

const durations = [
    { value: 15, label: "15 segundos" },
    { value: 30, label: "30 segundos" },
    { value: 60, label: "1 minuto" },
    { value: 90, label: "1 minuto e meio" },
    { value: 180, label: "3 minutos" },
    { value: 480, label: "8 minutos" },
    { value: 900, label: "15 minutos" },
]
const PLATFORM_STATUS = [
    { value: "pending", label: "Pendente" },
    { value: "published", label: "Publicado" },
    { value: "skipped", label: "Não vai sair" },
]
const card = "rounded-xl border border-border/60 bg-card p-6 shadow-xs"
</script>

<template>
    <div class="flex h-full flex-col">
        <PageHeader :title="isNew ? 'Novo post' : post.title || 'Sem título'">
            <template #leading>
                <Button
                    variant="ghost"
                    size="icon"
                    aria-label="Voltar"
                    @click="router.back()"
                    ><ArrowLeft
                /></Button>
            </template>
            <template v-if="!isNew" #description>
                {{ formatSchedule(post.scheduledAt) }}
                <template v-if="post.postponeCount">
                    · adiado {{ post.postponeCount }}x</template
                >
            </template>
            <template #actions>
                <template v-if="!isNew">
                    <Button
                        variant="ghost"
                        size="icon"
                        title="Duplicar"
                        aria-label="Duplicar"
                        @click="duplicate"
                    >
                        <Copy />
                    </Button>
                    <Button
                        variant="ghost"
                        size="icon"
                        class="text-destructive hover:text-destructive"
                        title="Excluir"
                        aria-label="Excluir"
                        @click="remove"
                    >
                        <Trash2 />
                    </Button>
                    <Button
                        variant="outline"
                        :disabled="post.status === 'published'"
                        @click="postponing = true"
                    >
                        <CalendarClock /> Adiar
                    </Button>
                </template>
                <Button :disabled="saving || (!dirty && !isNew)" @click="save">
                    {{ dirty || isNew ? "Salvar" : "Salvo" }}
                </Button>
            </template>
        </PageHeader>

        <div class="flex-1 overflow-y-auto px-8 py-6">
            <div
                v-if="!store.profiles.length"
                class="flex flex-col items-center justify-center gap-3 rounded-xl border border-dashed border-border/60 bg-card/40 p-8 py-16 text-center"
            >
                <h2 class="font-heading text-lg font-bold">
                    Crie um perfil primeiro
                </h2>
                <p class="max-w-md text-sm text-muted-foreground">
                    Todo post pertence a um perfil, e é nele que você diz em
                    quais plataformas publica.
                </p>
                <Button as-child
                    ><RouterLink to="/profiles?new=1"
                        >Criar perfil</RouterLink
                    ></Button
                >
            </div>

            <div
                v-else
                class="mx-auto grid max-w-7xl items-start gap-6 lg:grid-cols-[minmax(0,1fr)_380px]"
            >
                <form class="space-y-6" @submit.prevent="save">
                    <section :class="cn(card, 'space-y-5')">
                        <div class="grid gap-4 md:grid-cols-3">
                            <div class="space-y-2">
                                <Label>Perfil</Label>
                                <AppSelect
                                    v-model="post.profileId"
                                    :options="profileOptions"
                                    label="Perfil"
                                />
                            </div>
                            <div class="space-y-2">
                                <Label>Tema</Label>
                                <AppSelect
                                    v-model="post.themeId"
                                    :options="themeOptions"
                                    label="Tema"
                                />
                            </div>
                            <div class="space-y-2">
                                <Label>Formato</Label>
                                <AppSelect
                                    v-model="post.format"
                                    :options="formatOptions"
                                    label="Formato"
                                />
                            </div>
                        </div>
                        <div class="grid gap-4 md:grid-cols-2">
                            <div class="space-y-2">
                                <Label>Etapa</Label>
                                <AppSelect
                                    v-model="post.status"
                                    :options="statusOptions"
                                    label="Etapa"
                                />
                            </div>
                            <div class="space-y-2">
                                <Label for="scheduled"
                                    >Data de publicação</Label
                                >
                                <Input
                                    id="scheduled"
                                    type="datetime-local"
                                    :model-value="post.scheduledAt ?? ''"
                                    @update:model-value="
                                        (v) =>
                                            (post.scheduledAt =
                                                String(v) || null)
                                    "
                                />
                            </div>
                        </div>
                        <div class="grid grid-cols-6 gap-1" aria-hidden="true">
                            <span
                                v-for="(s, i) in store.catalog.statuses"
                                :key="s.id"
                                class="h-1.5 rounded-full transition-opacity"
                                :style="{
                                    background: STATUS_COLORS[s.id],
                                    opacity: i <= statusIndex ? 1 : 0.18,
                                }"
                            ></span>
                        </div>
                        <div class="space-y-2">
                            <Label>Publicar em</Label>
                            <PlatformPicker
                                v-if="availablePlatforms.length"
                                v-model="selectedPlatforms"
                                :available="availablePlatforms"
                            />
                            <p v-else class="text-sm text-muted-foreground">
                                Este perfil ainda não tem plataformas.
                                <RouterLink
                                    to="/profiles"
                                    class="font-medium text-primary underline-offset-4 hover:underline"
                                >
                                    Adicionar plataformas ao perfil
                                </RouterLink>
                            </p>
                        </div>
                    </section>

                    <section :class="cn(card, 'space-y-5')">
                        <div class="space-y-2">
                            <Label for="title">Título</Label>
                            <Input
                                id="title"
                                v-model="post.title"
                                class="h-11 text-base font-semibold md:text-base"
                                placeholder="Sobre o que é este conteúdo?"
                            />
                        </div>
                        <div class="space-y-2">
                            <Label for="description">Descrição / legenda</Label>
                            <Textarea
                                id="description"
                                v-model="post.description"
                                class="min-h-28"
                            />
                        </div>
                        <div class="grid gap-4 md:grid-cols-2">
                            <div class="space-y-2">
                                <Label>Tags</Label>
                                <TagInput
                                    v-model="post.tags"
                                    label="Nova tag"
                                    placeholder="palavra-chave"
                                />
                            </div>
                            <div class="space-y-2">
                                <Label>Hashtags</Label>
                                <TagInput
                                    v-model="post.hashtags"
                                    label="Nova hashtag"
                                    prefix="#"
                                    placeholder="#hashtag"
                                />
                            </div>
                        </div>
                    </section>

                    <section :class="cn(card, 'space-y-5')">
                        <div class="space-y-2">
                            <div class="flex items-center justify-between">
                                <Label for="script">Roteiro</Label>
                                <Button
                                    type="button"
                                    variant="ghost"
                                    size="sm"
                                    @click="scriptPreview = !scriptPreview"
                                >
                                    <component
                                        :is="scriptPreview ? Pencil : Eye"
                                    />
                                    {{
                                        scriptPreview ? "Editar" : "Visualizar"
                                    }}
                                </Button>
                            </div>
                            <MarkdownView
                                v-if="scriptPreview"
                                :source="post.script || '_Sem roteiro ainda._'"
                                class="min-h-32 rounded-md border border-input px-4 py-3"
                            />
                            <Textarea
                                v-else
                                id="script"
                                v-model="post.script"
                                class="min-h-72 text-[15px] md:text-[15px]"
                                placeholder="Escreva o roteiro ou gere um com a IA ao lado. Markdown é aceito."
                            />
                        </div>
                        <div class="space-y-2">
                            <Label for="notes">Anotações</Label>
                            <Textarea
                                id="notes"
                                v-model="post.notes"
                                placeholder="Referências, locação, cenas, links…"
                            />
                        </div>
                    </section>
                </form>

                <aside class="space-y-6 lg:sticky lg:top-0">
                    <section
                        class="rounded-xl border border-primary/30 bg-card p-5 shadow-xs"
                    >
                        <div class="mb-4 flex items-center gap-2">
                            <Sparkles class="size-5 text-primary" />
                            <h2 class="font-heading text-lg font-bold">
                                Copiloto
                            </h2>
                        </div>
                        <Tabs default-value="script">
                            <TabsList class="mb-4 w-full">
                                <TabsTrigger value="script"
                                    >Roteiro</TabsTrigger
                                >
                                <TabsTrigger value="metadata"
                                    >Metadados</TabsTrigger
                                >
                            </TabsList>

                            <TabsContent value="script" class="space-y-4">
                                <div
                                    v-if="skillsFor.script.length"
                                    class="flex flex-wrap items-center gap-1.5 text-xs text-muted-foreground"
                                >
                                    <BookOpen class="size-3.5" /> Seguindo:
                                    <Badge
                                        v-for="s in skillsFor.script"
                                        :key="s.id"
                                        variant="secondary"
                                        :title="s.description"
                                        >{{ s.name }}</Badge
                                    >
                                </div>
                                <div class="space-y-2">
                                    <Label>Duração</Label>
                                    <AppSelect
                                        v-model="ai.duration"
                                        :options="durations"
                                        label="Duração"
                                    />
                                </div>
                                <div class="space-y-2">
                                    <Label for="script-instructions"
                                        >Instruções (opcional)</Label
                                    >
                                    <Textarea
                                        id="script-instructions"
                                        v-model="ai.scriptInstructions"
                                        placeholder="Ex.: comece com uma pergunta polêmica"
                                    />
                                </div>
                                <Button
                                    v-if="!ai.generatingScript"
                                    class="w-full"
                                    @click="generateScript"
                                >
                                    <Wand2 />
                                    {{
                                        post.script
                                            ? "Reescrever roteiro"
                                            : "Gerar roteiro"
                                    }}
                                </Button>
                                <Button
                                    v-else
                                    variant="outline"
                                    class="w-full"
                                    @click="scriptCall?.cancel()"
                                >
                                    <Square /> Parar
                                </Button>

                                <div
                                    v-if="ai.scriptDraft || ai.generatingScript"
                                    :class="
                                        cn(
                                            'max-h-[420px] overflow-y-auto rounded-lg bg-muted p-4',
                                            ai.generatingScript &&
                                                'animate-pulse [animation-duration:2.5s]',
                                        )
                                    "
                                >
                                    <MarkdownView
                                        :source="
                                            ai.scriptDraft || 'Escrevendo…'
                                        "
                                    />
                                </div>
                                <div
                                    v-if="
                                        ai.scriptDraft && !ai.generatingScript
                                    "
                                    class="flex gap-2"
                                >
                                    <Button size="sm" @click="useScript"
                                        >Usar no post</Button
                                    >
                                    <Button
                                        size="sm"
                                        variant="ghost"
                                        @click="ai.scriptDraft = ''"
                                        >Descartar</Button
                                    >
                                </div>
                            </TabsContent>

                            <TabsContent value="metadata" class="space-y-4">
                                <div
                                    v-if="skillsFor.metadata.length"
                                    class="flex flex-wrap items-center gap-1.5 text-xs text-muted-foreground"
                                >
                                    <BookOpen class="size-3.5" /> Seguindo:
                                    <Badge
                                        v-for="s in skillsFor.metadata"
                                        :key="s.id"
                                        variant="secondary"
                                        :title="s.description"
                                        >{{ s.name }}</Badge
                                    >
                                </div>
                                <p class="text-sm text-muted-foreground">
                                    Usa o título, o roteiro e as plataformas
                                    selecionadas para sugerir título, descrição,
                                    tags e hashtags.
                                </p>
                                <div class="space-y-2">
                                    <Label for="meta-instructions"
                                        >Instruções (opcional)</Label
                                    >
                                    <Textarea
                                        id="meta-instructions"
                                        v-model="ai.metaInstructions"
                                        placeholder="Ex.: foque em SEO para iniciantes"
                                    />
                                </div>
                                <Button
                                    class="w-full"
                                    :disabled="ai.generatingMeta"
                                    @click="generateMetadata"
                                >
                                    <Wand2 />
                                    {{
                                        ai.generatingMeta
                                            ? "Gerando…"
                                            : "Gerar metadados"
                                    }}
                                </Button>

                                <div v-if="ai.meta" class="space-y-3">
                                    <div
                                        class="space-y-1 border-b border-border/60 pb-3"
                                    >
                                        <div
                                            class="flex items-center justify-between"
                                        >
                                            <span class="text-sm font-medium"
                                                >Título</span
                                            >
                                            <Button
                                                size="xs"
                                                variant="ghost"
                                                @click="applyMeta('title')"
                                                >Usar</Button
                                            >
                                        </div>
                                        <p class="text-sm">
                                            {{ ai.meta.title }}
                                        </p>
                                    </div>
                                    <div
                                        class="space-y-1 border-b border-border/60 pb-3"
                                    >
                                        <div
                                            class="flex items-center justify-between"
                                        >
                                            <span class="text-sm font-medium"
                                                >Descrição</span
                                            >
                                            <Button
                                                size="xs"
                                                variant="ghost"
                                                @click="
                                                    applyMeta('description')
                                                "
                                                >Usar</Button
                                            >
                                        </div>
                                        <p
                                            class="max-h-44 overflow-y-auto text-sm whitespace-pre-wrap"
                                        >
                                            {{ ai.meta.description }}
                                        </p>
                                    </div>
                                    <div
                                        class="space-y-1.5 border-b border-border/60 pb-3"
                                    >
                                        <div
                                            class="flex items-center justify-between"
                                        >
                                            <span class="text-sm font-medium"
                                                >Tags</span
                                            >
                                            <Button
                                                size="xs"
                                                variant="ghost"
                                                @click="applyMeta('tags')"
                                                >Usar</Button
                                            >
                                        </div>
                                        <div class="flex flex-wrap gap-1">
                                            <Badge
                                                v-for="t in ai.meta.tags"
                                                :key="t"
                                                variant="secondary"
                                                >{{ t }}</Badge
                                            >
                                        </div>
                                    </div>
                                    <div
                                        class="space-y-1.5 border-b border-border/60 pb-3"
                                    >
                                        <div
                                            class="flex items-center justify-between"
                                        >
                                            <span class="text-sm font-medium"
                                                >Hashtags</span
                                            >
                                            <Button
                                                size="xs"
                                                variant="ghost"
                                                @click="applyMeta('hashtags')"
                                                >Usar</Button
                                            >
                                        </div>
                                        <div class="flex flex-wrap gap-1">
                                            <Badge
                                                v-for="t in ai.meta.hashtags"
                                                :key="t"
                                                variant="secondary"
                                                >{{ t }}</Badge
                                            >
                                        </div>
                                    </div>
                                    <Button
                                        size="sm"
                                        class="w-full"
                                        @click="applyMeta('all')"
                                        >Usar tudo</Button
                                    >
                                </div>
                            </TabsContent>
                        </Tabs>
                    </section>

                    <section
                        v-if="!isNew && post.platforms.length"
                        :class="cn(card, 'space-y-4 p-5')"
                    >
                        <h2
                            class="flex items-center gap-2 font-heading text-base font-semibold"
                        >
                            <Send class="size-4" /> Publicação
                        </h2>
                        <div
                            v-for="pp in post.platforms"
                            :key="pp.platform"
                            class="space-y-2"
                        >
                            <div class="flex items-center gap-2">
                                <PlatformIcon :id="pp.platform" />
                                <span class="flex-1 text-sm font-medium">{{
                                    platformInfo(pp.platform).name
                                }}</span>
                                <AppSelect
                                    :model-value="pp.status"
                                    :options="PLATFORM_STATUS"
                                    size="sm"
                                    class="w-36"
                                    :label="`Situação em ${platformInfo(pp.platform).name}`"
                                    @update:model-value="
                                        (v) =>
                                            setPlatformStatus(
                                                pp.platform,
                                                String(v),
                                                pp.url,
                                            )
                                    "
                                />
                            </div>
                            <Input
                                v-if="pp.status === 'published'"
                                :model-value="pp.url"
                                placeholder="Link da publicação"
                                :aria-label="`Link em ${platformInfo(pp.platform).name}`"
                                @change="
                                    (e: Event) =>
                                        setPlatformStatus(
                                            pp.platform,
                                            pp.status,
                                            (e.target as HTMLInputElement)
                                                .value,
                                        )
                                "
                            />
                        </div>
                    </section>

                    <section
                        v-if="history.length"
                        :class="cn(card, 'space-y-3 p-5')"
                    >
                        <h2
                            class="flex items-center gap-2 font-heading text-base font-semibold"
                        >
                            <History class="size-4" /> Histórico de datas
                        </h2>
                        <ol class="space-y-2 text-sm">
                            <li v-for="h in history" :key="h.id">
                                {{ formatSchedule(h.fromAt) }} →
                                <strong>{{ formatSchedule(h.toAt) }}</strong>
                                <span
                                    v-if="h.reason"
                                    class="block text-xs text-muted-foreground"
                                    >{{ h.reason }}</span
                                >
                            </li>
                        </ol>
                    </section>
                </aside>
            </div>
        </div>

        <PostponeDialog
            v-if="postponing"
            :post="post"
            @close="postponing = false"
            @done="onPostponed"
        />
    </div>
</template>
