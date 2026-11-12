<script setup lang="ts">
import type { AvatarFallbackProps } from "reka-ui"
import { reactiveOmit } from "@vueuse/core"
import type { HTMLAttributes } from "vue"
import { AvatarFallback } from "reka-ui"
import { cn } from "cn"

const props = defineProps<
    AvatarFallbackProps & { class?: HTMLAttributes["class"] }
>()

const delegatedProps = reactiveOmit(props, "class")
</script>

<template>
    <AvatarFallback
        data-slot="avatar-fallback"
        v-bind="delegatedProps"
        :class="
            cn(
                'bg-muted flex size-full items-center justify-center rounded-full',
                props.class,
            )
        "
    >
        <slot />
    </AvatarFallback>
</template>
