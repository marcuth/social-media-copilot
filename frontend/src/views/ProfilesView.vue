<script setup lang="ts">
import { computed, reactive, ref } from "vue"
import { Plus, Trash2 } from "lucide-vue-next"

import PlatformIcon from "../components/PlatformIcon.vue"
import { ProfileService, normProfile, type Profile, type ProfilePlatform, type Theme } from "../lib/api"
import { PROFILE_COLORS, refreshProfiles, setActiveProfile, store, themesOf } from "../lib/store"
import { toast, toastError } from "../lib/toast"

type Draft = Profile & { platforms: ProfilePlatform[] }

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

const selectedId = ref<number | null>(store.activeProfileId ?? store.profiles[0]?.id ?? null)
const draft = ref<Draft>(load(selectedId.value))
const saving = ref(false)

function load(id: number | null): Draft {
    const p = store.profiles.find((x) => x.id === id)
    return p ? JSON.parse(JSON.stringify(normProfile(p))) : blank()
}

function select(id: number | null): void {
    selectedId.value = id
    draft.value = load(id)
}

const handles = computed(() => Object.fromEntries(draft.value.platforms.map((p) => [p.platform, p])))

function togglePlatform(id: string): void {
    const list = draft.value.platforms
    const i = list.findIndex((p) => p.platform === id)
    if (i >= 0) list.splice(i, 1)
    else list.push({ platform: id, handle: "" })
}

async function save(): Promise<void> {
    saving.value = true
    try {
        const saved = await ProfileService.SaveProfile(draft.value)
        await refreshProfiles()
        const wasNew = !draft.value.id
        select(saved.id)
        if (wasNew && store.profiles.length === 1) setActiveProfile(null)
        toast(wasNew ? "Perfil criado." : "Perfil salvo.")
    } catch (e) {
        toastError(e)
    } finally {
        saving.value = false
    }
}

async function remove(): Promise<void> {
    const name = draft.value.name
    if (!confirm(`Excluir o perfil "${name}"? Todos os temas e posts dele também serão excluídos.`)) return
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

// ── Themes ───────────────────────────────────────────────────────────────
const themes = computed(() => themesOf(draft.value.id))
const newTheme = reactive({ name: "", description: "", color: PROFILE_COLORS[1] })

async function addTheme(): Promise<void> {
    try {
        await ProfileService.SaveTheme({
            id: 0,
            profileId: draft.value.id,
            name: newTheme.name,
            description: newTheme.description,
            color: newTheme.color,
            createdAt: "",
        })
        newTheme.name = ""
        newTheme.description = ""
        newTheme.color = PROFILE_COLORS[(themes.value.length + 2) % PROFILE_COLORS.length]
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
    if (!confirm(`Excluir o tema "${t.name}"? Os posts dele continuam, sem tema.`)) return
    try {
        await ProfileService.DeleteTheme(t.id)
        await refreshProfiles()
    } catch (e) {
        toastError(e)
    }
}

const LANGUAGES = [
    { id: "pt-BR", label: "Português (Brasil)" },
    { id: "pt-PT", label: "Português (Portugal)" },
    { id: "en-US", label: "Inglês" },
    { id: "es-ES", label: "Espanhol" },
]
</script>

<template>
    <div class="page profiles">
        <header class="page-head">
            <div>
                <h1>Perfis e temas</h1>
                <p>Cada perfil é um canal ou marca, com as plataformas em que publica e os temas que aborda.</p>
            </div>
            <button class="btn" @click="select(null)"><Plus /> Novo perfil</button>
        </header>

        <div class="cols">
            <nav class="list" aria-label="Perfis">
                <button
                    v-for="p in store.profiles"
                    :key="p.id"
                    class="item"
                    :class="{ on: p.id === selectedId }"
                    @click="select(p.id)"
                >
                    <span class="swatch" :style="{ background: p.color }"></span>
                    <span class="grow">
                        <strong>{{ p.name }}</strong>
                        <span class="icons">
                            <PlatformIcon v-for="pp in p.platforms" :id="pp.platform" :key="pp.platform" :size="12" />
                            <span v-if="!p.platforms.length" class="muted">Sem plataformas</span>
                        </span>
                    </span>
                </button>
                <button v-if="selectedId === null" class="item on">
                    <span class="swatch" :style="{ background: draft.color }"></span>
                    <strong class="grow">{{ draft.name || "Novo perfil" }}</strong>
                </button>
            </nav>

            <div class="editor">
                <form class="panel block" @submit.prevent="save">
                    <div class="name-row">
                        <label class="field grow">
                            <span>Nome</span>
                            <input v-model="draft.name" class="input big" placeholder="Ex.: Canal de receitas" />
                        </label>
                        <div class="field">
                            <span>Cor</span>
                            <div class="colors">
                                <button
                                    v-for="c in PROFILE_COLORS"
                                    :key="c"
                                    type="button"
                                    class="color"
                                    :class="{ on: draft.color.toLowerCase() === c.toLowerCase() }"
                                    :style="{ background: c }"
                                    :aria-label="`Cor ${c}`"
                                    @click="draft.color = c"
                                ></button>
                                <input v-model="draft.color" type="color" class="color-input" aria-label="Outra cor" />
                            </div>
                        </div>
                    </div>

                    <label class="field">
                        <span>Descrição</span>
                        <textarea
                            v-model="draft.description"
                            class="textarea"
                            rows="2"
                            placeholder="O que este perfil publica e por quê"
                        ></textarea>
                    </label>
                    <div class="grid-2">
                        <label class="field">
                            <span>Nicho</span>
                            <input v-model="draft.niche" class="input" placeholder="Ex.: culinária vegana rápida" />
                        </label>
                        <label class="field">
                            <span>Público</span>
                            <input v-model="draft.audience" class="input" placeholder="Ex.: universitários que cozinham pouco" />
                        </label>
                        <label class="field">
                            <span>Tom de voz</span>
                            <input v-model="draft.tone" class="input" placeholder="Ex.: descontraído, direto, com humor" />
                        </label>
                        <label class="field">
                            <span>Idioma do conteúdo</span>
                            <select v-model="draft.language" class="select">
                                <option v-for="l in LANGUAGES" :key="l.id" :value="l.id">{{ l.label }}</option>
                            </select>
                        </label>
                    </div>
                    <small class="muted">Nicho, público e tom de voz dão contexto para a IA escrever como o perfil.</small>

                    <div class="field">
                        <span>Plataformas</span>
                        <div class="platforms">
                            <div
                                v-for="pl in store.catalog.platforms"
                                :key="pl.id"
                                class="platform"
                                :class="{ on: !!handles[pl.id] }"
                            >
                                <label class="check">
                                    <input type="checkbox" :checked="!!handles[pl.id]" @change="togglePlatform(pl.id)" />
                                    <PlatformIcon :id="pl.id" />
                                    {{ pl.name }}
                                </label>
                                <input
                                    v-if="handles[pl.id]"
                                    v-model="handles[pl.id].handle"
                                    class="input handle"
                                    placeholder="@usuario"
                                    :aria-label="`Usuário no ${pl.name}`"
                                />
                            </div>
                        </div>
                    </div>

                    <div class="actions">
                        <button v-if="draft.id" type="button" class="btn ghost danger" @click="remove">
                            <Trash2 /> Excluir perfil
                        </button>
                        <span class="grow"></span>
                        <button class="btn primary" :disabled="saving">
                            {{ draft.id ? "Salvar perfil" : "Criar perfil" }}
                        </button>
                    </div>
                </form>

                <section v-if="draft.id" class="panel block">
                    <div>
                        <h2>Temas</h2>
                        <p class="muted">Os pilares de conteúdo do perfil. Servem de filtro e de contexto para a IA.</p>
                    </div>
                    <ul class="themes">
                        <li v-for="t in themes" :key="t.id" class="theme">
                            <input
                                v-model="t.color"
                                type="color"
                                class="color-input"
                                :aria-label="`Cor do tema ${t.name}`"
                                @change="updateTheme(t)"
                            />
                            <input v-model="t.name" class="input" aria-label="Nome do tema" @change="updateTheme(t)" />
                            <input
                                v-model="t.description"
                                class="input"
                                placeholder="Descrição (opcional)"
                                aria-label="Descrição do tema"
                                @change="updateTheme(t)"
                            />
                            <button class="btn ghost icon sm danger" :aria-label="`Excluir ${t.name}`" @click="removeTheme(t)">
                                <Trash2 />
                            </button>
                        </li>
                        <li class="theme">
                            <input v-model="newTheme.color" type="color" class="color-input" aria-label="Cor do novo tema" />
                            <input
                                v-model="newTheme.name"
                                class="input"
                                placeholder="Novo tema"
                                aria-label="Nome do novo tema"
                                @keydown.enter.prevent="addTheme"
                            />
                            <input
                                v-model="newTheme.description"
                                class="input"
                                placeholder="Descrição (opcional)"
                                aria-label="Descrição do novo tema"
                                @keydown.enter.prevent="addTheme"
                            />
                            <button class="btn sm" :disabled="!newTheme.name.trim()" @click="addTheme">Adicionar</button>
                        </li>
                    </ul>
                </section>
            </div>
        </div>
    </div>
</template>

<style scoped>
.profiles {
    max-width: 1200px;
}

.cols {
    display: grid;
    grid-template-columns: 260px 1fr;
    gap: 20px;
    align-items: start;
}

.list {
    display: flex;
    flex-direction: column;
    gap: 4px;
}

.item {
    display: flex;
    align-items: center;
    gap: 10px;
    padding: 10px 12px;
    border: 1px solid transparent;
    border-radius: var(--r-md);
    background: transparent;
    color: var(--ink);
    font: inherit;
    text-align: left;
    cursor: pointer;
}

.item:hover {
    background: var(--surface);
}

.item.on {
    background: var(--surface);
    border-color: var(--line-strong);
}

.item .grow {
    display: flex;
    flex-direction: column;
    gap: 3px;
    min-width: 0;
}

.item strong {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
}

.icons {
    display: flex;
    gap: 4px;
    font-size: 12px;
}

.swatch {
    width: 14px;
    height: 14px;
    border-radius: 4px;
    flex: none;
}

.editor {
    display: flex;
    flex-direction: column;
    gap: 16px;
}

.block {
    display: flex;
    flex-direction: column;
    gap: 14px;
    padding: 20px;
}

.name-row {
    display: flex;
    gap: 16px;
    align-items: flex-end;
    flex-wrap: wrap;
}

.big {
    height: 42px;
    font-size: 17px;
    font-weight: 600;
}

.colors {
    display: flex;
    gap: 6px;
    align-items: center;
    height: 42px;
}

.color {
    width: 24px;
    height: 24px;
    border-radius: 50%;
    border: 0;
    cursor: pointer;
}

.color.on {
    box-shadow:
        0 0 0 2px var(--surface),
        0 0 0 4px var(--ink);
}

.color-input {
    width: 30px;
    height: 30px;
    padding: 0;
    border: 1px solid var(--line-strong);
    border-radius: var(--r-sm);
    background: none;
    cursor: pointer;
    flex: none;
}

.grid-2 {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 12px;
}

.platforms {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(230px, 1fr));
    gap: 8px;
}

.platform {
    display: flex;
    flex-direction: column;
    gap: 8px;
    padding: 10px 12px;
    border: 1px solid var(--line);
    border-radius: var(--r-md);
}

.platform .check {
    align-items: center;
    font-weight: 600;
}

.platform .check input {
    margin-top: 0;
}

.platform.on {
    border-color: var(--ink);
}

.handle {
    height: 30px;
}

.actions {
    display: flex;
    align-items: center;
    gap: 8px;
    padding-top: 6px;
}

.themes {
    list-style: none;
    margin: 0;
    padding: 0;
    display: flex;
    flex-direction: column;
    gap: 8px;
}

.theme {
    display: grid;
    grid-template-columns: 30px 1fr 1.6fr auto;
    gap: 8px;
    align-items: center;
}
</style>
