<script setup lang="ts">
import { computed } from "vue"
import { CalendarClock } from "lucide-vue-next"

import PlatformIcon from "./PlatformIcon.vue"
import type { PostN } from "../lib/api"
import { formatSchedule, formatTime, isOverdue, parseSchedule } from "../lib/dates"
import { STATUS_COLORS, profileById, statusLabel, themeById } from "../lib/store"

const props = withDefaults(defineProps<{ post: PostN; compact?: boolean; showDate?: boolean }>(), {
    compact: false,
    showDate: false,
})

const profile = computed(() => profileById(props.post.profileId))
const theme = computed(() => themeById(props.post.themeId))
const when = computed(() => parseSchedule(props.post.scheduledAt))
const overdue = computed(() => isOverdue(props.post.scheduledAt, props.post.status))
</script>

<template>
    <article
        class="card"
        :class="{ compact, overdue, done: post.status === 'published' }"
        :style="{ '--profile': profile?.color ?? 'var(--muted)' }"
    >
        <div class="top">
            <span
                class="status"
                :style="{ background: STATUS_COLORS[post.status] }"
                :title="statusLabel(post.status)"
            ></span>
            <span v-if="showDate" class="time">{{ formatSchedule(post.scheduledAt) }}</span>
            <span v-else-if="when" class="time">{{ formatTime(when) }}</span>
            <span v-if="post.postponeCount" class="postponed" :title="`Adiado ${post.postponeCount}x`">
                <CalendarClock :size="12" />{{ post.postponeCount }}
            </span>
        </div>
        <p class="title">{{ post.title }}</p>
        <div v-if="!compact || post.platforms.length" class="meta">
            <span class="icons">
                <PlatformIcon v-for="pp in post.platforms" :id="pp.platform" :key="pp.platform" :size="12" />
            </span>
            <span v-if="!compact && theme" class="theme">{{ theme.name }}</span>
            <span v-if="!compact && profile" class="profile">{{ profile.name }}</span>
        </div>
    </article>
</template>

<style scoped>
.card {
    position: relative;
    padding: 7px 9px 8px 11px;
    border-radius: 7px;
    background: var(--surface);
    border: 1px solid var(--line);
    box-shadow: inset 3px 0 0 var(--profile);
    cursor: pointer;
    user-select: none;
}

.card:hover {
    border-color: var(--line-strong);
}

.card.compact {
    padding: 4px 7px 5px 9px;
}

.card.done {
    opacity: 0.6;
}

.card.done .title {
    text-decoration: line-through;
    text-decoration-color: var(--muted);
}

.top {
    display: flex;
    align-items: center;
    gap: 6px;
    font-size: 11.5px;
    color: var(--muted);
    font-variant-numeric: tabular-nums;
}

.status {
    width: 7px;
    height: 7px;
    border-radius: 2px;
    flex: none;
}

.overdue .time {
    color: var(--rec);
    font-weight: 700;
}

.postponed {
    display: inline-flex;
    align-items: center;
    gap: 2px;
    margin-left: auto;
    color: var(--warn);
    font-weight: 700;
}

.title {
    margin-top: 2px;
    font-weight: 600;
    font-size: 13px;
    line-height: 1.3;
    display: -webkit-box;
    -webkit-line-clamp: 2;
    -webkit-box-orient: vertical;
    overflow: hidden;
    overflow-wrap: anywhere;
}

.compact .title {
    font-size: 12.5px;
}

.meta {
    display: flex;
    align-items: center;
    gap: 8px;
    margin-top: 5px;
    font-size: 11.5px;
    color: var(--muted);
    min-width: 0;
}

.compact .meta {
    margin-top: 3px;
}

.icons {
    display: inline-flex;
    gap: 4px;
}

.theme,
.profile {
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
}

.profile {
    margin-left: auto;
}
</style>
