import { reactive } from "vue"

import { errorMessage } from "./api"

export interface Toast {
    id: number
    text: string
    kind: "info" | "error"
}

export const toasts = reactive<Toast[]>([])
let seq = 0

export function toast(text: string, kind: Toast["kind"] = "info"): void {
    const id = ++seq
    toasts.push({ id, text, kind })
    setTimeout(
        () => {
            const i = toasts.findIndex((t) => t.id === id)
            if (i >= 0) toasts.splice(i, 1)
        },
        kind === "error" ? 7000 : 3500,
    )
}

export function toastError(e: unknown): void {
    toast(errorMessage(e), "error")
}
