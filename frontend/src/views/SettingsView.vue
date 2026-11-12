<script setup lang="ts">
import {
    BookOpen,
    Bot,
    Monitor,
    Moon,
    Plus,
    Settings,
    Star,
    Sun,
    Trash2,
    DatabaseBackup,
    Globe,
} from "@lucide/vue"
import { ref } from "vue"

import {
    Dialog,
    DialogContent,
    DialogDescription,
    DialogFooter,
    DialogHeader,
    DialogTitle,
} from "@/components/ui/dialog"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { ProviderService, errorMessage, type AIProvider } from "@/lib/api"
import WebSettingsPanel from "@/components/WebSettingsPanel.vue"
import BackupSettings from "@/components/BackupSettings.vue"
import SkillsSettings from "@/components/SkillsSettings.vue"
import { Separator } from "@/components/ui/separator"
import { refreshProviders, store } from "@/lib/store"
import PageHeader from "@/components/PageHeader.vue"
import { Checkbox } from "@/components/ui/checkbox"
import AppSelect from "@/components/AppSelect.vue"
import { Button } from "@/components/ui/button"
import { theme, type Theme } from "@/lib/theme"
import { toast, toastError } from "@/lib/toast"
import { Badge } from "@/components/ui/badge"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { confirmDialog } from "@/lib/confirm"
import { cn } from "@/lib/utils"

const KINDS = [
    {
        value: "openai_compatible",
        label: "Compatível com OpenAI",
        hint: "OpenAI, DeepSeek, Groq, OpenRouter, LM Studio e outros.",
    },
    {
        value: "anthropic",
        label: "Anthropic (Claude)",
        hint: "API nativa da Anthropic.",
    },
    {
        value: "google",
        label: "Google (Gemini)",
        hint: "API do Google AI Studio.",
    },
    { value: "ollama", label: "Ollama", hint: "Modelos locais pelo Ollama." },
]

type Preset = Pick<AIProvider, "name" | "kind" | "baseUrl" | "apiKey" | "model">

const PRESETS: Preset[] = [
    {
        name: "Ollama (local)",
        kind: "ollama",
        baseUrl: "http://localhost:11434",
        apiKey: "",
        model: "llama3.2:latest",
    },
    {
        name: "LM Studio (local)",
        kind: "openai_compatible",
        baseUrl: "http://localhost:1234/v1",
        apiKey: "",
        model: "local-model",
    },
    {
        name: "OpenAI",
        kind: "openai_compatible",
        baseUrl: "https://api.openai.com/v1",
        apiKey: "",
        model: "gpt-4o-mini",
    },
    {
        name: "Anthropic",
        kind: "anthropic",
        baseUrl: "",
        apiKey: "",
        model: "claude-opus-5-5",
    },
    {
        name: "Google Gemini",
        kind: "google",
        baseUrl: "",
        apiKey: "",
        model: "gemini-2.5-flash",
    },
    {
        name: "DeepSeek",
        kind: "openai_compatible",
        baseUrl: "https://api.deepseek.com/v1",
        apiKey: "",
        model: "deepseek-chat",
    },
    {
        name: "Groq",
        kind: "openai_compatible",
        baseUrl: "https://api.groq.com/openai/v1",
        apiKey: "",
        model: "llama-3.3-70b-versatile",
    },
    {
        name: "OpenRouter",
        kind: "openai_compatible",
        baseUrl: "https://openrouter.ai/api/v1",
        apiKey: "",
        model: "openrouter/auto",
    },
]

function blank(): AIProvider {
    return {
        id: 0,
        name: "",
        kind: "openai_compatible",
        baseUrl: "",
        apiKey: "",
        model: "",
        isDefault: false,
        createdAt: "",
        updatedAt: "",
    }
}

const editorOpen = ref(false)
const editing = ref<AIProvider>(blank())
const probeBusy = ref(false)
const probeResult = ref("")
const probeOk = ref(false)

function open(p?: AIProvider | Preset): void {
    editing.value = { ...blank(), ...(p ?? {}) }
    probeResult.value = ""
    editorOpen.value = true
}

const kindLabel = (k: string): string =>
    KINDS.find((x) => x.value === k)?.label ?? k

async function save(): Promise<void> {
    try {
        await ProviderService.SaveProvider(editing.value)
        await refreshProviders()
        toast("Provedor salvo.")
        editorOpen.value = false
    } catch (e) {
        toastError(e)
    }
}

async function runTest(): Promise<void> {
    probeBusy.value = true
    probeResult.value = ""
    try {
        const reply = await ProviderService.TestProvider(editing.value)
        probeOk.value = true
        probeResult.value = `Funcionou. Resposta: “${reply.slice(0, 80)}”`
    } catch (e) {
        probeOk.value = false
        probeResult.value = errorMessage(e)
    } finally {
        probeBusy.value = false
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
    const ok = await confirmDialog({
        title: "Remover provedor",
        description: `"${p.name}" deixará de ser usado pelas funções de IA.`,
        confirmText: "Remover",
        destructive: true,
    })
    if (!ok) return
    try {
        await ProviderService.DeleteProvider(p.id)
        await refreshProviders()
    } catch (e) {
        toastError(e)
    }
}

const THEMES: { value: Theme; label: string; icon: typeof Sun }[] = [
    { value: "light", label: "Claro", icon: Sun },
    { value: "dark", label: "Escuro", icon: Moon },
    { value: "system", label: "Sistema", icon: Monitor },
]
</script>

<template>
    <div class="flex h-full flex-col">
        <PageHeader
            title="Configurações"
            description="Personalize o Social Media Copilot"
            :icon="Settings"
        />

        <div class="flex-1 overflow-y-auto px-8 py-6">
            <div class="mx-auto max-w-2xl">
                <Tabs default-value="ai">
                    <TabsList class="mb-6">
                        <TabsTrigger value="ai"
                            ><Bot class="mr-1 size-4" /> Provedores de
                            IA</TabsTrigger
                        >
                        <TabsTrigger value="web"
                            ><Globe class="mr-1 size-4" /> Web</TabsTrigger
                        >
                        <TabsTrigger value="backup"
                            ><DatabaseBackup class="mr-1 size-4" />
                            Backup</TabsTrigger
                        >
                        <TabsTrigger value="skills"
                            ><BookOpen class="mr-1 size-4" />
                            Skills</TabsTrigger
                        >
                        <TabsTrigger value="general"
                            ><Settings class="mr-1 size-4" /> Geral</TabsTrigger
                        >
                    </TabsList>

                    <TabsContent value="ai" class="space-y-6">
                        <div class="flex items-start justify-between gap-4">
                            <div>
                                <h2 class="font-heading text-lg font-bold">
                                    Seus provedores
                                </h2>
                                <p class="text-sm text-muted-foreground">
                                    Usados para roteiros, metadados e ideias.
                                </p>
                            </div>
                            <Button variant="outline" @click="open()"
                                ><Plus /> Personalizado</Button
                            >
                        </div>

                        <div
                            v-if="!store.providers.length"
                            class="flex flex-col items-center justify-center gap-3 rounded-xl border border-dashed border-border/60 bg-card/40 p-8 py-12 text-center"
                        >
                            <Bot class="size-8 text-muted-foreground" />
                            <p class="max-w-sm text-sm text-muted-foreground">
                                Nenhum provedor ainda. Escolha um modelo pronto
                                abaixo ou adicione um personalizado.
                            </p>
                        </div>
                        <ul v-else class="space-y-3">
                            <li
                                v-for="p in store.providers"
                                :key="p.id"
                                :class="
                                    cn(
                                        'flex items-center gap-3 rounded-xl border bg-card p-4 shadow-xs',
                                        p.isDefault
                                            ? 'border-primary/30'
                                            : 'border-border/60',
                                    )
                                "
                            >
                                <div class="min-w-0 flex-1">
                                    <div class="flex items-center gap-2">
                                        <span class="truncate font-semibold">{{
                                            p.name
                                        }}</span>
                                        <Badge v-if="p.isDefault">Em uso</Badge>
                                    </div>
                                    <p
                                        class="truncate text-sm text-muted-foreground"
                                    >
                                        {{ kindLabel(p.kind) }} · {{ p.model }}
                                    </p>
                                </div>
                                <Button
                                    v-if="!p.isDefault"
                                    variant="ghost"
                                    size="sm"
                                    @click="makeDefault(p)"
                                >
                                    <Star /> Usar este
                                </Button>
                                <Button
                                    variant="outline"
                                    size="sm"
                                    @click="open(p)"
                                    >Editar</Button
                                >
                                <Button
                                    variant="ghost"
                                    size="icon-sm"
                                    class="text-destructive hover:text-destructive"
                                    :aria-label="`Remover ${p.name}`"
                                    @click="remove(p)"
                                >
                                    <Trash2 />
                                </Button>
                            </li>
                        </ul>

                        <Separator />

                        <div class="space-y-3">
                            <h3 class="text-sm font-semibold">
                                Adicionar a partir de um modelo pronto
                            </h3>
                            <div class="grid grid-cols-2 gap-3 sm:grid-cols-4">
                                <button
                                    v-for="preset in PRESETS"
                                    :key="preset.name"
                                    class="flex cursor-pointer flex-col items-start gap-0.5 rounded-xl border-2 border-border/50 bg-card p-3 text-left transition-colors hover:border-primary/30"
                                    @click="open(preset)"
                                >
                                    <span class="text-sm font-semibold">{{
                                        preset.name
                                    }}</span>
                                    <span
                                        class="truncate text-xs text-muted-foreground"
                                        >{{ preset.model }}</span
                                    >
                                </button>
                            </div>
                            <p class="text-xs text-muted-foreground">
                                As chaves de API ficam salvas apenas neste
                                computador, no banco de dados local do app.
                            </p>
                        </div>
                    </TabsContent>

                    <TabsContent value="web">
                        <WebSettingsPanel />
                    </TabsContent>

                    <TabsContent value="backup">
                        <BackupSettings />
                    </TabsContent>

                    <TabsContent value="skills">
                        <SkillsSettings />
                    </TabsContent>

                    <TabsContent value="general" class="space-y-6">
                        <div>
                            <h2 class="font-heading text-lg font-bold">
                                Aparência
                            </h2>
                            <p class="text-sm text-muted-foreground">
                                Escolha o tema da interface
                            </p>
                        </div>
                        <div class="grid grid-cols-3 gap-3">
                            <button
                                v-for="t in THEMES"
                                :key="t.value"
                                :class="
                                    cn(
                                        'flex cursor-pointer flex-col items-center gap-2 rounded-xl border-2 p-4 transition-colors',
                                        theme === t.value
                                            ? 'border-primary bg-primary/10'
                                            : 'border-border/50 bg-card hover:border-primary/30',
                                    )
                                "
                                @click="theme = t.value"
                            >
                                <component :is="t.icon" class="size-5" />
                                <span class="text-sm font-medium">{{
                                    t.label
                                }}</span>
                            </button>
                        </div>
                        <Separator />
                        <div class="space-y-2">
                            <h3 class="text-sm font-semibold">Sobre</h3>
                            <div
                                class="space-y-1 rounded-lg bg-muted p-4 font-mono text-xs"
                            >
                                <p>Social Media Copilot v0.1.2</p>
                                <p class="text-muted-foreground">
                                    Wails v3 + Go + Vue 3 + TypeScript + SQLite
                                </p>
                            </div>
                        </div>
                    </TabsContent>
                </Tabs>
            </div>
        </div>

        <Dialog v-model:open="editorOpen">
            <DialogContent class="sm:max-w-[520px]">
                <DialogHeader>
                    <DialogTitle>{{
                        editing.id ? "Editar provedor" : "Novo provedor"
                    }}</DialogTitle>
                    <DialogDescription>{{
                        KINDS.find((k) => k.value === editing.kind)?.hint
                    }}</DialogDescription>
                </DialogHeader>
                <form class="space-y-4" @submit.prevent="save">
                    <div class="space-y-2">
                        <Label for="provider-name">Nome</Label>
                        <Input id="provider-name" v-model="editing.name" />
                    </div>
                    <div class="space-y-2">
                        <Label>Tipo</Label>
                        <AppSelect
                            v-model="editing.kind"
                            :options="KINDS"
                            label="Tipo"
                        />
                    </div>
                    <div class="space-y-2">
                        <Label for="provider-url">
                            URL base
                            {{
                                editing.kind === "openai_compatible"
                                    ? ""
                                    : "(opcional)"
                            }}
                        </Label>
                        <Input
                            id="provider-url"
                            v-model="editing.baseUrl"
                            placeholder="https://…"
                        />
                    </div>
                    <div class="space-y-2">
                        <Label for="provider-key">
                            Chave de API
                            {{
                                ["anthropic", "google"].includes(editing.kind)
                                    ? ""
                                    : "(opcional para locais)"
                            }}
                        </Label>
                        <Input
                            id="provider-key"
                            v-model="editing.apiKey"
                            type="password"
                            autocomplete="off"
                        />
                    </div>
                    <div class="space-y-2">
                        <Label for="provider-model">Modelo</Label>
                        <Input id="provider-model" v-model="editing.model" />
                    </div>
                    <Label class="flex items-center gap-2 font-normal">
                        <Checkbox v-model="editing.isDefault" /> Usar este
                        provedor nas funções de IA
                    </Label>
                    <p
                        v-if="probeResult"
                        :class="
                            cn(
                                'rounded-lg p-3 text-sm break-words',
                                probeOk
                                    ? 'bg-confirm/10 text-confirm'
                                    : 'bg-destructive/10 text-destructive',
                            )
                        "
                    >
                        {{ probeResult }}
                    </p>
                </form>
                <DialogFooter class="sm:justify-between">
                    <Button
                        variant="ghost"
                        :disabled="probeBusy"
                        @click="runTest"
                    >
                        {{ probeBusy ? "Testando…" : "Testar conexão" }}
                    </Button>
                    <div class="flex gap-2">
                        <Button variant="outline" @click="editorOpen = false"
                            >Cancelar</Button
                        >
                        <Button @click="save">Salvar provedor</Button>
                    </div>
                </DialogFooter>
            </DialogContent>
        </Dialog>
    </div>
</template>
