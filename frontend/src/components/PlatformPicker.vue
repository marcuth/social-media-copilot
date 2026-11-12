<script setup lang="ts">
import { Check } from "@lucide/vue"

import PlatformIcon from "@/components/PlatformIcon.vue"
import { platformInfo } from "@/lib/store"
import { cn } from "@/lib/utils"

// Multi-select limited to the platforms a profile has.
const props = defineProps<{ available: string[] }>()
const model = defineModel<string[]>({ required: true })

function toggle(id: string): void {
    const set = new Set(model.value)
    if (set.has(id)) set.delete(id)
    else set.add(id)
    model.value = props.available.filter((p) => set.has(p))
}
</script>

<template>
    <div class="flex flex-wrap gap-2" role="group" aria-label="Plataformas">
        <button
            v-for="id in available"
            :key="id"
            type="button"
            :aria-pressed="model.includes(id)"
            :class="
                cn(
                    'flex h-9 cursor-pointer items-center gap-2 rounded-md border-2 px-3 text-sm font-medium transition-colors outline-none focus-visible:ring-[3px] focus-visible:ring-ring/50',
                    model.includes(id)
                        ? 'border-primary bg-primary/10'
                        : 'border-border/50 bg-card text-muted-foreground hover:border-primary/30',
                )
            "
            @click="toggle(id)"
        >
            <PlatformIcon :id="id" :size="15" />
            {{ platformInfo(id).name }}
            <Check v-if="model.includes(id)" class="size-3.5 text-primary" />
        </button>
    </div>
</template>
