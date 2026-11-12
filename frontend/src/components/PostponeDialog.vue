<script setup lang="ts">
import { computed, ref } from "vue"
import { CalendarClock } from "@lucide/vue"

import { Button } from "@/components/ui/button"
import { Checkbox } from "@/components/ui/checkbox"
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { PostService, normPost, type Post, type PostN } from "@/lib/api"
import { addDays, formatDay, formatTime, parseSchedule, toSchedule } from "@/lib/dates"
import { toast, toastError } from "@/lib/toast"
import { cn } from "@/lib/utils"

const props = defineProps<{ post: Post }>()
const emit = defineEmits<{ close: []; done: [PostN] }>()

type Choice = "1" | "3" | "7" | "next_free" | "date"

const current = computed(() => parseSchedule(props.post.scheduledAt))
const choice = ref<Choice>(current.value ? "1" : "date")
const date = ref(toSchedule(addDays(current.value ?? new Date(), 1)))
const reason = ref("")
const cascade = ref(false)
const busy = ref(false)

const options: { id: Choice; label: string }[] = [
    { id: "1", label: "Amanhã (+1 dia)" },
    { id: "3", label: "+3 dias" },
    { id: "7", label: "+1 semana" },
    { id: "next_free", label: "Próximo dia livre" },
    { id: "date", label: "Escolher data" },
]

const preview = computed(() => {
    if (choice.value === "next_free") return "o primeiro dia sem posts deste perfil, no mesmo horário"
    let d: Date | null = null
    if (choice.value === "date") d = parseSchedule(date.value)
    else if (current.value) d = addDays(current.value, Number(choice.value))
    return d ? `${formatDay(d)}, ${formatTime(d)}` : "—"
})

async function confirm(): Promise<void> {
    busy.value = true
    try {
        const res = await PostService.Postpone({
            postId: props.post.id,
            mode: choice.value === "date" ? "date" : choice.value === "next_free" ? "next_free" : "days",
            days: Number(choice.value) || 0,
            date: date.value,
            reason: reason.value,
            cascade: cascade.value,
        })
        const others = res.movedCount - 1
        toast(others > 0 ? `Post adiado. ${others} post(s) seguintes também foram empurrados.` : "Post adiado.")
        emit("done", normPost(res.post))
    } catch (e) {
        toastError(e)
    } finally {
        busy.value = false
    }
}
</script>

<template>
    <Dialog :open="true" @update:open="(open) => !open && emit('close')">
        <DialogContent class="sm:max-w-[480px]">
            <DialogHeader>
                <DialogTitle class="flex items-center gap-2">
                    <CalendarClock class="size-5 text-primary" /> Adiar post
                </DialogTitle>
                <DialogDescription>
                    <strong class="text-foreground">{{ post.title }}</strong>
                    {{ current ? `está marcado para ${formatDay(current)}, ${formatTime(current)}.` : "ainda não tem data." }}
                </DialogDescription>
            </DialogHeader>

            <div class="space-y-4">
                <div class="flex flex-wrap gap-2" role="radiogroup" aria-label="Para quando">
                    <button
                        v-for="o in options"
                        :key="o.id"
                        type="button"
                        role="radio"
                        :aria-checked="choice === o.id"
                        :disabled="!current && o.id !== 'date'"
                        :class="
                            cn(
                                'h-9 cursor-pointer rounded-md border-2 px-3 text-sm font-medium transition-colors outline-none focus-visible:ring-[3px] focus-visible:ring-ring/50 disabled:cursor-not-allowed disabled:opacity-40',
                                choice === o.id
                                    ? 'border-primary bg-primary/10'
                                    : 'border-border/50 bg-card hover:border-primary/30',
                            )
                        "
                        @click="choice = o.id"
                    >
                        {{ o.label }}
                    </button>
                </div>

                <div v-if="choice === 'date'" class="space-y-2">
                    <Label for="postpone-date">Nova data</Label>
                    <Input id="postpone-date" v-model="date" type="datetime-local" />
                </div>

                <div class="rounded-lg bg-muted p-4 text-sm">
                    Vai para <strong>{{ preview }}</strong>
                </div>

                <div class="space-y-2">
                    <Label for="postpone-reason">Motivo (opcional)</Label>
                    <Input id="postpone-reason" v-model="reason" placeholder="Ex.: falta gravar a cena externa" />
                </div>

                <Label v-if="current" class="flex items-start gap-3 font-normal">
                    <Checkbox v-model="cascade" class="mt-0.5" />
                    <span>
                        <span class="block font-medium">Empurrar a fila deste perfil</span>
                        <span class="text-xs text-muted-foreground">
                            Os próximos posts não publicados andam a mesma distância, mantendo o espaçamento.
                        </span>
                    </span>
                </Label>
            </div>

            <DialogFooter>
                <Button variant="outline" @click="emit('close')">Cancelar</Button>
                <Button :disabled="busy" @click="confirm">Adiar post</Button>
            </DialogFooter>
        </DialogContent>
    </Dialog>
</template>
