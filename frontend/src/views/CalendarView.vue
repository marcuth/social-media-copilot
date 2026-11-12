<script setup lang="ts">
import { CalendarDays, ChevronLeft, ChevronRight, Plus } from "@lucide/vue"
import { computed, onMounted, ref, watch } from "vue"
import { useRouter } from "vue-router"

import {
    WEEKDAYS,
    dayKey,
    formatMonth,
    isOverdue,
    monthGrid,
    parseSchedule,
} from "@/lib/dates"
import {
    PostService,
    emptyFilter,
    normPost,
    type PostN,
    type Stats,
} from "@/lib/api"
import PostponeDialog from "@/components/PostponeDialog.vue"
import PageHeader from "@/components/PageHeader.vue"
import PostCard from "@/components/PostCard.vue"
import { profileById, store } from "@/lib/store"
import { Button } from "@/components/ui/button"
import { toast, toastError } from "@/lib/toast"
import { Badge } from "@/components/ui/badge"
import { cn } from "@/lib/utils"

const router = useRouter()

const MONTH_KEY = "smc:calendar-month"
const saved = sessionStorage.getItem(MONTH_KEY)
const month = ref(saved ? new Date(saved) : new Date())
const posts = ref<PostN[]>([])
const overdue = ref<PostN[]>([])
const unscheduled = ref<PostN[]>([])
const stats = ref<Stats | null>(null)
const postponing = ref<PostN | null>(null)
const dragOver = ref("")

const days = computed(() => monthGrid(month.value))
const todayKey = dayKey(new Date())
const activeProfile = computed(() => profileById(store.activeProfileId))

const byDay = computed(() => {
    const map: Record<string, PostN[]> = {}
    for (const p of posts.value) {
        const d = parseSchedule(p.scheduledAt)
        if (d) (map[dayKey(d)] ??= []).push(p)
    }
    return map
})

async function load(): Promise<void> {
    const profileId = store.activeProfileId
    const grid = days.value
    try {
        const [inMonth, past, noDate, st] = await Promise.all([
            PostService.ListPosts(
                emptyFilter({
                    profileId,
                    from: dayKey(grid[0]),
                    to: dayKey(grid[41]),
                }),
            ),
            PostService.ListPosts(emptyFilter({ profileId, to: todayKey })),
            PostService.ListPosts(
                emptyFilter({ profileId, unscheduled: true }),
            ),
            PostService.GetStats(profileId),
        ])
        posts.value = (inMonth ?? []).map(normPost)
        overdue.value = (past ?? [])
            .map(normPost)
            .filter((p) => isOverdue(p.scheduledAt, p.status))
        unscheduled.value = (noDate ?? [])
            .map(normPost)
            .filter((p) => p.status !== "published")
        stats.value = st
    } catch (e) {
        toastError(e)
    }
}

function shift(n: number): void {
    month.value = new Date(
        month.value.getFullYear(),
        month.value.getMonth() + n,
        1,
    )
}

watch(month, (m) => {
    sessionStorage.setItem(MONTH_KEY, m.toISOString())
    load()
})
watch(() => store.activeProfileId, load)
onMounted(load)

function open(p: PostN): void {
    router.push({ name: "post", params: { id: p.id } })
}

function createOn(d: Date): void {
    const query: Record<string, string> = { date: `${dayKey(d)}T09:00` }
    if (store.activeProfileId) query.profile = String(store.activeProfileId)
    router.push({ name: "post", params: { id: "new" }, query })
}

// Drag and drop: move a post to another day keeping its time.
function onDragStart(e: DragEvent, p: PostN): void {
    e.dataTransfer?.setData("text/post-id", String(p.id))
    if (e.dataTransfer) e.dataTransfer.effectAllowed = "move"
}

async function onDrop(e: DragEvent, d: Date): Promise<void> {
    dragOver.value = ""
    const id = Number(e.dataTransfer?.getData("text/post-id"))
    const post = [...posts.value, ...unscheduled.value, ...overdue.value].find(
        (p) => p.id === id,
    )
    if (!post) return
    const old = parseSchedule(post.scheduledAt)
    const time = old
        ? `${String(old.getHours()).padStart(2, "0")}:${String(old.getMinutes()).padStart(2, "0")}`
        : "09:00"
    const next = `${dayKey(d)}T${time}`
    if (next === post.scheduledAt) return
    try {
        const updated = normPost(await PostService.Reschedule(id, next))
        if (old && updated.postponeCount > post.postponeCount)
            toast("Post adiado e registrado no histórico.")
        await load()
    } catch (err) {
        toastError(err)
    }
}

function onPostponed(): void {
    postponing.value = null
    load()
}

const statCards = computed(() => [
    {
        label: "Atrasados",
        value: stats.value?.overdue ?? 0,
        alert: (stats.value?.overdue ?? 0) > 0,
    },
    { label: "Nesta semana", value: stats.value?.thisWeek ?? 0, alert: false },
    { label: "Sem data", value: stats.value?.unscheduled ?? 0, alert: false },
    { label: "Já adiados", value: stats.value?.postponed ?? 0, alert: false },
])
</script>

<template>
    <div class="flex h-full flex-col">
        <PageHeader
            title="Calendário"
            :description="
                activeProfile
                    ? `Publicações de ${activeProfile.name}`
                    : 'Publicações de todos os perfis'
            "
            :icon="CalendarDays"
        >
            <template #actions>
                <Button
                    variant="outline"
                    size="icon"
                    aria-label="Mês anterior"
                    @click="shift(-1)"
                >
                    <ChevronLeft />
                </Button>
                <Button variant="outline" @click="month = new Date()"
                    >Hoje</Button
                >
                <Button
                    variant="outline"
                    size="icon"
                    aria-label="Próximo mês"
                    @click="shift(1)"
                >
                    <ChevronRight />
                </Button>
            </template>
        </PageHeader>

        <div class="flex-1 overflow-y-auto px-8 py-6">
            <div class="mb-6 grid grid-cols-2 gap-4 lg:grid-cols-4">
                <div
                    v-for="s in statCards"
                    :key="s.label"
                    :class="
                        cn(
                            'rounded-xl border bg-card p-5 shadow-xs',
                            s.alert
                                ? 'border-primary/40 bg-primary/5'
                                : 'border-border/60',
                        )
                    "
                >
                    <p class="text-sm font-medium text-muted-foreground">
                        {{ s.label }}
                    </p>
                    <p
                        :class="
                            cn(
                                'mt-1 font-heading text-3xl leading-none font-bold tabular-nums',
                                s.alert && 'text-primary',
                            )
                        "
                    >
                        {{ s.value }}
                    </p>
                </div>
            </div>

            <div class="grid items-start gap-6 xl:grid-cols-[1fr_300px]">
                <section
                    class="rounded-xl border border-border/60 bg-card shadow-xs"
                >
                    <h2 class="px-5 pt-4 pb-3 font-heading text-lg font-bold">
                        {{ formatMonth(month) }}
                    </h2>
                    <div class="grid grid-cols-7 border-t border-border/60">
                        <div
                            v-for="w in WEEKDAYS"
                            :key="w"
                            class="border-b border-border/60 px-3 py-2 text-xs font-semibold text-muted-foreground"
                        >
                            {{ w }}
                        </div>
                        <div
                            v-for="(d, i) in days"
                            :key="dayKey(d)"
                            :class="
                                cn(
                                    'group flex min-h-32 min-w-0 flex-col gap-1 border-border/60 p-1.5 transition-colors',
                                    i % 7 !== 6 && 'border-r',
                                    i < 35 && 'border-b',
                                    d.getMonth() !== month.getMonth() &&
                                        'bg-muted/40',
                                    dragOver === dayKey(d) && 'bg-primary/10',
                                )
                            "
                            @dragover.prevent="dragOver = dayKey(d)"
                            @dragleave="
                                dragOver =
                                    dragOver === dayKey(d) ? '' : dragOver
                            "
                            @drop.prevent="onDrop($event, d)"
                            @dblclick.self="createOn(d)"
                        >
                            <div class="flex items-center justify-between px-1">
                                <span
                                    :class="
                                        cn(
                                            'flex size-6 items-center justify-center rounded-full text-sm font-semibold tabular-nums',
                                            dayKey(d) === todayKey
                                                ? 'bg-primary text-primary-foreground'
                                                : d.getMonth() !==
                                                        month.getMonth() ||
                                                    dayKey(d) < todayKey
                                                  ? 'text-muted-foreground'
                                                  : '',
                                        )
                                    "
                                >
                                    {{ d.getDate() }}
                                </span>
                                <Button
                                    variant="ghost"
                                    size="icon-xs"
                                    class="opacity-0 group-hover:opacity-100 focus-visible:opacity-100"
                                    :aria-label="`Novo post em ${d.toLocaleDateString('pt-BR')}`"
                                    @click="createOn(d)"
                                >
                                    <Plus />
                                </Button>
                            </div>
                            <div
                                class="flex max-h-52 flex-col gap-1 overflow-y-auto"
                            >
                                <PostCard
                                    v-for="p in byDay[dayKey(d)] ?? []"
                                    :key="p.id"
                                    :post="p"
                                    compact
                                    draggable="true"
                                    tabindex="0"
                                    @dragstart="onDragStart($event, p)"
                                    @click="open(p)"
                                    @keydown.enter="open(p)"
                                />
                            </div>
                        </div>
                    </div>
                </section>

                <aside class="space-y-6 xl:sticky xl:top-0">
                    <section class="space-y-3">
                        <h2
                            class="flex items-center gap-2 font-heading text-base font-semibold"
                        >
                            Atrasados
                            <Badge v-if="overdue.length">{{
                                overdue.length
                            }}</Badge>
                        </h2>
                        <p
                            v-if="!overdue.length"
                            class="text-sm text-muted-foreground"
                        >
                            Nada atrasado. Tudo em dia.
                        </p>
                        <div
                            v-for="p in overdue"
                            :key="p.id"
                            class="flex items-center gap-2"
                        >
                            <PostCard
                                :post="p"
                                show-date
                                class="flex-1"
                                @click="open(p)"
                            />
                            <Button
                                variant="outline"
                                size="sm"
                                @click="postponing = p"
                                >Adiar</Button
                            >
                        </div>
                    </section>

                    <section class="space-y-3">
                        <h2
                            class="flex items-center gap-2 font-heading text-base font-semibold"
                        >
                            Sem data
                            <Badge
                                v-if="unscheduled.length"
                                variant="secondary"
                                >{{ unscheduled.length }}</Badge
                            >
                        </h2>
                        <p class="text-sm text-muted-foreground">
                            {{
                                unscheduled.length
                                    ? "Arraste para um dia do calendário para agendar."
                                    : "Todo post ativo já tem data."
                            }}
                        </p>
                        <PostCard
                            v-for="p in unscheduled"
                            :key="p.id"
                            :post="p"
                            draggable="true"
                            @dragstart="onDragStart($event, p)"
                            @click="open(p)"
                        />
                    </section>
                </aside>
            </div>
        </div>

        <PostponeDialog
            v-if="postponing"
            :post="postponing"
            @close="postponing = null"
            @done="onPostponed"
        />
    </div>
</template>
