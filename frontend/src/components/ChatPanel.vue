<script setup lang="ts">
import {
    Bot,
    History,
    Loader,
    Maximize2,
    Minimize2,
    Plus,
    Send,
    Sparkles,
    Square,
    Trash2,
    User,
} from "@lucide/vue"
import { computed, nextTick, onMounted, ref, watch } from "vue"

import {
    DropdownMenu,
    DropdownMenuContent,
    DropdownMenuItem,
    DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import {
    InputGroup,
    InputGroupAddon,
    InputGroupButton,
    InputGroupTextarea,
} from "@/components/ui/input-group"
import {
    chat,
    deleteSession,
    newSession,
    sendMessage,
    stopStreaming,
    switchSession,
} from "@/lib/chat"
import MarkdownView from "@/components/MarkdownView.vue"
import { profileById, store } from "@/lib/store"
import { Button } from "@/components/ui/button"
import { confirmDialog } from "@/lib/confirm"
import type { ChatMessage } from "@/lib/api"
import { cn } from "@/lib/utils"

const scroller = ref<HTMLElement | null>(null)

const contextLabel = computed(() => {
    if (chat.post)
        return chat.post.title ? `Post “${chat.post.title}”` : "Novo post"
    return profileById(store.activeProfileId)?.name ?? "Todos os perfis"
})

const quickActions = computed(() =>
    chat.post
        ? [
              {
                  label: "🎬 Melhorar o roteiro",
                  prompt: "Revise o roteiro deste post e reescreva o que puder ficar mais forte, explicando as mudanças principais.",
              },
              {
                  label: "🪝 5 ganchos",
                  prompt: "Sugira 5 ganchos diferentes para os primeiros segundos deste post.",
              },
              {
                  label: "🏷️ Títulos alternativos",
                  prompt: "Sugira 5 títulos alternativos para este post, adequados às plataformas selecionadas.",
              },
              {
                  label: "🔁 Adaptar para outra plataforma",
                  prompt: "Adapte este conteúdo para outra plataforma do perfil, mudando o formato onde fizer sentido.",
              },
          ]
        : [
              {
                  label: "🗓️ Analisar minha agenda",
                  prompt: "Analise minha agenda das próximas semanas: frequência, equilíbrio entre temas e plataformas, e o que está atrasado. Sugira ajustes.",
              },
              {
                  label: "💡 Ideias para a semana",
                  prompt: "Sugira 5 ideias de conteúdo para esta semana, com formato e plataforma de cada uma.",
              },
              {
                  label: "📸 Otimizar perfil do Instagram",
                  prompt: "Quero otimizar meu perfil do Instagram. Me diga o que você precisa saber e depois proponha nome, bio e destaques.",
              },
              {
                  label: "🧭 Estratégia de 30 dias",
                  prompt: "Com base no nicho, no público e nos temas, proponha uma estratégia de conteúdo para os próximos 30 dias.",
              },
          ],
)

const canSend = computed(() => chat.draft.trim().length > 0 && !chat.streaming)

async function submit(): Promise<void> {
    if (!canSend.value) return
    const text = chat.draft
    chat.draft = ""
    await sendMessage(text)
}

function onKeydown(e: KeyboardEvent): void {
    if (e.key === "Enter" && !e.shiftKey) {
        e.preventDefault()
        submit()
    }
}

async function removeSession(id: number, title: string): Promise<void> {
    const ok = await confirmDialog({
        title: "Excluir conversa",
        description: `"${title}" será excluída.`,
        confirmText: "Excluir",
        destructive: true,
    })
    if (ok) await deleteSession(id)
}

async function scrollToEnd(): Promise<void> {
    await nextTick()
    if (scroller.value) scroller.value.scrollTop = scroller.value.scrollHeight
}
watch(() => [chat.messages.length, chat.streamingContent], scrollToEnd)
onMounted(scrollToEnd)

const streamingMessage = computed<ChatMessage>(() => ({
    id: 0,
    sessionId: chat.activeSessionId,
    role: "assistant",
    content: chat.streamingContent,
    model: "",
    createdAt: "",
}))
</script>

<template>
    <div
        class="flex shrink-0 flex-col gap-1.5 border-b border-border/50 p-4 pr-12"
    >
        <div class="flex items-center justify-between">
            <div class="flex min-w-0 items-center gap-2">
                <div
                    class="flex size-7 shrink-0 items-center justify-center rounded-md bg-primary"
                >
                    <Bot class="size-4 text-primary-foreground" />
                </div>
                <div class="min-w-0">
                    <p class="text-sm leading-tight font-semibold">Copilot</p>
                    <p class="truncate text-xs text-muted-foreground">
                        Contexto:
                        <span class="font-medium text-primary">{{
                            contextLabel
                        }}</span>
                    </p>
                </div>
            </div>
            <div class="flex items-center gap-1">
                <DropdownMenu>
                    <DropdownMenuTrigger as-child>
                        <Button
                            variant="ghost"
                            size="icon"
                            class="size-7"
                            aria-label="Conversas anteriores"
                            title="Conversas anteriores"
                            :disabled="chat.streaming"
                        >
                            <History class="size-4" />
                        </Button>
                    </DropdownMenuTrigger>
                    <DropdownMenuContent
                        align="end"
                        class="max-h-72 w-72 overflow-y-auto"
                    >
                        <p
                            v-if="!chat.sessions.length"
                            class="px-2 py-1.5 text-xs text-muted-foreground"
                        >
                            Nenhuma conversa ainda.
                        </p>
                        <DropdownMenuItem
                            v-for="s in chat.sessions"
                            :key="s.id"
                            :class="
                                cn(
                                    'group justify-between',
                                    s.id === chat.activeSessionId &&
                                        'bg-accent',
                                )
                            "
                            @select="switchSession(s.id)"
                        >
                            <span class="truncate text-xs">{{ s.title }}</span>
                            <button
                                class="cursor-pointer opacity-0 group-hover:opacity-70 hover:text-destructive hover:opacity-100"
                                :aria-label="`Excluir ${s.title}`"
                                @click.stop.prevent="
                                    removeSession(s.id, s.title)
                                "
                            >
                                <Trash2 class="size-3.5" />
                            </button>
                        </DropdownMenuItem>
                    </DropdownMenuContent>
                </DropdownMenu>
                <Button
                    variant="ghost"
                    size="icon"
                    class="size-7"
                    aria-label="Nova conversa"
                    title="Nova conversa"
                    :disabled="chat.streaming"
                    @click="newSession"
                >
                    <Plus class="size-4" />
                </Button>
                <Button
                    variant="ghost"
                    size="icon"
                    class="size-7"
                    :aria-label="
                        chat.expanded ? 'Recolher chat' : 'Expandir chat'
                    "
                    :title="chat.expanded ? 'Recolher chat' : 'Expandir chat'"
                    @click="chat.expanded = !chat.expanded"
                >
                    <component
                        :is="chat.expanded ? Minimize2 : Maximize2"
                        class="size-4"
                    />
                </Button>
            </div>
        </div>
    </div>

    <div ref="scroller" class="min-h-0 flex-1 overflow-y-auto">
        <div
            :class="
                cn(
                    'flex flex-col gap-4 px-4 py-4',
                    chat.expanded && 'mx-auto w-full max-w-3xl',
                )
            "
        >
            <div
                v-if="!chat.messages.length && !chat.streaming"
                class="flex flex-col items-center gap-4 py-8 text-center"
            >
                <div
                    class="flex size-12 items-center justify-center rounded-2xl bg-primary/10"
                >
                    <Sparkles class="size-6 text-primary" />
                </div>
                <div>
                    <p class="text-sm font-semibold">Olá! Sou seu Copilot</p>
                    <p
                        class="mt-1 text-xs leading-relaxed text-muted-foreground"
                    >
                        Conheço seus perfis, a agenda e o post que estiver
                        aberto. Peça ideias, roteiros, legendas, críticas ou
                        ajuda com o calendário.
                    </p>
                </div>
                <div
                    :class="
                        cn(
                            'grid w-full gap-2',
                            chat.expanded ? 'grid-cols-2' : 'grid-cols-1',
                        )
                    "
                >
                    <Button
                        v-for="action in quickActions"
                        :key="action.label"
                        variant="outline"
                        size="sm"
                        class="h-auto justify-start py-2 text-left text-xs whitespace-normal"
                        @click="sendMessage(action.prompt)"
                    >
                        {{ action.label }}
                    </Button>
                </div>
            </div>

            <div
                v-for="msg in chat.streaming && chat.streamingContent
                    ? [...chat.messages, streamingMessage]
                    : chat.messages"
                :key="msg.id"
                :data-align="msg.role === 'assistant' ? 'start' : 'end'"
                class="group/message flex w-full min-w-0 gap-2 text-sm data-[align=end]:flex-row-reverse"
            >
                <div
                    :class="
                        cn(
                            'flex size-6 shrink-0 items-center justify-center rounded-full',
                            msg.role === 'assistant'
                                ? 'bg-primary text-primary-foreground'
                                : 'bg-secondary text-secondary-foreground',
                        )
                    "
                >
                    <component
                        :is="msg.role === 'assistant' ? Bot : User"
                        class="size-3"
                    />
                </div>
                <div
                    class="flex w-full min-w-0 flex-col gap-1 group-data-[align=end]/message:items-end"
                >
                    <div
                        :class="
                            cn(
                                'w-fit max-w-[90%] min-w-0 overflow-hidden rounded-xl px-3 py-2 text-sm leading-relaxed break-words',
                                msg.role === 'assistant'
                                    ? 'bg-muted'
                                    : 'bg-primary text-primary-foreground',
                            )
                        "
                    >
                        <MarkdownView
                            v-if="msg.role === 'assistant'"
                            :source="msg.content"
                            class="[&>*:last-child]:mb-0"
                        />
                        <p v-else class="whitespace-pre-wrap">
                            {{ msg.content }}
                        </p>
                        <span
                            v-if="msg.id === 0"
                            class="ml-1 inline-flex items-center gap-0.5"
                        >
                            <span
                                class="size-1 animate-bounce rounded-full bg-current [animation-delay:-0.3s]"
                            ></span>
                            <span
                                class="size-1 animate-bounce rounded-full bg-current [animation-delay:-0.15s]"
                            ></span>
                            <span
                                class="size-1 animate-bounce rounded-full bg-current"
                            ></span>
                        </span>
                    </div>
                    <p v-if="msg.model" class="px-1 text-[10px] opacity-60">
                        {{ msg.model }}
                    </p>
                </div>
            </div>

            <div
                v-if="chat.streaming && !chat.streamingContent"
                class="flex items-center gap-2 text-muted-foreground"
            >
                <Loader class="size-3.5 animate-spin" />
                <span class="text-xs">Pensando...</span>
            </div>
            <div
                v-if="chat.error"
                class="rounded-lg border border-destructive/30 bg-destructive/10 p-3 text-xs text-destructive"
            >
                {{ chat.error }}
            </div>
            <div
                v-if="!store.providers.length"
                class="rounded-lg border border-border/60 bg-card p-3 text-xs text-muted-foreground"
            >
                Configure um provedor de IA em
                <RouterLink
                    to="/settings"
                    class="font-medium text-primary"
                    @click="chat.open = false"
                >
                    Configurações
                </RouterLink>
                para conversar.
            </div>
        </div>
    </div>

    <form
        class="shrink-0 border-t border-border/50 p-4"
        @submit.prevent="submit"
    >
        <InputGroup :class="cn(chat.expanded && 'mx-auto max-w-3xl')">
            <InputGroupTextarea
                v-model="chat.draft"
                placeholder="Pergunte algo ou peça um conteúdo... (Enter para enviar)"
                rows="3"
                class="resize-none text-sm"
                :disabled="chat.streaming"
                @keydown="onKeydown"
            />
            <InputGroupAddon align="block-end">
                <InputGroupButton
                    v-if="chat.streaming"
                    type="button"
                    size="icon-sm"
                    variant="outline"
                    class="ml-auto rounded-full"
                    aria-label="Parar"
                    title="Parar"
                    @click="stopStreaming"
                >
                    <Square />
                </InputGroupButton>
                <InputGroupButton
                    v-else
                    type="submit"
                    size="icon-sm"
                    variant="default"
                    class="ml-auto rounded-full"
                    :disabled="!canSend"
                    aria-label="Enviar"
                >
                    <Send />
                </InputGroupButton>
            </InputGroupAddon>
        </InputGroup>
    </form>
</template>
