<script setup lang="ts">
import {
    Dialog,
    DialogContent,
    DialogDescription,
    DialogTitle,
} from "@/components/ui/dialog"
import {
    Sheet,
    SheetContent,
    SheetDescription,
    SheetTitle,
} from "@/components/ui/sheet"
import ChatPanel from "@/components/ChatPanel.vue"
import { chat } from "@/lib/chat"

// The panel opens as a sheet on the right and can expand into a dialog; its
// state lives in the chat store, so switching keeps the conversation.
function onOpenChange(open: boolean): void {
    if (!open) chat.open = false
}
</script>

<template>
    <Sheet :open="chat.open && !chat.expanded" @update:open="onOpenChange">
        <SheetContent
            side="right"
            class="flex h-full max-h-screen w-[420px] flex-col gap-0 overflow-hidden p-0 sm:max-w-[480px]"
        >
            <SheetTitle class="sr-only">Copilot</SheetTitle>
            <SheetDescription class="sr-only"
                >Chat com o copiloto de conteúdo</SheetDescription
            >
            <ChatPanel />
        </SheetContent>
    </Sheet>
    <Dialog :open="chat.open && chat.expanded" @update:open="onOpenChange">
        <DialogContent
            class="flex h-[88vh] w-[min(1100px,calc(100vw-2rem))] max-w-none flex-col gap-0 overflow-hidden p-0 sm:max-w-none"
        >
            <DialogTitle class="sr-only">Copilot</DialogTitle>
            <DialogDescription class="sr-only"
                >Chat com o copiloto de conteúdo</DialogDescription
            >
            <ChatPanel />
        </DialogContent>
    </Dialog>
</template>
