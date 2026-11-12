<script setup lang="ts">
import { computed, onMounted, ref } from "vue"
import { BookOpen } from "@lucide/vue"
import { Browser } from "@wailsio/runtime"

import MarkdownView from "@/components/MarkdownView.vue"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from "@/components/ui/dialog"
import { Switch } from "@/components/ui/switch"
import { SkillService, type SkillInfo } from "@/lib/api"
import { toastError } from "@/lib/toast"
import { cn } from "@/lib/utils"

const skills = ref<SkillInfo[]>([])
const reading = ref<SkillInfo | null>(null)
const readingText = ref("")

const groups = computed(() => {
    const map = new Map<string, SkillInfo[]>()
    for (const s of skills.value) map.set(s.source, [...(map.get(s.source) ?? []), s])
    return [...map.entries()]
})

async function load(): Promise<void> {
    try {
        skills.value = (await SkillService.ListSkills()) ?? []
    } catch (e) {
        toastError(e)
    }
}
onMounted(load)

async function toggle(s: SkillInfo, enabled: boolean): Promise<void> {
    s.enabled = enabled
    try {
        await SkillService.SetSkillEnabled(s.id, enabled)
    } catch (e) {
        s.enabled = !enabled
        toastError(e)
    }
}

async function read(s: SkillInfo): Promise<void> {
    reading.value = s
    readingText.value = ""
    try {
        readingText.value = await SkillService.GetSkillText(s.id)
    } catch (e) {
        toastError(e)
    }
}
</script>

<template>
    <div class="space-y-6">
        <div>
            <h2 class="font-heading text-lg font-bold">Skills</h2>
            <p class="text-sm text-muted-foreground">
                Manuais de método que a IA segue automaticamente em cada tarefa. No chat, ela carrega a skill quando o
                assunto pede. Desligue as que não quiser usar.
            </p>
        </div>

        <section v-for="[source, list] in groups" :key="source" class="space-y-3">
            <div class="flex items-center gap-2">
                <h3 class="text-sm font-semibold">{{ source }}</h3>
                <Badge v-if="list[0].license" variant="outline">{{ list[0].license }}</Badge>
            </div>
            <ul class="space-y-3">
                <li
                    v-for="s in list"
                    :key="s.id"
                    :class="
                        cn(
                            'flex items-start gap-4 rounded-xl border bg-card p-4 shadow-xs transition-opacity',
                            s.enabled ? 'border-border/60' : 'border-border/40 opacity-60',
                        )
                    "
                >
                    <div class="min-w-0 flex-1 space-y-1">
                        <p class="font-semibold">{{ s.name }}</p>
                        <p class="text-sm text-muted-foreground">{{ s.description }}</p>
                        <p class="text-xs text-muted-foreground">{{ s.appliesTo }}</p>
                    </div>
                    <div class="flex shrink-0 items-center gap-2">
                        <Button variant="ghost" size="sm" @click="read(s)"><BookOpen /> Ler</Button>
                        <Switch
                            :model-value="s.enabled"
                            :aria-label="`${s.enabled ? 'Desligar' : 'Ligar'} ${s.name}`"
                            @update:model-value="(v) => toggle(s, Boolean(v))"
                        />
                    </div>
                </li>
            </ul>
            <p v-if="list[0].license" class="text-xs text-muted-foreground">
                Copiadas sem alterações de
                <button
                    class="cursor-pointer font-medium text-primary underline-offset-4 hover:underline"
                    @click="Browser.OpenURL('https://github.com/sergebulaev/instagram-skills')"
                >
                    sergebulaev/instagram-skills
                </button>
                (licença MIT, © 2026 Sergey Bulaev). Passos de publicação e de ferramentas externas são ignorados dentro
                do app.
            </p>
        </section>

        <Dialog :open="!!reading" @update:open="(o) => !o && (reading = null)">
            <DialogContent class="flex max-h-[85vh] flex-col sm:max-w-3xl">
                <DialogHeader>
                    <DialogTitle>{{ reading?.name }}</DialogTitle>
                    <DialogDescription>{{ reading?.appliesTo }}</DialogDescription>
                </DialogHeader>
                <div class="min-h-0 flex-1 overflow-y-auto rounded-lg bg-muted p-4">
                    <MarkdownView :source="readingText || 'Carregando…'" />
                </div>
            </DialogContent>
        </Dialog>
    </div>
</template>
