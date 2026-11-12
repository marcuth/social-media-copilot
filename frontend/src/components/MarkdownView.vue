<script setup lang="ts">
import { computed } from "vue"
import DOMPurify from "dompurify"
import { marked } from "marked"

const props = defineProps<{ source: string }>()

const html = computed(() => DOMPurify.sanitize(marked.parse(props.source || "", { async: false, breaks: true })))
</script>

<template>
    <!-- eslint-disable-next-line vue/no-v-html -->
    <div class="md" v-html="html"></div>
</template>

<style scoped>
.md {
    line-height: 1.6;
    overflow-wrap: anywhere;
}

.md :deep(h1),
.md :deep(h2),
.md :deep(h3) {
    font-size: 15px;
    margin: 14px 0 4px;
}

.md :deep(p) {
    margin: 0 0 8px;
}

.md :deep(ul),
.md :deep(ol) {
    margin: 0 0 8px;
    padding-left: 20px;
}

.md :deep(strong) {
    color: var(--ink);
}

.md :deep(blockquote) {
    margin: 0 0 8px;
    padding-left: 10px;
    border-left: 3px solid var(--line-strong);
    color: var(--ink-2);
}
</style>
