<script setup lang="ts">
import { TriangleAlert } from "@lucide/vue"
import { nextTick } from "vue"

import {
    AlertDialog,
    AlertDialogAction,
    AlertDialogCancel,
    AlertDialogContent,
    AlertDialogDescription,
    AlertDialogFooter,
    AlertDialogHeader,
    AlertDialogTitle,
} from "@/components/ui/alert-dialog"
import { confirmState, settleConfirm } from "@/lib/confirm"
import { buttonVariants } from "@/components/ui/button"

// Dismissing (Esc, overlay) cancels. The action button also closes the dialog,
// and that close arrives before its own click handler, so wait a tick to let
// settleConfirm(true) run first instead of resolving the promise with false.
function onOpenChange(open: boolean): void {
    if (open) return
    void nextTick(() => {
        if (confirmState.open) settleConfirm(false)
    })
}
</script>

<template>
    <AlertDialog :open="confirmState.open" @update:open="onOpenChange">
        <AlertDialogContent class="sm:max-w-[440px]">
            <AlertDialogHeader>
                <AlertDialogTitle class="flex items-center gap-2">
                    <TriangleAlert
                        v-if="confirmState.destructive"
                        class="size-5 text-destructive"
                    />
                    {{ confirmState.title }}
                </AlertDialogTitle>
                <AlertDialogDescription>{{
                    confirmState.description
                }}</AlertDialogDescription>
            </AlertDialogHeader>
            <AlertDialogFooter>
                <AlertDialogCancel @click="settleConfirm(false)"
                    >Cancelar</AlertDialogCancel
                >
                <AlertDialogAction
                    :class="
                        buttonVariants({
                            variant: confirmState.destructive
                                ? 'destructive'
                                : 'default',
                        })
                    "
                    @click="settleConfirm(true)"
                >
                    {{ confirmState.confirmText ?? "Confirmar" }}
                </AlertDialogAction>
            </AlertDialogFooter>
        </AlertDialogContent>
    </AlertDialog>
</template>
