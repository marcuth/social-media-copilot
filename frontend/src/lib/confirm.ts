import { reactive } from "vue"

export interface ConfirmOptions {
    title: string
    description: string
    confirmText?: string
    destructive?: boolean
}

interface ConfirmState extends ConfirmOptions {
    open: boolean
    resolve: ((ok: boolean) => void) | null
}

export const confirmState = reactive<ConfirmState>({
    open: false,
    title: "",
    description: "",
    resolve: null,
})

export function confirmDialog(options: ConfirmOptions): Promise<boolean> {
    confirmState.resolve?.(false)
    Object.assign(confirmState, {
        confirmText: undefined,
        destructive: false,
        ...options,
        open: true,
    })
    return new Promise((resolve) => {
        confirmState.resolve = resolve
    })
}

export function settleConfirm(ok: boolean): void {
    confirmState.open = false
    confirmState.resolve?.(ok)
    confirmState.resolve = null
}
