<script setup lang="ts">
import { computed, onMounted, ref, watch } from "vue"
import { useRouter } from "vue-router"
import { ChevronLeft, ChevronRight, Plus } from "lucide-vue-next"

import PostCard from "../components/PostCard.vue"
import PostponeDialog from "../components/PostponeDialog.vue"
import { PostService, emptyFilter, normPost, type PostN, type Stats } from "../lib/api"
import { WEEKDAYS, dayKey, formatMonth, isOverdue, monthGrid, parseSchedule } from "../lib/dates"
import { store } from "../lib/store"
import { toast, toastError } from "../lib/toast"

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

const byDay = computed(() => {
    const map: Record<string, PostN[]> = {}
    for (const p of posts.value) {
        const d = parseSchedule(p.scheduledAt)
        if (!d) continue
        ;(map[dayKey(d)] ??= []).push(p)
    }
    return map
})

async function load(): Promise<void> {
    const profileId = store.activeProfileId
    const grid = days.value
    try {
        const [inMonth, past, noDate, st] = await Promise.all([
            PostService.ListPosts(emptyFilter({ profileId, from: dayKey(grid[0]), to: dayKey(grid[41]) })),
            PostService.ListPosts(emptyFilter({ profileId, to: todayKey })),
            PostService.ListPosts(emptyFilter({ profileId, unscheduled: true })),
            PostService.GetStats(profileId),
        ])
        posts.value = (inMonth ?? []).map(normPost)
        overdue.value = (past ?? []).map(normPost).filter((p) => isOverdue(p.scheduledAt, p.status))
        unscheduled.value = (noDate ?? []).map(normPost).filter((p) => p.status !== "published")
        stats.value = st
    } catch (e) {
        toastError(e)
    }
}

function shift(n: number): void {
    const d = new Date(month.value.getFullYear(), month.value.getMonth() + n, 1)
    month.value = d
}

function goToday(): void {
    month.value = new Date()
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

// ── Drag and drop: move a post to another day keeping its time ───────────
function onDragStart(e: DragEvent, p: PostN): void {
    e.dataTransfer?.setData("text/post-id", String(p.id))
    if (e.dataTransfer) e.dataTransfer.effectAllowed = "move"
}

async function onDrop(e: DragEvent, d: Date): Promise<void> {
    dragOver.value = ""
    const id = Number(e.dataTransfer?.getData("text/post-id"))
    const post = [...posts.value, ...unscheduled.value, ...overdue.value].find((p) => p.id === id)
    if (!post) return
    const old = parseSchedule(post.scheduledAt)
    const time = old ? `${String(old.getHours()).padStart(2, "0")}:${String(old.getMinutes()).padStart(2, "0")}` : "09:00"
    const next = `${dayKey(d)}T${time}`
    if (next === post.scheduledAt) return
    try {
        const updated = normPost(await PostService.Reschedule(id, next))
        if (old && updated.postponeCount > post.postponeCount) toast("Post adiado e registrado no histórico.")
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
    { label: "Atrasados", value: stats.value?.overdue ?? 0, alert: (stats.value?.overdue ?? 0) > 0 },
    { label: "Nesta semana", value: stats.value?.thisWeek ?? 0, alert: false },
    { label: "Sem data", value: stats.value?.unscheduled ?? 0, alert: false },
    { label: "Já adiados", value: stats.value?.postponed ?? 0, alert: false },
])
</script>

<template>
    <div class="page calendar-page">
        <header class="page-head">
            <div class="month-title">
                <h1>{{ formatMonth(month) }}</h1>
                <div class="row">
                    <button class="btn icon sm" aria-label="Mês anterior" @click="shift(-1)"><ChevronLeft /></button>
                    <button class="btn sm" @click="goToday">Hoje</button>
                    <button class="btn icon sm" aria-label="Próximo mês" @click="shift(1)"><ChevronRight /></button>
                </div>
            </div>
            <dl class="stats">
                <div v-for="s in statCards" :key="s.label" class="stat" :class="{ alert: s.alert }">
                    <dt>{{ s.label }}</dt>
                    <dd>{{ s.value }}</dd>
                </div>
            </dl>
        </header>

        <div class="layout">
            <section class="grid" aria-label="Calendário do mês">
                <div v-for="w in WEEKDAYS" :key="w" class="weekday">{{ w }}</div>
                <div
                    v-for="d in days"
                    :key="dayKey(d)"
                    class="day"
                    :class="{
                        out: d.getMonth() !== month.getMonth(),
                        today: dayKey(d) === todayKey,
                        past: dayKey(d) < todayKey,
                        over: dragOver === dayKey(d),
                    }"
                    @dragover.prevent="dragOver = dayKey(d)"
                    @dragleave="dragOver = dragOver === dayKey(d) ? '' : dragOver"
                    @drop.prevent="onDrop($event, d)"
                    @dblclick.self="createOn(d)"
                >
                    <div class="day-head">
                        <span class="num">{{ d.getDate() }}</span>
                        <button
                            class="add"
                            :aria-label="`Novo post em ${d.toLocaleDateString('pt-BR')}`"
                            @click="createOn(d)"
                        >
                            <Plus :size="14" />
                        </button>
                    </div>
                    <div class="day-posts">
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
            </section>

            <aside class="queues">
                <section class="queue">
                    <h2>
                        Atrasados <span v-if="overdue.length" class="count alert">{{ overdue.length }}</span>
                    </h2>
                    <p v-if="!overdue.length" class="muted small">Nada atrasado. Tudo em dia.</p>
                    <div v-for="p in overdue" :key="p.id" class="queue-item">
                        <PostCard :post="p" show-date @click="open(p)" />
                        <button class="btn sm" @click="postponing = p">Adiar</button>
                    </div>
                </section>

                <section class="queue">
                    <h2>
                        Sem data <span v-if="unscheduled.length" class="count">{{ unscheduled.length }}</span>
                    </h2>
                    <p class="muted small">
                        {{ unscheduled.length ? "Arraste para um dia do calendário para agendar." : "Todo post ativo já tem data." }}
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

        <PostponeDialog v-if="postponing" :post="postponing" @close="postponing = null" @done="onPostponed" />
    </div>
</template>

<style scoped>
.calendar-page {
    max-width: none;
}

.month-title {
    display: flex;
    align-items: baseline;
    gap: 18px;
}

.month-title h1 {
    font-size: 44px;
    font-weight: 700;
    font-variation-settings: "wdth" 75;
    letter-spacing: -0.02em;
    text-transform: lowercase;
}

.stats {
    display: flex;
    gap: 6px;
    margin: 0;
}

.stat {
    min-width: 100px;
    padding: 8px 14px;
    border-radius: var(--r-md);
    background: var(--surface);
    border: 1px solid var(--line);
}

.stat dt {
    font-size: 12px;
    color: var(--muted);
    font-weight: 600;
}

.stat dd {
    margin: 0;
    font-family: var(--font-display);
    font-size: 24px;
    font-weight: 700;
    font-variant-numeric: tabular-nums;
}

.stat.alert {
    border-color: var(--rec);
    background: var(--rec-soft);
}

.stat.alert dd,
.stat.alert dt {
    color: var(--rec);
}

.layout {
    display: grid;
    grid-template-columns: 1fr 290px;
    gap: 20px;
    align-items: start;
}

.grid {
    display: grid;
    grid-template-columns: repeat(7, minmax(0, 1fr));
    grid-auto-rows: auto;
    gap: 1px;
    background: var(--line);
    border: 1px solid var(--line);
    border-radius: var(--r-md);
    overflow: hidden;
}

.weekday {
    padding: 8px 10px;
    background: var(--surface-2);
    color: var(--muted);
    font-size: 12px;
    font-weight: 700;
}

.day {
    min-height: 128px;
    padding: 6px;
    background: var(--surface);
    display: flex;
    flex-direction: column;
    gap: 4px;
    min-width: 0;
    transition: background-color 100ms;
}

.day.out {
    background: var(--paper);
}

.day.out .num {
    color: var(--line-strong);
}

.day.over {
    background: var(--cobalt-soft);
}

.day-head {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 0 2px;
}

.num {
    font-family: var(--font-display);
    font-weight: 650;
    font-size: 17px;
    font-variation-settings: "wdth" 75;
    font-variant-numeric: tabular-nums;
    color: var(--ink-2);
}

.day.past:not(.out) .num {
    color: var(--muted);
}

/* Today wears the REC light. */
.day.today .num {
    display: inline-flex;
    align-items: center;
    gap: 6px;
    color: var(--rec);
}

.day.today .num::before {
    content: "";
    width: 8px;
    height: 8px;
    border-radius: 50%;
    background: var(--rec);
    box-shadow: 0 0 0 3px var(--rec-soft);
}

.add {
    display: inline-flex;
    padding: 3px;
    border: 0;
    border-radius: 4px;
    background: transparent;
    color: var(--muted);
    cursor: pointer;
    opacity: 0;
}

.day:hover .add,
.add:focus-visible {
    opacity: 1;
}

.add:hover {
    background: var(--surface-2);
    color: var(--ink);
}

.day-posts {
    display: flex;
    flex-direction: column;
    gap: 4px;
    max-height: 210px;
    overflow-y: auto;
}

.queues {
    display: flex;
    flex-direction: column;
    gap: 22px;
    position: sticky;
    top: 0;
}

.queue {
    display: flex;
    flex-direction: column;
    gap: 6px;
}

.queue h2 {
    display: flex;
    align-items: center;
    gap: 8px;
    font-size: 16px;
}

.count {
    padding: 1px 8px;
    border-radius: 999px;
    background: var(--surface-2);
    font-family: var(--font-body);
    font-size: 12px;
}

.count.alert {
    background: var(--rec);
    color: #fff;
}

.small {
    font-size: 12.5px;
}

.queue-item {
    display: grid;
    grid-template-columns: 1fr auto;
    gap: 6px;
    align-items: center;
}

@media (max-width: 1180px) {
    .layout {
        grid-template-columns: 1fr;
    }

    .queues {
        position: static;
    }
}
</style>
