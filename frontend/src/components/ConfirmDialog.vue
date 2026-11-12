<script setup lang="ts">
import { TriangleAlert } from "@lucide/vue"

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

function onOpenChange(open: boolean): void {
    if (!open && confirmState.open) settleConfirm(false)
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
