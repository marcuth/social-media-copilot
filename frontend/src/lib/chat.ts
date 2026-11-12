import { Events } from "@wailsio/runtime"
import { reactive } from "vue"

import {
    AiDelta,
    AiStatus,
    ChatService,
    errorMessage,
    isCancel,
    type ChatMessage,
    type ChatSession,
    type Post,
} from "./api"
import { store } from "./store"

export const chat = reactive({
    open: false,
    expanded: false,
    sessions: [] as ChatSession[],
    activeSessionId: 0,
    messages: [] as ChatMessage[],
    streaming: false,
    streamingContent: "",
    /** What a running answer is doing, e.g. searching the web. */
    status: "",
    error: "",
    /** Text being typed, kept when switching between sheet and dialog. */
    draft: "",
    /** Post open in the editor (with unsaved edits), sent as context. */
    post: null as Post | null,
})

let requestId = ""
let call: { cancel(): void } | null = null

Events.On(AiStatus, (ev: { data: { requestId: string; text: string } }) => {
    if (ev.data.requestId === requestId) chat.status = ev.data.text
})

Events.On(AiDelta, (ev: { data: { requestId: string; delta: string } }) => {
    if (ev.data.requestId !== requestId) return
    chat.streamingContent += ev.data.delta
    chat.status = ""
})

export async function openChat(): Promise<void> {
    chat.open = true
    await refreshSessions()
}

export async function refreshSessions(): Promise<void> {
    chat.sessions = (await ChatService.ListSessions()) ?? []
}

export function newSession(): void {
    chat.activeSessionId = 0
    chat.messages = []
    chat.error = ""
}

export async function switchSession(id: number): Promise<void> {
    chat.activeSessionId = id
    chat.error = ""
    chat.messages = (await ChatService.GetMessages(id)) ?? []
}

export async function deleteSession(id: number): Promise<void> {
    await ChatService.DeleteSession(id)
    if (chat.activeSessionId === id) newSession()
    await refreshSessions()
}

export function stopStreaming(): void {
    call?.cancel()
}

export async function sendMessage(text: string): Promise<void> {
    const content = text.trim()
    if (!content || chat.streaming) return
    chat.error = ""
    // Show the user's message right away; the server saves it too.
    chat.messages.push({
        id: -Date.now(),
        sessionId: chat.activeSessionId,
        role: "user",
        content,
        model: "",
        createdAt: new Date().toISOString(),
    })
    requestId = crypto.randomUUID()
    chat.streaming = true
    chat.streamingContent = ""
    chat.status = ""
    const pending = ChatService.SendMessage({
        sessionId: chat.activeSessionId,
        requestId,
        content,
        profileId: store.activeProfileId,
        post: chat.post,
    })
    call = pending
    try {
        const reply = await pending
        chat.activeSessionId = reply.session.id
        chat.messages = (await ChatService.GetMessages(reply.session.id)) ?? []
    } catch (e) {
        if (isCancel(e)) {
            if (chat.streamingContent) {
                chat.messages.push({
                    id: -Date.now(),
                    sessionId: chat.activeSessionId,
                    role: "assistant",
                    content: chat.streamingContent,
                    model: "",
                    createdAt: new Date().toISOString(),
                })
            }
        } else {
            chat.error = errorMessage(e)
        }
    } finally {
        chat.streaming = false
        chat.streamingContent = ""
        chat.status = ""
        call = null
        await refreshSessions()
    }
}
