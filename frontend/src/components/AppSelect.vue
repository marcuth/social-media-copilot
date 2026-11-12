<script setup lang="ts" generic="T extends string | number | null">
import { computed } from "vue"

import {
    Select,
    SelectContent,
    SelectItem,
    SelectTrigger,
    SelectValue,
} from "@/components/ui/select"
import { cn } from "@/lib/utils"

// shadcn Select keyed by index, so options can carry null or numeric values.
export interface SelectOption<V> {
    value: V
    label: string
}

const props = withDefaults(
    defineProps<{
        options: SelectOption<T>[]
        placeholder?: string
        label?: string
        size?: "sm" | "default"
        class?: string
    }>(),
    { placeholder: "Selecione", label: undefined, size: "default", class: "" },
)
const model = defineModel<T>({ required: true })

const key = computed({
    get: () => {
        const i = props.options.findIndex((o) => o.value === model.value)
        return i >= 0 ? String(i) : undefined
    },
    set: (k: string | undefined) => {
        if (k === undefined) return
        const option = props.options[Number(k)]
        if (option) model.value = option.value
    },
})
</script>

<template>
    <Select v-model="key">
        <SelectTrigger
            :size="size"
            :aria-label="label"
            :class="cn('w-full', props.class)"
        >
            <SelectValue :placeholder="placeholder" />
        </SelectTrigger>
        <SelectContent>
            <SelectItem v-for="(o, i) in options" :key="i" :value="String(i)">{{
                o.label
            }}</SelectItem>
        </SelectContent>
    </Select>
</template>
