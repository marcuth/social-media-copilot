import { createApp } from "vue"
import "vue-sonner/style.css"

import { initTheme } from "./lib/theme"
import { router } from "./router"
import App from "./App.vue"
import "./styles/main.css"

initTheme()
createApp(App).use(router).mount("#app")
