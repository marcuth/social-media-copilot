<script setup lang="ts">
import { computed, ref } from "vue"

import AppModal from "./AppModal.vue"
import { PostService, normPost, type Post, type PostN } from "../lib/api"
import { addDays, formatDay, formatTime, parseSchedule, toSchedule } from "../lib/dates"
import { toast, toastError } from "../lib/toast"

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
        const days = Number(choice.value) || 0
        const res = await PostService.Postpone({
            postId: props.post.id,
            mode: choice.value === "date" ? "date" : choice.value === "next_free" ? "next_free" : "days",
            days,
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
    <AppModal title="Adiar post" @close="emit('close')">
        <p class="what">
            <strong>{{ post.title }}</strong>
            <span class="muted">
                {{ current ? `está marcado para ${formatDay(current)}, ${formatTime(current)}` : "ainda não tem data" }}
            </span>
        </p>

        <div class="choices" role="radiogroup" aria-label="Para quando">
            <label
                v-for="o in options"
                :key="o.id"
                class="choice"
                :class="{ on: choice === o.id, disabled: !current && o.id !== 'date' }"
            >
                <input v-model="choice" type="radio" :value="o.id" :disabled="!current && o.id !== 'date'" />
                {{ o.label }}
            </label>
        </div>

        <label v-if="choice === 'date'" class="field">
            <span>Nova data</span>
            <input v-model="date" type="datetime-local" class="input" />
        </label>

        <p class="preview">
            Vai para <strong>{{ preview }}</strong>
        </p>

        <label class="field">
            <span>Motivo (opcional)</span>
            <input v-model="reason" class="input" placeholder="Ex.: falta gravar a cena externa" />
        </label>

        <label v-if="current" class="check cascade">
            <input v-model="cascade" type="checkbox" />
            <span>
                Empurrar a fila deste perfil
                <small class="muted">
                    Os próximos posts não publicados andam a mesma distância, mantendo o espaçamento entre eles.
                </small>
            </span>
        </label>

        <template #footer>
            <button class="btn ghost" @click="emit('close')">Cancelar</button>
            <button class="btn primary" :disabled="busy" @click="confirm">Adiar post</button>
        </template>
    </AppModal>
</template>

<style scoped>
.what {
    display: flex;
    flex-direction: column;
    margin-bottom: 16px;
}

.choices {
    display: flex;
    flex-wrap: wrap;
    gap: 6px;
    margin-bottom: 14px;
}

.choice {
    display: inline-flex;
    align-items: center;
    height: 32px;
    padding: 0 12px;
    border: 1px solid var(--line-strong);
    border-radius: 999px;
    font-weight: 600;
    font-size: 13px;
    cursor: pointer;
}

.choice input {
    position: absolute;
    opacity: 0;
    pointer-events: none;
}

.choice:focus-within {
    outline: 2px solid var(--cobalt);
    outline-offset: 2px;
}

.choice.on {
    background: var(--ink);
    border-color: var(--ink);
    color: var(--paper);
}

.choice.disabled {
    opacity: 0.4;
    cursor: not-allowed;
}

.preview {
    margin: 4px 0 16px;
    padding: 10px 12px;
    border-radius: var(--r-sm);
    background: var(--surface-2);
}

.cascade {
    margin-top: 14px;
}

.cascade small {
    display: block;
}
</style>
