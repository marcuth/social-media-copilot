import { ref, watch } from "vue"

export type Theme = "light" | "dark" | "system"

const KEY = "smc:theme"
const media = window.matchMedia("(prefers-color-scheme: dark)")

export const theme = ref<Theme>((localStorage.getItem(KEY) as Theme) || "system")

function apply(): void {
    const dark = theme.value === "dark" || (theme.value === "system" && media.matches)
    document.documentElement.classList.toggle("dark", dark)
}

export function initTheme(): void {
    apply()
    media.addEventListener("change", apply)
    watch(theme, (value) => {
        localStorage.setItem(KEY, value)
        apply()
    })
}
