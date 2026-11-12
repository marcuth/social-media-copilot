<script setup lang="ts">
import { TriangleAlert } from "@lucide/vue"
import { onMounted, ref } from "vue"

import { SidebarInset, SidebarProvider } from "@/components/ui/sidebar"
import ConfirmDialog from "@/components/ConfirmDialog.vue"
import { TooltipProvider } from "@/components/ui/tooltip"
import CopilotChat from "@/components/CopilotChat.vue"
import AppSidebar from "@/components/AppSidebar.vue"
import { Toaster } from "@/components/ui/sonner"
import { loadAll, store } from "@/lib/store"
import { errorMessage } from "@/lib/api"
import { theme } from "@/lib/theme"

const loadError = ref("")

onMounted(async () => {
    try {
        await loadAll()
    } catch (e) {
        loadError.value = errorMessage(e)
    }
})
</script>

<template>
    <TooltipProvider :delay-duration="300">
        <div
            v-if="loadError"
            class="flex h-screen flex-col items-center justify-center gap-3 p-8 text-center"
        >
            <TriangleAlert class="size-8 text-destructive" />
            <h1 class="font-heading text-2xl font-bold">
                Não foi possível abrir os dados
            </h1>
            <p class="max-w-md text-sm text-muted-foreground">
                {{ loadError }}
            </p>
        </div>
        <SidebarProvider v-else>
            <AppSidebar />
            <SidebarInset class="h-screen overflow-hidden">
                <RouterView v-if="store.ready" v-slot="{ Component, route }">
                    <component :is="Component" :key="route.path" />
                </RouterView>
            </SidebarInset>
        </SidebarProvider>
        <CopilotChat v-if="store.ready" />
        <ConfirmDialog />
        <Toaster rich-colors position="top-center" :theme="theme" />
    </TooltipProvider>
</template>
