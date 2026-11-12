<script setup lang="ts">
import { ref } from "vue"
import { Plus, X } from "@lucide/vue"

import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"

const props = withDefaults(defineProps<{ placeholder?: string; prefix?: string; label: string }>(), {
    placeholder: "",
    prefix: "",
})
const model = defineModel<string[]>({ required: true })

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
    const seen = new Set(model.value.map((t) => t.toLowerCase()))
    const next = [...model.value]
    for (const p of parts) {
        if (!seen.has(p.toLowerCase())) {
            next.push(p)
            seen.add(p.toLowerCase())
        }
    }
    model.value = next
    draft.value = ""
}

function remove(i: number): void {
    model.value = model.value.filter((_, j) => j !== i)
}

function onKey(e: KeyboardEvent): void {
    if (e.key === "Enter" || e.key === ",") {
        e.preventDefault()
        commit()
    }
}
</script>

<template>
    <div class="flex flex-col gap-2">
        <div class="flex gap-2">
            <Input
                v-model="draft"
                class="flex-1 text-sm"
                :aria-label="label"
                :placeholder="placeholder"
                @keydown="onKey"
                @blur="commit"
            />
            <Button type="button" variant="outline" class="shrink-0" :disabled="!draft.trim()" @click="commit">
                <Plus class="size-3.5" /> Adicionar
            </Button>
        </div>
        <div v-if="model.length" class="flex flex-wrap gap-1.5 pt-1">
            <Badge
                v-for="(t, i) in model"
                :key="t"
                variant="secondary"
                class="flex items-center gap-1.5 px-2.5 py-1 text-xs font-normal"
            >
                {{ t }}
                <button
                    type="button"
                    class="cursor-pointer transition-colors hover:text-destructive focus:outline-none"
                    :aria-label="`Remover ${t}`"
                    @click="remove(i)"
                >
                    <X class="size-3" />
                </button>
            </Badge>
        </div>
    </div>
</template>
