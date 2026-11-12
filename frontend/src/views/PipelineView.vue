<script setup lang="ts">
import { computed, onMounted, ref, watch } from "vue"
import { useRouter } from "vue-router"
import { Plus, Search } from "lucide-vue-next"

import PostCard from "../components/PostCard.vue"
import { PostService, emptyFilter, normPost, type PostN } from "../lib/api"
import { STATUS_COLORS, platformInfo, store, themesOf } from "../lib/store"
import { toastError } from "../lib/toast"

const router = useRouter()

const posts = ref<PostN[]>([])
const search = ref("")
const themeId = ref<number | null>(null)
const platform = ref("")
const hidePublished = ref(true)
const dragOver = ref("")

const themes = computed(() => (store.activeProfileId ? themesOf(store.activeProfileId) : store.themes))
const platforms = computed(() => {
    const profiles = store.activeProfileId
        ? store.profiles.filter((p) => p.id === store.activeProfileId)
        : store.profiles
    return [...new Set(profiles.flatMap((p) => p.platforms.map((x) => x.platform)))]
})

const columns = computed(() =>
    store.catalog.statuses
        .filter((s) => !(hidePublished.value && s.id === "published"))
        .map((s) => ({ ...s, posts: posts.value.filter((p) => p.status === s.id) })),
)

let timer: ReturnType<typeof setTimeout> | undefined
async function load(): Promise<void> {
    try {
        const list = await PostService.ListPosts(
            emptyFilter({
                profileId: store.activeProfileId,
                themeId: themeId.value,
                platform: platform.value,
                search: search.value,
            }),
        )
        posts.value = (list ?? []).map(normPost)
    } catch (e) {
        toastError(e)
    }
}

watch([() => store.activeProfileId, themeId, platform], load)
watch(search, () => {
    clearTimeout(timer)
    timer = setTimeout(load, 250)
})
onMounted(load)

function onDragStart(e: DragEvent, p: PostN): void {
    e.dataTransfer?.setData("text/post-id", String(p.id))
}

async function onDrop(e: DragEvent, status: string): Promise<void> {
    dragOver.value = ""
    const id = Number(e.dataTransfer?.getData("text/post-id"))
    const post = posts.value.find((p) => p.id === id)
    if (!post || post.status === status) return
    const previous = post.status
    post.status = status
    try {
        Object.assign(post, normPost(await PostService.UpdateStatus(id, status)))
    } catch (err) {
        post.status = previous
        toastError(err)
    }
}

function newPost(status: string): void {
    const query: Record<string, string> = { status }
    if (store.activeProfileId) query.profile = String(store.activeProfileId)
    router.push({ name: "post", params: { id: "new" }, query })
}
</script>

<template>
    <div class="page pipeline">
        <header class="page-head">
            <div>
                <h1>Produção</h1>
                <p>Arraste os posts entre as etapas conforme o trabalho avança.</p>
            </div>
            <div class="filters">
                <label class="search">
                    <Search :size="15" />
                    <input v-model="search" class="input" placeholder="Buscar" aria-label="Buscar posts" />
                </label>
                <select v-model="themeId" class="select" aria-label="Tema">
                    <option :value="null">Todos os temas</option>
                    <option v-for="t in themes" :key="t.id" :value="t.id">{{ t.name }}</option>
                </select>
                <select v-model="platform" class="select" aria-label="Plataforma">
                    <option value="">Todas as plataformas</option>
                    <option v-for="p in platforms" :key="p" :value="p">{{ platformInfo(p).name }}</option>
                </select>
                <label class="check">
                    <input v-model="hidePublished" type="checkbox" />
                    Ocultar publicados
                </label>
            </div>
        </header>

        <div class="board" :style="{ '--cols': columns.length }">
            <section
                v-for="col in columns"
                :key="col.id"
                class="column"
                :class="{ over: dragOver === col.id }"
                :aria-label="col.label"
                @dragover.prevent="dragOver = col.id"
                @dragleave="dragOver = dragOver === col.id ? '' : dragOver"
                @drop.prevent="onDrop($event, col.id)"
            >
                <header>
                    <span class="bar" :style="{ background: STATUS_COLORS[col.id] }"></span>
                    <h2>{{ col.label }}</h2>
                    <span class="n">{{ col.posts.length }}</span>
                    <button class="btn ghost icon sm" :aria-label="`Novo post em ${col.label}`" @click="newPost(col.id)">
                        <Plus />
                    </button>
                </header>
                <div class="cards">
                    <PostCard
                        v-for="p in col.posts"
                        :key="p.id"
                        :post="p"
                        show-date
                        draggable="true"
                        tabindex="0"
                        @dragstart="onDragStart($event, p)"
                        @click="router.push({ name: 'post', params: { id: p.id } })"
                        @keydown.enter="router.push({ name: 'post', params: { id: p.id } })"
                    />
                    <p v-if="!col.posts.length" class="muted none">Nenhum post nesta etapa.</p>
                </div>
            </section>
        </div>
    </div>
</template>

<style scoped>
.pipeline {
    max-width: none;
}

.filters {
    display: flex;
    align-items: center;
    gap: 8px;
    flex-wrap: wrap;
}

.filters .select {
    width: auto;
}

.search {
    position: relative;
    display: flex;
    align-items: center;
}

.search svg {
    position: absolute;
    left: 10px;
    color: var(--muted);
}

.search .input {
    width: 200px;
    padding-left: 32px;
}

.board {
    display: grid;
    grid-template-columns: repeat(var(--cols), minmax(180px, 1fr));
    gap: 12px;
    overflow-x: auto;
    padding-bottom: 8px;
}

.column {
    display: flex;
    flex-direction: column;
    gap: 8px;
    min-height: 60vh;
    padding: 10px;
    border-radius: var(--r-md);
    background: var(--surface-2);
    transition: background-color 100ms;
}

.column.over {
    background: var(--cobalt-soft);
}

.column header {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 2px 2px 4px;
}

.bar {
    width: 4px;
    height: 16px;
    border-radius: 2px;
}

.column h2 {
    font-size: 15px;
}

.n {
    color: var(--muted);
    font-weight: 700;
    font-variant-numeric: tabular-nums;
    margin-right: auto;
}

.cards {
    display: flex;
    flex-direction: column;
    gap: 6px;
}

.none {
    padding: 8px 4px;
    font-size: 12.5px;
}
</style>
