<script setup lang="ts">
import { CalendarClock } from "@lucide/vue"
import { computed } from "vue"

import {
    formatSchedule,
    formatTime,
    isOverdue,
    parseSchedule,
} from "@/lib/dates"
import { STATUS_COLORS, profileById, statusLabel, themeById } from "@/lib/store"
import PlatformIcon from "@/components/PlatformIcon.vue"
import type { PostN } from "@/lib/api"
import { cn } from "@/lib/utils"

const props = withDefaults(
    defineProps<{ post: PostN; compact?: boolean; showDate?: boolean }>(),
    {
        compact: false,
        showDate: false,
    },
)

const profile = computed(() => profileById(props.post.profileId))
const theme = computed(() => themeById(props.post.themeId))
const when = computed(() => parseSchedule(props.post.scheduledAt))
const overdue = computed(() =>
    isOverdue(props.post.scheduledAt, props.post.status),
)
const done = computed(() => props.post.status === "published")
</script>

<template>
    <article
        :class="
            cn(
                'relative cursor-pointer rounded-md border border-border/60 bg-card shadow-xs transition-colors outline-none select-none hover:border-primary/40 focus-visible:ring-[3px] focus-visible:ring-ring/50',
                compact ? 'px-2 py-1.5' : 'px-3 py-2.5',
                done && 'opacity-60',
            )
        "
        :style="{
            boxShadow: `inset 3px 0 0 ${profile?.color ?? 'var(--muted-foreground)'}`,
        }"
    >
        <div
            class="flex items-center gap-1.5 text-[11px] text-muted-foreground tabular-nums"
        >
            <span
                class="size-2 shrink-0 rounded-[3px]"
                :style="{ background: STATUS_COLORS[post.status] }"
                :title="statusLabel(post.status)"
            ></span>
            <span :class="overdue && 'font-bold text-primary'">
                {{
                    showDate
                        ? formatSchedule(post.scheduledAt)
                        : when
                          ? formatTime(when)
                          : ""
                }}
            </span>
            <span
                v-if="post.postponeCount"
                class="ml-auto inline-flex items-center gap-0.5 font-bold text-amber-600 dark:text-amber-400"
                :title="`Adiado ${post.postponeCount}x`"
            >
                <CalendarClock class="size-3" />{{ post.postponeCount }}
            </span>
        </div>
        <p
            :class="
                cn(
                    'mt-0.5 line-clamp-2 leading-snug font-semibold break-words',
                    compact ? 'text-xs' : 'text-sm',
                    done && 'line-through decoration-muted-foreground',
                )
            "
        >
            {{ post.title }}
        </p>
        <div
            v-if="!compact || post.platforms.length"
            :class="
                cn(
                    'flex min-w-0 items-center gap-2 text-[11px] text-muted-foreground',
                    compact ? 'mt-1' : 'mt-1.5',
                )
            "
        >
            <span class="inline-flex gap-1">
                <PlatformIcon
                    v-for="pp in post.platforms"
                    :id="pp.platform"
                    :key="pp.platform"
                    :size="12"
                />
            </span>
            <span v-if="!compact && theme" class="truncate">{{
                theme.name
            }}</span>
            <span v-if="!compact && profile" class="ml-auto truncate">{{
                profile.name
            }}</span>
        </div>
    </article>
</template>
