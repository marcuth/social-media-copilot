import { toast as sonner } from "vue-sonner"

import { errorMessage } from "./api"

export function toast(text: string, kind: "info" | "error" = "info"): void {
    if (kind === "error") sonner.error(text)
    else sonner.success(text)
}

export function toastError(e: unknown): void {
    sonner.error(errorMessage(e))
}
