<script setup lang="ts">
import { onMounted, ref } from "vue"
import { CalendarDays, Columns3, Lightbulb, Plus, Settings, Users } from "lucide-vue-next"

import { errorMessage } from "./lib/api"
import { loadAll, setActiveProfile, store } from "./lib/store"
import { toasts } from "./lib/toast"

const loadError = ref("")

const nav = [
    { to: "/", label: "Calendário", icon: CalendarDays },
    { to: "/pipeline", label: "Produção", icon: Columns3 },
    { to: "/ideas", label: "Ideias com IA", icon: Lightbulb },
    { to: "/profiles", label: "Perfis e temas", icon: Users },
    { to: "/settings", label: "Configurações", icon: Settings },
]

onMounted(async () => {
    try {
        await loadAll()
    } catch (e) {
        loadError.value = errorMessage(e)
    }
})
</script>

<template>
    <div v-if="loadError" class="fatal">
        <h1>Não foi possível abrir os dados</h1>
        <p>{{ loadError }}</p>
    </div>
    <div v-else class="shell">
        <aside class="side">
            <div class="brand">
                <span class="rec" aria-hidden="true"></span>
                <span>Social Media<br />Copilot</span>
            </div>

            <RouterLink :to="{ name: 'post', params: { id: 'new' } }" class="btn primary new">
                <Plus /> Novo post
            </RouterLink>

            <nav aria-label="Principal">
                <RouterLink
                    v-for="item in nav"
                    :key="item.to"
                    :to="item.to"
                    class="nav-link"
                    :class="{ exact: item.to === '/' }"
                >
                    <component :is="item.icon" :size="17" />
                    {{ item.label }}
                </RouterLink>
            </nav>

            <section class="profiles" aria-label="Filtrar por perfil">
                <h3>Perfis</h3>
                <button
                    class="profile-filter"
                    :class="{ on: store.activeProfileId === null }"
                    @click="setActiveProfile(null)"
                >
                    <span class="swatch all"></span>
                    Todos os perfis
                </button>
                <button
                    v-for="p in store.profiles"
                    :key="p.id"
                    class="profile-filter"
                    :class="{ on: store.activeProfileId === p.id }"
                    @click="setActiveProfile(p.id)"
                >
                    <span class="swatch" :style="{ background: p.color }"></span>
                    <span class="name">{{ p.name }}</span>
                </button>
                <RouterLink v-if="!store.profiles.length && store.ready" to="/profiles" class="hint">
                    Crie seu primeiro perfil para começar a planejar.
                </RouterLink>
            </section>
        </aside>

        <main class="main">
            <RouterView v-if="store.ready" v-slot="{ Component, route }">
                <component :is="Component" :key="route.fullPath" />
            </RouterView>
        </main>

        <div class="toasts" role="status" aria-live="polite">
            <div v-for="t in toasts" :key="t.id" class="toast" :class="t.kind">{{ t.text }}</div>
        </div>
    </div>
</template>

<style scoped>
.shell {
    display: grid;
    grid-template-columns: var(--sidebar) 1fr;
    height: 100%;
}

.side {
    display: flex;
    flex-direction: column;
    gap: 18px;
    padding: 20px 14px;
    border-right: 1px solid var(--line);
    background: var(--surface);
    overflow-y: auto;
}

.brand {
    display: flex;
    align-items: center;
    gap: 10px;
    padding: 0 8px;
    font-family: var(--font-display);
    font-weight: 700;
    font-size: 15px;
    line-height: 1.1;
    font-variation-settings: "wdth" 80;
}

.rec {
    width: 12px;
    height: 12px;
    border-radius: 50%;
    background: var(--rec);
    box-shadow: 0 0 0 4px var(--rec-soft);
}

.new {
    width: 100%;
    text-decoration: none;
}

nav {
    display: flex;
    flex-direction: column;
    gap: 2px;
}

.nav-link {
    display: flex;
    align-items: center;
    gap: 10px;
    height: 36px;
    padding: 0 10px;
    border-radius: var(--r-sm);
    color: var(--ink-2);
    text-decoration: none;
    font-weight: 600;
}

.nav-link:hover {
    background: var(--surface-2);
    color: var(--ink);
}

.nav-link.router-link-active:not(.exact),
.nav-link.exact.router-link-exact-active {
    background: var(--cobalt-soft);
    color: var(--cobalt);
}

.profiles {
    display: flex;
    flex-direction: column;
    gap: 2px;
    padding-top: 14px;
    border-top: 1px solid var(--line);
}

.profiles h3 {
    padding: 0 10px 6px;
    font-size: 12.5px;
    color: var(--muted);
    font-family: var(--font-body);
}

.profile-filter {
    display: flex;
    align-items: center;
    gap: 10px;
    height: 32px;
    padding: 0 10px;
    border: 0;
    border-radius: var(--r-sm);
    background: transparent;
    color: var(--ink-2);
    font: inherit;
    text-align: left;
    cursor: pointer;
}

.profile-filter:hover {
    background: var(--surface-2);
}

.profile-filter.on {
    color: var(--ink);
    font-weight: 700;
    background: var(--surface-2);
}

.profile-filter .name {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
}

.swatch {
    width: 12px;
    height: 12px;
    border-radius: 4px;
    flex: none;
}

.swatch.all {
    background: conic-gradient(#3f51d9 0 25%, #e8336d 0 50%, #1d8f63 0 75%, #d9822b 0);
}

.hint {
    padding: 4px 10px;
    font-size: 13px;
}

.main {
    overflow-y: auto;
    min-width: 0;
}

.toasts {
    position: fixed;
    right: 20px;
    bottom: 20px;
    z-index: 100;
    display: flex;
    flex-direction: column;
    gap: 8px;
    max-width: 380px;
}

.toast {
    padding: 11px 14px;
    border-radius: var(--r-md);
    background: var(--ink);
    color: var(--paper);
    box-shadow: var(--shadow-pop);
    animation: toast-in 180ms ease-out;
}

.toast.error {
    background: var(--rec);
    color: #fff;
}

.fatal {
    padding: 48px;
}

@keyframes toast-in {
    from {
        opacity: 0;
        transform: translateY(6px);
    }
}
</style>
