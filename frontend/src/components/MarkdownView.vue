<script setup lang="ts">
import { Browser } from "@wailsio/runtime"
import DOMPurify from "dompurify"
import { marked } from "marked"
import { computed } from "vue"

const props = defineProps<{ source: string }>()

// Links open in the system browser instead of replacing the app's page.
function onClick(e: MouseEvent): void {
    const link = (e.target as HTMLElement).closest("a")
    if (!link) return
    e.preventDefault()
    const href = link.getAttribute("href") ?? ""
    if (/^https?:\/\//i.test(href)) Browser.OpenURL(href)
}

const html = computed(() =>
    DOMPurify.sanitize(
        marked.parse(props.source || "", { async: false, breaks: true }),
    ),
)
</script>

<template>
    <!-- eslint-disable vue/no-v-html -- the HTML is sanitized with DOMPurify -->
    <div
        class="text-sm leading-relaxed break-words [&_a]:font-medium [&_a]:text-primary [&_a]:underline [&_a]:underline-offset-4 [&_blockquote]:mb-2 [&_blockquote]:border-l-2 [&_blockquote]:pl-3 [&_blockquote]:text-muted-foreground [&_h1]:mt-3 [&_h1]:mb-1 [&_h1]:font-heading [&_h1]:font-bold [&_h2]:mt-3 [&_h2]:mb-1 [&_h2]:font-heading [&_h2]:font-bold [&_h3]:mt-3 [&_h3]:mb-1 [&_h3]:font-semibold [&_ol]:mb-2 [&_ol]:list-decimal [&_ol]:pl-5 [&_p]:mb-2 [&_strong]:font-bold [&_ul]:mb-2 [&_ul]:list-disc [&_ul]:pl-5"
        @click="onClick"
        v-html="html"
    ></div>
</template>
