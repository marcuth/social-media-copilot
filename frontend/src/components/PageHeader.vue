<script setup lang="ts">
import type { Component } from "vue"

defineProps<{ title: string; description?: string; icon?: Component }>()
</script>

<template>
    <div class="flex items-center justify-between gap-4 border-b border-border/50 px-8 py-5">
        <div class="flex min-w-0 items-center gap-3">
            <slot name="leading">
                <component :is="icon" v-if="icon" class="size-5 shrink-0" />
            </slot>
            <div class="min-w-0">
                <h1 class="truncate font-heading text-2xl font-bold">{{ title }}</h1>
                <p v-if="description || $slots.description" class="mt-0.5 text-sm text-muted-foreground">
                    <slot name="description">{{ description }}</slot>
                </p>
            </div>
        </div>
        <div v-if="$slots.actions" class="flex shrink-0 items-center gap-2">
            <slot name="actions" />
        </div>
    </div>
</template>
