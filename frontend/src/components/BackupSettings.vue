<script setup lang="ts">
import {
    DatabaseBackup,
    FolderOpen,
    RotateCcw,
    TriangleAlert,
} from "@lucide/vue"
import { onMounted, ref } from "vue"

import { BackupService, type BackupInfo } from "@/lib/api"
import { Button } from "@/components/ui/button"
import { toast, toastError } from "@/lib/toast"
import { confirmDialog } from "@/lib/confirm"

const info = ref<BackupInfo | null>(null)
const busy = ref(false)

const sizeLabel = (bytes: number): string =>
    bytes < 1024 * 1024
        ? `${Math.max(1, Math.round(bytes / 1024))} KB`
        : `${(bytes / 1024 / 1024).toFixed(1)} MB`

const dateLabel = (iso: string): string =>
    iso
        ? new Date(iso).toLocaleString("pt-BR", {
              dateStyle: "short",
              timeStyle: "short",
          })
        : "nunca"

async function load(): Promise<void> {
    try {
        info.value = await BackupService.GetBackupInfo()
    } catch (e) {
        toastError(e)
    }
}
onMounted(load)

async function exportBackup(): Promise<void> {
    busy.value = true
    try {
        const res = await BackupService.ExportBackup()
        if (!res.cancelled) {
            toast(`Backup salvo em ${res.path}`)
            await load()
        }
    } catch (e) {
        toastError(e)
    } finally {
        busy.value = false
    }
}

async function restore(): Promise<void> {
    const ok = await confirmDialog({
        title: "Restaurar backup",
        description:
            "Todos os dados atuais (perfis, posts, conversas, provedores e configurações) serão substituídos pelos do backup. Uma cópia do estado atual é salva antes, na pasta de dados.",
        confirmText: "Escolher backup",
        destructive: true,
    })
    if (!ok) return
    busy.value = true
    try {
        const res = await BackupService.ImportBackup()
        if (res.cancelled) return
        toast(
            `Backup restaurado: ${res.profiles} perfil(is) e ${res.posts} post(s). Recarregando…`,
        )
        setTimeout(() => window.location.reload(), 1200)
    } catch (e) {
        toastError(e)
    } finally {
        busy.value = false
    }
}

async function openFolder(): Promise<void> {
    try {
        await BackupService.OpenDataFolder()
    } catch (e) {
        toastError(e)
    }
}
</script>

<template>
    <div class="space-y-6">
        <div>
            <h2 class="font-heading text-lg font-bold">Backup e restauração</h2>
            <p class="text-sm text-muted-foreground">
                Salve uma cópia de todos os dados em um arquivo e restaure
                quando precisar, inclusive em outro computador.
            </p>
        </div>

        <div
            v-if="info"
            class="space-y-3 rounded-xl border border-border/60 bg-card p-5 shadow-xs"
        >
            <div class="grid gap-3 sm:grid-cols-2">
                <div>
                    <p class="text-xs font-medium text-muted-foreground">
                        Tamanho do banco
                    </p>
                    <p class="font-semibold">{{ sizeLabel(info.sizeBytes) }}</p>
                </div>
                <div>
                    <p class="text-xs font-medium text-muted-foreground">
                        Último backup
                    </p>
                    <p class="font-semibold">
                        {{ dateLabel(info.lastBackupAt) }}
                    </p>
                </div>
            </div>
            <div>
                <p class="text-xs font-medium text-muted-foreground">
                    Local dos dados
                </p>
                <p class="font-mono text-xs break-all">{{ info.path }}</p>
            </div>
        </div>

        <div class="grid gap-3 sm:grid-cols-3">
            <Button :disabled="busy" @click="exportBackup">
                <DatabaseBackup /> Fazer backup
            </Button>
            <Button variant="outline" :disabled="busy" @click="restore">
                <RotateCcw /> Restaurar backup
            </Button>
            <Button variant="ghost" @click="openFolder">
                <FolderOpen /> Abrir pasta de dados
            </Button>
        </div>

        <div
            class="flex gap-3 rounded-xl border border-amber-500/30 bg-amber-500/5 p-4 text-sm"
        >
            <TriangleAlert
                class="mt-0.5 size-4 shrink-0 text-amber-600 dark:text-amber-400"
            />
            <p>
                O arquivo de backup inclui as chaves de API dos provedores de IA
                e do Firecrawl. Guarde-o em um lugar seguro e não o compartilhe.
            </p>
        </div>
    </div>
</template>
