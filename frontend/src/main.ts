import "./styles/main.css"
import "vue-sonner/style.css"

import { createApp } from "vue"

import App from "./App.vue"
import { initTheme } from "./lib/theme"
import { router } from "./router"

initTheme()
createApp(App).use(router).mount("#app")
