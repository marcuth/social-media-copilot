<script setup lang="ts">
import { Plus, Tags, Trash2, Users } from "@lucide/vue"
import { computed, reactive, ref } from "vue"
import { useRoute } from "vue-router"

import {
    ProfileService,
    normProfile,
    type Profile,
    type ProfilePlatform,
    type Theme,
} from "@/lib/api"
import {
    PROFILE_COLORS,
    refreshProfiles,
    setActiveProfile,
    store,
    themesOf,
} from "@/lib/store"
import PlatformIcon from "@/components/PlatformIcon.vue"
import PageHeader from "@/components/PageHeader.vue"
import { Checkbox } from "@/components/ui/checkbox"
import { Textarea } from "@/components/ui/textarea"
import AppSelect from "@/components/AppSelect.vue"
import { Button } from "@/components/ui/button"
import { toast, toastError } from "@/lib/toast"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { confirmDialog } from "@/lib/confirm"
import { cn } from "@/lib/utils"

type Draft = Profile & { platforms: ProfilePlatform[] }

const route = useRoute()

function blank(): Draft {
    return {
        id: 0,
        name: "",
        description: "",
        niche: "",
        audience: "",
        tone: "",
        language: "pt-BR",
        color: PROFILE_COLORS[store.profiles.length % PROFILE_COLORS.length],
        platforms: [],
        createdAt: "",
        updatedAt: "",
    }
}

function load(id: number | null): Draft {
    const p = store.profiles.find((x) => x.id === id)
    return p ? JSON.parse(JSON.stringify(normProfile(p))) : blank()
}

const initial = route.query.new
    ? null
    : (store.activeProfileId ?? store.profiles[0]?.id ?? null)
const selectedId = ref<number | null>(initial)
const draft = ref<Draft>(load(initial))
const saving = ref(false)

function select(id: number | null): void {
    selectedId.value = id
    draft.value = load(id)
}

const handles = computed(() =>
    Object.fromEntries(draft.value.platforms.map((p) => [p.platform, p])),
)

function togglePlatform(id: string): void {
    const list = draft.value.platforms
    const i = list.findIndex((p) => p.platform === id)
    if (i >= 0) list.splice(i, 1)
    else list.push({ platform: id, handle: "" })
}

async function save(): Promise<void> {
    saving.value = true
    try {
        const wasNew = !draft.value.id
        const saved = await ProfileService.SaveProfile(draft.value)
        await refreshProfiles()
        select(saved.id)
        toast(wasNew ? "Perfil criado." : "Perfil salvo.")
    } catch (e) {
        toastError(e)
    } finally {
        saving.value = false
    }
}

async function remove(): Promise<void> {
    const name = draft.value.name
    const ok = await confirmDialog({
        title: "Excluir perfil",
        description: `"${name}" será excluído junto com todos os temas e posts dele. Isso não pode ser desfeito.`,
        confirmText: "Excluir perfil",
        destructive: true,
    })
    if (!ok) return
    try {
        await ProfileService.DeleteProfile(draft.value.id)
        if (store.activeProfileId === draft.value.id) setActiveProfile(null)
        await refreshProfiles()
        select(store.profiles[0]?.id ?? null)
        toast(`Perfil "${name}" excluído.`)
    } catch (e) {
        toastError(e)
    }
}

const themes = computed(() => themesOf(draft.value.id))
const newTheme = reactive({
    name: "",
    description: "",
    color: PROFILE_COLORS[1],
})

async function addTheme(): Promise<void> {
    if (!newTheme.name.trim()) return
    try {
        await ProfileService.SaveTheme({
            id: 0,
            profileId: draft.value.id,
            ...newTheme,
            createdAt: "",
        })
        newTheme.name = ""
        newTheme.description = ""
        newTheme.color =
            PROFILE_COLORS[(themes.value.length + 2) % PROFILE_COLORS.length]
        await refreshProfiles()
    } catch (e) {
        toastError(e)
    }
}

async function updateTheme(t: Theme): Promise<void> {
    try {
        await ProfileService.SaveTheme(t)
        await refreshProfiles()
    } catch (e) {
        toastError(e)
    }
}

async function removeTheme(t: Theme): Promise<void> {
    const ok = await confirmDialog({
        title: "Excluir tema",
        description: `Os posts de "${t.name}" continuam existindo, só ficam sem tema.`,
        confirmText: "Excluir tema",
        destructive: true,
    })
    if (!ok) return
    try {
        await ProfileService.DeleteTheme(t.id)
        await refreshProfiles()
    } catch (e) {
        toastError(e)
    }
}

const LANGUAGES = [
    { value: "pt-BR", label: "Português (Brasil)" },
    { value: "pt-PT", label: "Português (Portugal)" },
    { value: "en-US", label: "Inglês" },
    { value: "es-ES", label: "Espanhol" },
]
const card = "rounded-xl border border-border/60 bg-card p-6 shadow-xs"
</script>

<template>
    <div class="flex h-full flex-col">
        <PageHeader
            title="Perfis e temas"
            description="Cada perfil é um canal ou marca, com as plataformas em que publica e os temas que aborda"
            :icon="Users"
        >
            <template #actions>
                <Button variant="outline" @click="select(null)"
                    ><Plus /> Novo perfil</Button
                >
            </template>
        </PageHeader>

        <div class="flex-1 overflow-y-auto px-8 py-6">
            <div
                class="mx-auto grid max-w-6xl items-start gap-6 lg:grid-cols-[260px_1fr]"
            >
                <nav class="flex flex-col gap-1" aria-label="Perfis">
                    <button
                        v-for="p in store.profiles"
                        :key="p.id"
                        :class="
                            cn(
                                'flex cursor-pointer items-center gap-3 rounded-xl border-2 px-3 py-2.5 text-left transition-colors',
                                p.id === selectedId
                                    ? 'border-primary bg-primary/10'
                                    : 'border-transparent hover:bg-accent/60',
                            )
                        "
                        @click="select(p.id)"
                    >
                        <span
                            class="flex size-8 shrink-0 items-center justify-center rounded-md text-sm font-bold text-white"
                            :style="{ background: p.color }"
                            >{{ p.name.charAt(0).toUpperCase() }}</span
                        >
                        <span class="min-w-0">
                            <span
                                class="block truncate text-sm font-semibold"
                                >{{ p.name }}</span
                            >
                            <span class="flex gap-1 pt-0.5">
                                <PlatformIcon
                                    v-for="pp in p.platforms"
                                    :id="pp.platform"
                                    :key="pp.platform"
                                    :size="12"
                                />
                                <span
                                    v-if="!p.platforms.length"
                                    class="text-xs text-muted-foreground"
                                    >Sem plataformas</span
                                >
                            </span>
                        </span>
                    </button>
                    <div
                        v-if="selectedId === null"
                        class="flex items-center gap-3 rounded-xl border-2 border-primary bg-primary/10 px-3 py-2.5"
                    >
                        <span
                            class="size-8 shrink-0 rounded-md"
                            :style="{ background: draft.color }"
                        ></span>
                        <span class="truncate text-sm font-semibold">{{
                            draft.name || "Novo perfil"
                        }}</span>
                    </div>
                </nav>

                <div class="space-y-6">
                    <form :class="cn(card, 'space-y-5')" @submit.prevent="save">
                        <div class="flex flex-wrap items-end gap-4">
                            <div class="min-w-64 flex-1 space-y-2">
                                <Label for="profile-name">Nome</Label>
                                <Input
                                    id="profile-name"
                                    v-model="draft.name"
                                    class="h-11 text-base font-semibold md:text-base"
                                    placeholder="Ex.: Canal de receitas"
                                />
                            </div>
                            <div class="space-y-2">
                                <Label>Cor</Label>
                                <div class="flex h-11 items-center gap-1.5">
                                    <button
                                        v-for="c in PROFILE_COLORS"
                                        :key="c"
                                        type="button"
                                        :class="
                                            cn(
                                                'size-6 cursor-pointer rounded-full transition-shadow',
                                                draft.color.toLowerCase() ===
                                                    c.toLowerCase() &&
                                                    'ring-2 ring-foreground ring-offset-2 ring-offset-card',
                                            )
                                        "
                                        :style="{ background: c }"
                                        :aria-label="`Cor ${c}`"
                                        @click="draft.color = c"
                                    ></button>
                                    <input
                                        v-model="draft.color"
                                        type="color"
                                        class="ml-1 size-8 cursor-pointer rounded-md border border-input bg-transparent p-0.5"
                                        aria-label="Outra cor"
                                    />
                                </div>
                            </div>
                        </div>

                        <div class="space-y-2">
                            <Label for="profile-description">Descrição</Label>
                            <Textarea
                                id="profile-description"
                                v-model="draft.description"
                                placeholder="O que este perfil publica e por quê"
                            />
                        </div>
                        <div class="grid gap-4 md:grid-cols-2">
                            <div class="space-y-2">
                                <Label for="niche">Nicho</Label>
                                <Input
                                    id="niche"
                                    v-model="draft.niche"
                                    placeholder="Ex.: culinária vegana rápida"
                                />
                            </div>
                            <div class="space-y-2">
                                <Label for="audience">Público</Label>
                                <Input
                                    id="audience"
                                    v-model="draft.audience"
                                    placeholder="Ex.: universitários que cozinham pouco"
                                />
                            </div>
                            <div class="space-y-2">
                                <Label for="tone">Tom de voz</Label>
                                <Input
                                    id="tone"
                                    v-model="draft.tone"
                                    placeholder="Ex.: descontraído, direto, com humor"
                                />
                            </div>
                            <div class="space-y-2">
                                <Label>Idioma do conteúdo</Label>
                                <AppSelect
                                    v-model="draft.language"
                                    :options="LANGUAGES"
                                    label="Idioma do conteúdo"
                                />
                            </div>
                        </div>
                        <p class="text-xs text-muted-foreground">
                            Nicho, público e tom de voz dão contexto para a IA
                            escrever como o perfil.
                        </p>

                        <div class="space-y-2">
                            <Label>Plataformas</Label>
                            <div
                                class="grid gap-2 sm:grid-cols-2 xl:grid-cols-3"
                            >
                                <div
                                    v-for="pl in store.catalog.platforms"
                                    :key="pl.id"
                                    :class="
                                        cn(
                                            'space-y-2 rounded-xl border-2 p-3 transition-colors',
                                            handles[pl.id]
                                                ? 'border-primary bg-primary/5'
                                                : 'border-border/50',
                                        )
                                    "
                                >
                                    <Label
                                        class="flex cursor-pointer items-center gap-2"
                                    >
                                        <Checkbox
                                            :model-value="!!handles[pl.id]"
                                            @update:model-value="
                                                togglePlatform(pl.id)
                                            "
                                        />
                                        <PlatformIcon :id="pl.id" />
                                        {{ pl.name }}
                                    </Label>
                                    <Input
                                        v-if="handles[pl.id]"
                                        v-model="handles[pl.id].handle"
                                        class="h-8"
                                        placeholder="@usuario"
                                        :aria-label="`Usuário no ${pl.name}`"
                                    />
                                </div>
                            </div>
                        </div>

                        <div class="flex items-center gap-2 pt-2">
                            <Button
                                v-if="draft.id"
                                type="button"
                                variant="ghost"
                                class="text-destructive hover:text-destructive"
                                @click="remove"
                            >
                                <Trash2 /> Excluir perfil
                            </Button>
                            <Button
                                type="submit"
                                class="ml-auto"
                                :disabled="saving"
                            >
                                {{
                                    draft.id ? "Salvar perfil" : "Criar perfil"
                                }}
                            </Button>
                        </div>
                    </form>

                    <section v-if="draft.id" :class="cn(card, 'space-y-4')">
                        <div>
                            <h2
                                class="flex items-center gap-2 font-heading text-lg font-bold"
                            >
                                <Tags class="size-5" /> Temas
                            </h2>
                            <p class="text-sm text-muted-foreground">
                                Os pilares de conteúdo do perfil. Servem de
                                filtro e de contexto para a IA.
                            </p>
                        </div>
                        <ul class="space-y-2">
                            <li
                                v-for="t in themes"
                                :key="t.id"
                                class="grid grid-cols-[32px_1fr_1.6fr_auto] items-center gap-2"
                            >
                                <input
                                    v-model="t.color"
                                    type="color"
                                    class="size-8 cursor-pointer rounded-md border border-input bg-transparent p-0.5"
                                    :aria-label="`Cor do tema ${t.name}`"
                                    @change="updateTheme(t)"
                                />
                                <Input
                                    v-model="t.name"
                                    aria-label="Nome do tema"
                                    @change="updateTheme(t)"
                                />
                                <Input
                                    v-model="t.description"
                                    placeholder="Descrição (opcional)"
                                    aria-label="Descrição do tema"
                                    @change="updateTheme(t)"
                                />
                                <Button
                                    variant="ghost"
                                    size="icon"
                                    class="text-destructive hover:text-destructive"
                                    :aria-label="`Excluir ${t.name}`"
                                    @click="removeTheme(t)"
                                >
                                    <Trash2 />
                                </Button>
                            </li>
                            <li
                                class="grid grid-cols-[32px_1fr_1.6fr_auto] items-center gap-2"
                            >
                                <input
                                    v-model="newTheme.color"
                                    type="color"
                                    class="size-8 cursor-pointer rounded-md border border-input bg-transparent p-0.5"
                                    aria-label="Cor do novo tema"
                                />
                                <Input
                                    v-model="newTheme.name"
                                    placeholder="Novo tema"
                                    aria-label="Nome do novo tema"
                                    @keydown.enter.prevent="addTheme"
                                />
                                <Input
                                    v-model="newTheme.description"
                                    placeholder="Descrição (opcional)"
                                    aria-label="Descrição do novo tema"
                                    @keydown.enter.prevent="addTheme"
                                />
                                <Button
                                    variant="outline"
                                    :disabled="!newTheme.name.trim()"
                                    @click="addTheme"
                                >
                                    <Plus /> Adicionar
                                </Button>
                            </li>
                        </ul>
                    </section>
                </div>
            </div>
        </div>
    </div>
</template>
