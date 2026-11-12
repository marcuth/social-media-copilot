<script setup lang="ts">
import { reactive, ref } from "vue"
import { Plus, Star, Trash2 } from "lucide-vue-next"

import AppModal from "../components/AppModal.vue"
import { ProviderService, errorMessage, type AIProvider } from "../lib/api"
import { refreshProviders, store } from "../lib/store"
import { toast, toastError } from "../lib/toast"

type Kind = "openai_compatible" | "anthropic" | "google" | "ollama"

const KINDS: { id: Kind; label: string; hint: string }[] = [
    { id: "openai_compatible", label: "Compatível com OpenAI", hint: "OpenAI, DeepSeek, Groq, OpenRouter, LM Studio e outros." },
    { id: "anthropic", label: "Anthropic (Claude)", hint: "API nativa da Anthropic." },
    { id: "google", label: "Google (Gemini)", hint: "API do Google AI Studio." },
    { id: "ollama", label: "Ollama", hint: "Modelos locais pelo Ollama." },
]

const PRESETS: Omit<AIProvider, "id" | "isDefault" | "createdAt" | "updatedAt">[] = [
    { name: "Ollama (local)", kind: "ollama", baseUrl: "http://localhost:11434", apiKey: "", model: "llama3.2:latest" },
    { name: "LM Studio (local)", kind: "openai_compatible", baseUrl: "http://localhost:1234/v1", apiKey: "", model: "local-model" },
    { name: "OpenAI", kind: "openai_compatible", baseUrl: "https://api.openai.com/v1", apiKey: "", model: "gpt-4o-mini" },
    { name: "Anthropic", kind: "anthropic", baseUrl: "", apiKey: "", model: "claude-opus-5-5" },
    { name: "Google Gemini", kind: "google", baseUrl: "", apiKey: "", model: "gemini-2.5-flash" },
    { name: "DeepSeek", kind: "openai_compatible", baseUrl: "https://api.deepseek.com/v1", apiKey: "", model: "deepseek-chat" },
    { name: "Groq", kind: "openai_compatible", baseUrl: "https://api.groq.com/openai/v1", apiKey: "", model: "llama-3.3-70b-versatile" },
    { name: "OpenRouter", kind: "openai_compatible", baseUrl: "https://openrouter.ai/api/v1", apiKey: "", model: "openrouter/auto" },
]

function blank(): AIProvider {
    return { id: 0, name: "", kind: "openai_compatible", baseUrl: "", apiKey: "", model: "", isDefault: false, createdAt: "", updatedAt: "" }
}

const editing = ref<AIProvider | null>(null)
const test = reactive({ busy: false, result: "", ok: false })

function open(p?: AIProvider | (typeof PRESETS)[number]): void {
    editing.value = { ...blank(), ...(p ?? {}) }
    test.result = ""
}

const kindLabel = (k: string): string => KINDS.find((x) => x.id === k)?.label ?? k

async function save(): Promise<void> {
    if (!editing.value) return
    try {
        await ProviderService.SaveProvider(editing.value)
        await refreshProviders()
        toast("Provedor salvo.")
        editing.value = null
    } catch (e) {
        toastError(e)
    }
}

async function runTest(): Promise<void> {
    if (!editing.value) return
    test.busy = true
    test.result = ""
    try {
        const reply = await ProviderService.TestProvider(editing.value)
        test.ok = true
        test.result = `Funcionou. Resposta: “${reply.slice(0, 80)}”`
    } catch (e) {
        test.ok = false
        test.result = errorMessage(e)
    } finally {
        test.busy = false
    }
}

async function makeDefault(p: AIProvider): Promise<void> {
    try {
        await ProviderService.SetDefaultProvider(p.id)
        await refreshProviders()
    } catch (e) {
        toastError(e)
    }
}

async function remove(p: AIProvider): Promise<void> {
    if (!confirm(`Remover o provedor "${p.name}"?`)) return
    try {
        await ProviderService.DeleteProvider(p.id)
        await refreshProviders()
    } catch (e) {
        toastError(e)
    }
}
</script>

<template>
    <div class="page settings">
        <header class="page-head">
            <div>
                <h1>Configurações</h1>
                <p>Provedores de IA usados para roteiros, metadados e ideias.</p>
            </div>
            <button class="btn" @click="open()"><Plus /> Provedor personalizado</button>
        </header>

        <section class="panel block">
            <h2>Seus provedores</h2>
            <p v-if="!store.providers.length" class="muted">
                Nenhum provedor ainda. Escolha um modelo pronto abaixo ou adicione um personalizado.
            </p>
            <ul class="providers">
                <li v-for="p in store.providers" :key="p.id" class="provider">
                    <div class="grow">
                        <strong>{{ p.name }}</strong>
                        <span v-if="p.isDefault" class="chip default">Em uso</span>
                        <p class="muted">{{ kindLabel(p.kind) }} · {{ p.model }}</p>
                    </div>
                    <button v-if="!p.isDefault" class="btn ghost sm" @click="makeDefault(p)"><Star /> Usar este</button>
                    <button class="btn sm" @click="open(p)">Editar</button>
                    <button class="btn ghost icon sm danger" :aria-label="`Remover ${p.name}`" @click="remove(p)">
                        <Trash2 />
                    </button>
                </li>
            </ul>
        </section>

        <section class="panel block">
            <h2>Adicionar a partir de um modelo pronto</h2>
            <div class="presets">
                <button v-for="preset in PRESETS" :key="preset.name" class="preset" @click="open(preset)">
                    <strong>{{ preset.name }}</strong>
                    <span class="muted">{{ preset.model }}</span>
                </button>
            </div>
        </section>

        <p class="muted note">
            As chaves de API ficam salvas apenas neste computador, no banco de dados local do app.
        </p>

        <AppModal v-if="editing" :title="editing.id ? 'Editar provedor' : 'Novo provedor'" width="520px" @close="editing = null">
            <form class="form" @submit.prevent="save">
                <label class="field">
                    <span>Nome</span>
                    <input v-model="editing.name" class="input" />
                </label>
                <label class="field">
                    <span>Tipo</span>
                    <select v-model="editing.kind" class="select">
                        <option v-for="k in KINDS" :key="k.id" :value="k.id">{{ k.label }}</option>
                    </select>
                    <small>{{ KINDS.find((k) => k.id === editing!.kind)?.hint }}</small>
                </label>
                <label class="field">
                    <span>URL base {{ editing.kind === "openai_compatible" ? "" : "(opcional)" }}</span>
                    <input v-model="editing.baseUrl" class="input" placeholder="https://…" />
                </label>
                <label class="field">
                    <span>Chave de API {{ ["anthropic", "google"].includes(editing.kind) ? "" : "(opcional para locais)" }}</span>
                    <input v-model="editing.apiKey" type="password" class="input" autocomplete="off" />
                </label>
                <label class="field">
                    <span>Modelo</span>
                    <input v-model="editing.model" class="input" />
                </label>
                <label class="check">
                    <input v-model="editing.isDefault" type="checkbox" />
                    Usar este provedor nas funções de IA
                </label>
                <p v-if="test.result" class="test" :class="{ ok: test.ok }">{{ test.result }}</p>
            </form>
            <template #footer>
                <button class="btn ghost" :disabled="test.busy" @click="runTest">
                    {{ test.busy ? "Testando…" : "Testar conexão" }}
                </button>
                <span class="grow"></span>
                <button class="btn ghost" @click="editing = null">Cancelar</button>
                <button class="btn primary" @click="save">Salvar provedor</button>
            </template>
        </AppModal>
    </div>
</template>

<style scoped>
.settings {
    max-width: 920px;
}

.block {
    display: flex;
    flex-direction: column;
    gap: 12px;
    padding: 20px;
    margin-bottom: 16px;
}

.providers {
    list-style: none;
    margin: 0;
    padding: 0;
    display: flex;
    flex-direction: column;
}

.provider {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 12px 0;
    border-top: 1px solid var(--line);
}

.provider:first-child {
    border-top: 0;
}

.default {
    margin-left: 8px;
    background: var(--ok-soft);
    color: var(--ok);
}

.presets {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(190px, 1fr));
    gap: 8px;
}

.preset {
    display: flex;
    flex-direction: column;
    align-items: flex-start;
    gap: 2px;
    padding: 12px 14px;
    border: 1px solid var(--line);
    border-radius: var(--r-md);
    background: var(--surface);
    color: var(--ink);
    font: inherit;
    text-align: left;
    cursor: pointer;
}

.preset:hover {
    border-color: var(--ink);
}

.preset .muted {
    font-size: 12.5px;
}

.form {
    display: flex;
    flex-direction: column;
    gap: 12px;
}

.test {
    padding: 10px 12px;
    border-radius: var(--r-sm);
    background: var(--rec-soft);
    color: var(--rec);
    font-size: 13px;
    overflow-wrap: anywhere;
}

.test.ok {
    background: var(--ok-soft);
    color: var(--ok);
}

.note {
    font-size: 12.5px;
}
</style>
