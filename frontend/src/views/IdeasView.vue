<script setup lang="ts">
import { computed, ref, watch } from "vue"
import { useRouter } from "vue-router"
import { Check, Plus, Wand2 } from "lucide-vue-next"

import { AIService, PostService, isCancel, type Idea } from "../lib/api"
import { formatLabel, profileById, store, themesOf } from "../lib/store"
import { toast, toastError } from "../lib/toast"

const router = useRouter()

const profileId = ref<number | null>(store.activeProfileId ?? store.profiles[0]?.id ?? null)
const themeId = ref<number | null>(null)
const format = ref("")
const count = ref(6)
const instructions = ref("")
const ideas = ref<(Idea & { added?: number })[]>([])
const busy = ref(false)

const themes = computed(() => themesOf(profileId.value))
watch(profileId, () => (themeId.value = null))

async function generate(): Promise<void> {
    if (!profileId.value) return
    if (!store.providers.length) {
        toast("Configure um provedor de IA em Configurações para gerar ideias.", "error")
        return
    }
    busy.value = true
    try {
        ideas.value =
            (await AIService.GenerateIdeas({
                profileId: profileId.value,
                themeId: themeId.value,
                count: count.value,
                format: format.value,
                platforms: [],
                instructions: instructions.value,
            })) ?? []
        if (!ideas.value.length) toast("A IA não trouxe ideias desta vez. Tente de novo.", "error")
    } catch (e) {
        if (!isCancel(e)) toastError(e)
    } finally {
        busy.value = false
    }
}

async function add(idea: Idea & { added?: number }, openAfter: boolean): Promise<void> {
    const profile = profileById(profileId.value)
    if (!profile) return
    try {
        const post = await PostService.SavePost({
            id: 0,
            profileId: profile.id,
            themeId: themeId.value,
            title: idea.title,
            description: "",
            script: "",
            notes: [idea.hook && `Gancho: ${idea.hook}`, idea.summary].filter(Boolean).join("\n\n"),
            tags: [],
            hashtags: [],
            format: idea.format,
            status: "idea",
            scheduledAt: null,
            publishedAt: null,
            postponeCount: 0,
            platforms: profile.platforms.map((p) => ({
                platform: p.platform,
                status: "pending",
                url: "",
                publishedAt: null,
            })),
            createdAt: "",
            updatedAt: "",
        })
        idea.added = post.id
        if (openAfter) router.push({ name: "post", params: { id: post.id } })
        else toast("Ideia adicionada à produção.")
    } catch (e) {
        toastError(e)
    }
}
</script>

<template>
    <div class="page ideas">
        <header class="page-head">
            <div>
                <h1>Ideias com IA</h1>
                <p>Peça ideias para um perfil e mande as boas direto para a produção.</p>
            </div>
        </header>

        <div v-if="!store.profiles.length" class="panel empty">
            <h3>Crie um perfil primeiro</h3>
            <p>As ideias partem do nicho, do público e do tom de voz do perfil.</p>
            <RouterLink to="/profiles" class="btn primary">Criar perfil</RouterLink>
        </div>

        <template v-else>
            <form class="panel controls" @submit.prevent="generate">
                <label class="field">
                    <span>Perfil</span>
                    <select v-model="profileId" class="select">
                        <option v-for="p in store.profiles" :key="p.id" :value="p.id">{{ p.name }}</option>
                    </select>
                </label>
                <label class="field">
                    <span>Tema</span>
                    <select v-model="themeId" class="select">
                        <option :value="null">Qualquer tema</option>
                        <option v-for="t in themes" :key="t.id" :value="t.id">{{ t.name }}</option>
                    </select>
                </label>
                <label class="field">
                    <span>Formato</span>
                    <select v-model="format" class="select">
                        <option value="">Variado</option>
                        <option v-for="f in store.catalog.formats" :key="f.id" :value="f.id">{{ f.label }}</option>
                    </select>
                </label>
                <label class="field narrow">
                    <span>Quantidade</span>
                    <input v-model.number="count" type="number" min="1" max="20" class="input" />
                </label>
                <label class="field wide">
                    <span>Direção (opcional)</span>
                    <input
                        v-model="instructions"
                        class="input"
                        placeholder="Ex.: ideias para a Black Friday, com bastidores"
                    />
                </label>
                <button class="btn accent" :disabled="busy || !profileId">
                    <Wand2 /> {{ busy ? "Pensando…" : "Gerar ideias" }}
                </button>
            </form>

            <ol v-if="ideas.length" class="list">
                <li v-for="(idea, i) in ideas" :key="i" class="idea panel">
                    <div class="idea-main">
                        <h2>{{ idea.title }}</h2>
                        <p v-if="idea.hook" class="hook">“{{ idea.hook }}”</p>
                        <p class="summary">{{ idea.summary }}</p>
                        <span class="chip">{{ formatLabel(idea.format) }}</span>
                    </div>
                    <div class="idea-actions">
                        <template v-if="idea.added">
                            <span class="added"><Check :size="15" /> Na produção</span>
                            <RouterLink class="btn sm" :to="{ name: 'post', params: { id: idea.added } }">Abrir</RouterLink>
                        </template>
                        <template v-else>
                            <button class="btn sm" @click="add(idea, false)"><Plus /> Adicionar</button>
                            <button class="btn ghost sm" @click="add(idea, true)">Adicionar e abrir</button>
                        </template>
                    </div>
                </li>
            </ol>
            <p v-else-if="!busy" class="muted hint">
                As ideias aparecem aqui. Elas entram na produção como “Ideia”, com o gancho nas anotações.
            </p>
        </template>
    </div>
</template>

<style scoped>
.ideas {
    max-width: 1100px;
}

.controls {
    display: grid;
    grid-template-columns: 1fr 1fr 1fr 110px;
    gap: 12px;
    align-items: end;
    padding: 18px;
    margin-bottom: 20px;
}

.controls .wide {
    grid-column: 1 / 4;
}

.list {
    list-style: none;
    margin: 0;
    padding: 0;
    display: flex;
    flex-direction: column;
    gap: 10px;
}

.idea {
    display: flex;
    gap: 20px;
    justify-content: space-between;
    padding: 16px 18px;
}

.idea-main {
    display: flex;
    flex-direction: column;
    align-items: flex-start;
    gap: 6px;
    max-width: 72ch;
}

.idea h2 {
    font-size: 17px;
}

.hook {
    font-style: italic;
    color: var(--ink-2);
}

.summary {
    color: var(--ink-2);
}

.idea-actions {
    display: flex;
    flex-direction: column;
    align-items: stretch;
    gap: 6px;
    flex: none;
}

.added {
    display: inline-flex;
    align-items: center;
    gap: 5px;
    color: var(--ok);
    font-weight: 700;
    font-size: 13px;
}

.hint {
    max-width: 60ch;
}
</style>
