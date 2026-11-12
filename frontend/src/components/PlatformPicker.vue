<script setup lang="ts">
import PlatformIcon from "./PlatformIcon.vue"
import { platformInfo } from "../lib/store"

// Multi-select limited to the platforms a profile has.
const props = defineProps<{ available: string[]; modelValue: string[] }>()
const emit = defineEmits<{ "update:modelValue": [string[]] }>()

function toggle(id: string): void {
    const set = new Set(props.modelValue)
    if (set.has(id)) set.delete(id)
    else set.add(id)
    emit(
        "update:modelValue",
        props.available.filter((p) => set.has(p)),
    )
}
</script>

<template>
    <div class="picker" role="group" aria-label="Plataformas">
        <button
            v-for="id in available"
            :key="id"
            type="button"
            class="opt"
            :class="{ on: modelValue.includes(id) }"
            :aria-pressed="modelValue.includes(id)"
            @click="toggle(id)"
        >
            <PlatformIcon :id="id" :size="15" />
            {{ platformInfo(id).name }}
        </button>
    </div>
</template>

<style scoped>
.picker {
    display: flex;
    flex-wrap: wrap;
    gap: 6px;
}

.opt {
    display: inline-flex;
    align-items: center;
    gap: 7px;
    height: 32px;
    padding: 0 12px 0 10px;
    border: 1px solid var(--line-strong);
    border-radius: 999px;
    background: var(--surface);
    color: var(--ink-2);
    font: inherit;
    font-weight: 600;
    font-size: 13px;
    cursor: pointer;
    opacity: 0.75;
}

.opt:hover {
    opacity: 1;
}

.opt.on {
    opacity: 1;
    border-color: var(--ink);
    color: var(--ink);
    box-shadow: inset 0 0 0 1px var(--ink);
}
</style>
