import { createRouter, createWebHashHistory } from "vue-router"

import CalendarView from "./views/CalendarView.vue"

export const router = createRouter({
    history: createWebHashHistory(),
    routes: [
        { path: "/", name: "calendar", component: CalendarView },
        { path: "/pipeline", name: "pipeline", component: () => import("./views/PipelineView.vue") },
        { path: "/posts/:id", name: "post", component: () => import("./views/PostEditor.vue"), props: true },
        { path: "/ideas", name: "ideas", component: () => import("./views/IdeasView.vue") },
        { path: "/profiles", name: "profiles", component: () => import("./views/ProfilesView.vue") },
        { path: "/settings", name: "settings", component: () => import("./views/SettingsView.vue") },
    ],
})
