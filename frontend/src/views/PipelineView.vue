<script setup lang="ts">
import { computed, onMounted, ref, watch } from "vue"
import { useRouter } from "vue-router"
import { Columns3, Plus, Search } from "@lucide/vue"

import AppSelect from "@/components/AppSelect.vue"
import PageHeader from "@/components/PageHeader.vue"
import PostCard from "@/components/PostCard.vue"
import { Button } from "@/components/ui/button"
import { Checkbox } from "@/components/ui/checkbox"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { PostService, emptyFilter, normPost, type PostN } from "@/lib/api"
import { STATUS_COLORS, platformInfo, store, themesOf } from "@/lib/store"
import { toastError } from "@/lib/toast"
import { cn } from "@/lib/utils"

const router = useRouter()

const posts = ref<PostN[]>([])
const search = ref("")
const themeId = ref<number | null>(null)
const platform = ref("")
const hidePublished = ref(true)
const dragOver = ref("")

const themeOptions = computed(() => [
    { value: null, label: "Todos os temas" },
    ...(store.activeProfileId ? themesOf(store.activeProfileId) : store.themes).map((t) => ({
        value: t.id,
        label: t.name,
    })),
])
const platformOptions = computed(() => {
    const profiles = store.activeProfileId
        ? store.profiles.filter((p) => p.id === store.activeProfileId)
        : store.profiles
    const ids = [...new Set(profiles.flatMap((p) => p.platforms.map((x) => x.platform)))]
    return [{ value: "", label: "Todas as plataformas" }, ...ids.map((id) => ({ value: id, label: platformInfo(id).name }))]
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

function open(p: PostN): void {
    router.push({ name: "post", params: { id: p.id } })
}
</script>

<template>
    <div class="flex h-full flex-col">
        <PageHeader
            title="Produção"
            description="Arraste os posts entre as etapas conforme o trabalho avança"
            :icon="Columns3"
        />

        <div class="flex flex-wrap items-center gap-3 border-b border-border/50 px-8 py-3">
            <div class="relative">
                <Search class="pointer-events-none absolute top-1/2 left-3 size-4 -translate-y-1/2 text-muted-foreground" />
                <Input v-model="search" class="w-60 pl-9" placeholder="Buscar" aria-label="Buscar posts" />
            </div>
            <AppSelect v-model="themeId" :options="themeOptions" label="Tema" class="w-48" />
            <AppSelect v-model="platform" :options="platformOptions" label="Plataforma" class="w-52" />
            <Label class="ml-auto flex items-center gap-2 font-normal">
                <Checkbox v-model="hidePublished" /> Ocultar publicados
            </Label>
        </div>

        <div class="flex-1 overflow-auto px-8 py-6">
            <div class="grid h-full min-h-[480px] gap-4" :style="{ gridTemplateColumns: `repeat(${columns.length}, minmax(200px, 1fr))` }">
                <section
                    v-for="col in columns"
                    :key="col.id"
                    :aria-label="col.label"
                    :class="
                        cn(
                            'flex flex-col gap-2 rounded-xl border border-border/60 bg-muted/40 p-3 transition-colors',
                            dragOver === col.id && 'border-primary/40 bg-primary/5',
                        )
                    "
                    @dragover.prevent="dragOver = col.id"
                    @dragleave="dragOver = dragOver === col.id ? '' : dragOver"
                    @drop.prevent="onDrop($event, col.id)"
                >
                    <header class="flex items-center gap-2 px-1">
                        <span class="h-4 w-1 rounded-full" :style="{ background: STATUS_COLORS[col.id] }"></span>
                        <h2 class="font-heading text-sm font-bold">{{ col.label }}</h2>
                        <span class="mr-auto text-xs font-semibold text-muted-foreground tabular-nums">
                            {{ col.posts.length }}
                        </span>
                        <Button variant="ghost" size="icon-xs" :aria-label="`Novo post em ${col.label}`" @click="newPost(col.id)">
                            <Plus />
                        </Button>
                    </header>
                    <div class="flex flex-col gap-2 overflow-y-auto">
                        <PostCard
                            v-for="p in col.posts"
                            :key="p.id"
                            :post="p"
                            show-date
                            draggable="true"
                            tabindex="0"
                            @dragstart="onDragStart($event, p)"
                            @click="open(p)"
                            @keydown.enter="open(p)"
                        />
                        <p v-if="!col.posts.length" class="px-1 py-2 text-xs text-muted-foreground">Nenhum post nesta etapa.</p>
                    </div>
                </section>
            </div>
        </div>
    </div>
</template>
