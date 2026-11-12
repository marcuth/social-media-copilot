<script setup lang="ts">
import {
    CalendarDays,
    Columns3,
    Layers,
    Lightbulb,
    Plus,
    Settings,
    Users,
} from "@lucide/vue"
import { useRoute } from "vue-router"

import {
    Sidebar,
    SidebarContent,
    SidebarGroup,
    SidebarGroupAction,
    SidebarGroupContent,
    SidebarGroupLabel,
    SidebarHeader,
    SidebarMenu,
    SidebarMenuButton,
    SidebarMenuItem,
    SidebarSeparator,
} from "@/components/ui/sidebar"
import { setActiveProfile, store } from "@/lib/store"
import { Button } from "@/components/ui/button"

const route = useRoute()

const nav = [
    {
        to: "/",
        label: "Calendário",
        icon: CalendarDays,
        match: (p: string) => p === "/",
    },
    {
        to: "/pipeline",
        label: "Produção",
        icon: Columns3,
        match: (p: string) => p.startsWith("/pipeline"),
    },
    {
        to: "/ideas",
        label: "Ideias com IA",
        icon: Lightbulb,
        match: (p: string) => p.startsWith("/ideas"),
    },
    {
        to: "/profiles",
        label: "Perfis e temas",
        icon: Users,
        match: (p: string) => p.startsWith("/profiles"),
    },
    {
        to: "/settings",
        label: "Configurações",
        icon: Settings,
        match: (p: string) => p.startsWith("/settings"),
    },
]
</script>

<template>
    <Sidebar class="border-r border-border/50" collapsible="icon">
        <SidebarHeader class="p-4">
            <div class="flex items-center gap-3">
                <img src="/icon.png" alt="" class="size-8 shrink-0" />
                <div class="flex flex-col group-data-[collapsible=icon]:hidden">
                    <span class="font-heading text-sm leading-tight font-bold"
                        >Social Media Copilot</span
                    >
                    <span class="text-xs text-muted-foreground">v0.1.2</span>
                </div>
            </div>
        </SidebarHeader>

        <SidebarSeparator />

        <SidebarContent>
            <SidebarGroup>
                <SidebarGroupContent>
                    <SidebarMenu>
                        <SidebarMenuItem>
                            <SidebarMenuButton as-child tooltip="Novo post">
                                <RouterLink
                                    :to="{
                                        name: 'post',
                                        params: { id: 'new' },
                                    }"
                                    class="font-semibold text-primary"
                                >
                                    <Plus />
                                    <span>Novo post</span>
                                </RouterLink>
                            </SidebarMenuButton>
                        </SidebarMenuItem>
                        <SidebarMenuItem v-for="item in nav" :key="item.to">
                            <SidebarMenuButton
                                as-child
                                :is-active="item.match(route.path)"
                                :tooltip="item.label"
                            >
                                <RouterLink :to="item.to">
                                    <component :is="item.icon" />
                                    <span>{{ item.label }}</span>
                                </RouterLink>
                            </SidebarMenuButton>
                        </SidebarMenuItem>
                    </SidebarMenu>
                </SidebarGroupContent>
            </SidebarGroup>

            <SidebarSeparator />

            <SidebarGroup>
                <SidebarGroupLabel>Perfis</SidebarGroupLabel>
                <SidebarGroupAction as-child title="Novo perfil">
                    <RouterLink to="/profiles?new=1"><Plus /></RouterLink>
                </SidebarGroupAction>
                <SidebarGroupContent>
                    <SidebarMenu>
                        <SidebarMenuItem>
                            <SidebarMenuButton
                                :is-active="store.activeProfileId === null"
                                tooltip="Todos os perfis"
                                @click="setActiveProfile(null)"
                            >
                                <Layers />
                                <span>Todos os perfis</span>
                            </SidebarMenuButton>
                        </SidebarMenuItem>
                        <SidebarMenuItem
                            v-for="p in store.profiles"
                            :key="p.id"
                        >
                            <SidebarMenuButton
                                :is-active="store.activeProfileId === p.id"
                                :tooltip="p.name"
                                @click="setActiveProfile(p.id)"
                            >
                                <span
                                    class="flex size-4 shrink-0 items-center justify-center rounded-[5px] text-[10px] font-bold text-white"
                                    :style="{ background: p.color }"
                                    >{{ p.name.charAt(0).toUpperCase() }}</span
                                >
                                <span class="truncate">{{ p.name }}</span>
                            </SidebarMenuButton>
                        </SidebarMenuItem>
                    </SidebarMenu>
                    <div
                        v-if="store.ready && !store.profiles.length"
                        class="px-2 py-4 text-center group-data-[collapsible=icon]:hidden"
                    >
                        <p class="text-xs text-muted-foreground">
                            Nenhum perfil ainda
                        </p>
                        <Button
                            variant="outline"
                            size="sm"
                            class="mt-2 h-7 text-xs"
                            as-child
                        >
                            <RouterLink to="/profiles?new=1"
                                ><Plus class="size-3" /> Criar
                                perfil</RouterLink
                            >
                        </Button>
                    </div>
                </SidebarGroupContent>
            </SidebarGroup>
        </SidebarContent>
    </Sidebar>
</template>
