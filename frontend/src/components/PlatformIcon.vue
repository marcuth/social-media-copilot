<script setup lang="ts">
import { computed } from "vue"
import { siFacebook, siInstagram, siPinterest, siThreads, siTiktok, siX, siYoutube } from "simple-icons"

import { platformInfo } from "../lib/store"

const props = withDefaults(defineProps<{ id: string; size?: number; mono?: boolean }>(), { size: 16, mono: false })

// LinkedIn and Kwai are not in simple-icons; they get a lettermark.
const PATHS: Record<string, string> = {
    youtube: siYoutube.path,
    tiktok: siTiktok.path,
    instagram: siInstagram.path,
    facebook: siFacebook.path,
    x: siX.path,
    threads: siThreads.path,
    pinterest: siPinterest.path,
}
const LETTERS: Record<string, string> = { linkedin: "in", kwai: "K" }

const info = computed(() => platformInfo(props.id))
// Black-branded marks follow the text colour so they stay visible in dark mode.
const INK_MARKS = new Set(["x", "threads"])
const color = computed(() => (props.mono || INK_MARKS.has(props.id) ? "var(--ink)" : info.value.color))
</script>

<template>
    <svg
        v-if="PATHS[id]"
        :width="size"
        :height="size"
        viewBox="0 0 24 24"
        role="img"
        :aria-label="info.name"
        class="platform-icon"
    >
        <title>{{ info.name }}</title>
        <path :d="PATHS[id]" :fill="color" />
    </svg>
    <span
        v-else
        class="platform-letter"
        role="img"
        :aria-label="info.name"
        :title="info.name"
        :style="{ width: `${size}px`, height: `${size}px`, background: color, fontSize: `${size * 0.55}px` }"
        >{{ LETTERS[id] ?? info.name.charAt(0) }}</span
    >
</template>

<style scoped>
.platform-icon {
    flex: none;
    display: block;
}

.platform-letter {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    flex: none;
    border-radius: 3px;
    color: #fff;
    font-weight: 800;
    line-height: 1;
}
</style>
