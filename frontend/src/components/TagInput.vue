<script setup lang="ts">
import { ref } from "vue"
import { X } from "lucide-vue-next"

const props = withDefaults(
    defineProps<{ modelValue: string[]; placeholder?: string; prefix?: string; label: string }>(),
    { placeholder: "", prefix: "" },
)
const emit = defineEmits<{ "update:modelValue": [string[]] }>()

const draft = ref("")

function normalize(raw: string): string {
    let v = raw.trim()
    if (props.prefix) {
        v = v.replace(/\s+/g, "")
        if (v && !v.startsWith(props.prefix)) v = props.prefix + v
        if (v === props.prefix) v = ""
    }
    return v
}

function commit(): void {
    const parts = draft.value.split(/[,\n]/).map(normalize).filter(Boolean)
    if (!parts.length) return
    const seen = new Set(props.modelValue.map((t) => t.toLowerCase()))
    const next = [...props.modelValue]
    for (const p of parts) {
        if (!seen.has(p.toLowerCase())) {
            next.push(p)
            seen.add(p.toLowerCase())
        }
    }
    emit("update:modelValue", next)
    draft.value = ""
}

function remove(i: number): void {
    emit(
        "update:modelValue",
        props.modelValue.filter((_, j) => j !== i),
    )
}

function onKey(e: KeyboardEvent): void {
    if (e.key === "Enter" || e.key === ",") {
        e.preventDefault()
        commit()
    } else if (e.key === "Backspace" && !draft.value && props.modelValue.length) {
        remove(props.modelValue.length - 1)
    }
}

function onPaste(e: ClipboardEvent): void {
    const text = e.clipboardData?.getData("text") ?? ""
    if (/[,\n]/.test(text)) {
        e.preventDefault()
        draft.value += text
        commit()
    }
}
</script>

<template>
    <div class="tags" @click="($refs.inp as HTMLInputElement).focus()">
        <span v-for="(t, i) in modelValue" :key="t" class="tag">
            {{ t }}
            <button type="button" :aria-label="`Remover ${t}`" @click.stop="remove(i)"><X :size="12" /></button>
        </span>
        <input
            ref="inp"
            v-model="draft"
            class="tag-input"
            :aria-label="label"
            :placeholder="modelValue.length ? '' : placeholder"
            @keydown="onKey"
            @blur="commit"
            @paste="onPaste"
        />
    </div>
</template>

<style scoped>
.tags {
    display: flex;
    flex-wrap: wrap;
    gap: 5px;
    min-height: 34px;
    padding: 4px 6px;
    border: 1px solid var(--line-strong);
    border-radius: var(--r-sm);
    background: var(--surface);
    cursor: text;
}

.tags:focus-within {
    border-color: var(--cobalt);
    box-shadow: 0 0 0 3px var(--cobalt-soft);
}

.tag {
    display: inline-flex;
    align-items: center;
    gap: 3px;
    height: 24px;
    padding: 0 4px 0 9px;
    border-radius: 999px;
    background: var(--surface-2);
    font-size: 12.5px;
    font-weight: 600;
}

.tag button {
    display: inline-flex;
    border: 0;
    padding: 3px;
    border-radius: 50%;
    background: transparent;
    color: var(--muted);
    cursor: pointer;
}

.tag button:hover {
    color: var(--rec);
}

.tag-input {
    flex: 1;
    min-width: 120px;
    border: 0;
    outline: none;
    background: transparent;
    color: var(--ink);
    font: inherit;
}
</style>
