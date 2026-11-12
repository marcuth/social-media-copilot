<script setup lang="ts">
import { nextTick, onBeforeUnmount, onMounted, ref } from "vue"
import { X } from "lucide-vue-next"

withDefaults(defineProps<{ title: string; width?: string }>(), { width: "460px" })
const emit = defineEmits<{ close: [] }>()

const box = ref<HTMLElement>()
let previous: Element | null = null

function onKey(e: KeyboardEvent): void {
    if (e.key === "Escape") emit("close")
}

onMounted(async () => {
    previous = document.activeElement
    document.addEventListener("keydown", onKey)
    await nextTick()
    const first = box.value?.querySelector<HTMLElement>("input, select, textarea, button:not(.close)")
    first?.focus()
})

onBeforeUnmount(() => {
    document.removeEventListener("keydown", onKey)
    if (previous instanceof HTMLElement) previous.focus()
})
</script>

<template>
    <Teleport to="body">
        <div class="overlay" @mousedown.self="emit('close')">
            <div ref="box" class="modal" role="dialog" aria-modal="true" :aria-label="title" :style="{ width }">
                <header>
                    <h2>{{ title }}</h2>
                    <button class="btn ghost icon sm close" aria-label="Fechar" @click="emit('close')">
                        <X />
                    </button>
                </header>
                <div class="body"><slot /></div>
                <footer v-if="$slots.footer"><slot name="footer" /></footer>
            </div>
        </div>
    </Teleport>
</template>

<style scoped>
.overlay {
    position: fixed;
    inset: 0;
    z-index: 50;
    display: grid;
    place-items: center;
    padding: 24px;
    background: rgba(20, 18, 29, 0.45);
    animation: fade 140ms ease-out;
}

.modal {
    max-width: 100%;
    max-height: calc(100vh - 48px);
    display: flex;
    flex-direction: column;
    background: var(--surface);
    border-radius: 14px;
    box-shadow: var(--shadow-pop);
    animation: rise 180ms cubic-bezier(0.2, 0.9, 0.3, 1);
}

header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 18px 20px 6px;
}

.body {
    padding: 10px 20px 18px;
    overflow: auto;
}

footer {
    display: flex;
    justify-content: flex-end;
    gap: 8px;
    padding: 14px 20px;
    border-top: 1px solid var(--line);
}

@keyframes fade {
    from {
        opacity: 0;
    }
}

@keyframes rise {
    from {
        opacity: 0;
        transform: translateY(8px) scale(0.98);
    }
}
</style>
