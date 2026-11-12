<script setup lang="ts">
import { Browser } from "@wailsio/runtime"
import { onMounted, ref } from "vue"
import { Globe } from "@lucide/vue"

import { WebService, errorMessage, type WebSettings } from "@/lib/api"
import { Separator } from "@/components/ui/separator"
import { refreshWebStatus, store } from "@/lib/store"
import { Button } from "@/components/ui/button"
import { Switch } from "@/components/ui/switch"
import { toast, toastError } from "@/lib/toast"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { cn } from "@/lib/utils"

const form = ref<WebSettings>({
    firecrawlApiKey: "",
    firecrawlBaseUrl: "",
    nativeSearch: true,
})
const saving = ref(false)
const testing = ref(false)
const testResult = ref("")
const testOk = ref(false)

onMounted(async () => {
    try {
        form.value = await WebService.GetWebSettings()
        await refreshWebStatus()
    } catch (e) {
        toastError(e)
    }
})

async function save(): Promise<void> {
    saving.value = true
    try {
        await WebService.SaveWebSettings(form.value)
        await refreshWebStatus()
        toast("Configurações de web salvas.")
    } catch (e) {
        toastError(e)
    } finally {
        saving.value = false
    }
}

async function test(): Promise<void> {
    testing.value = true
    testResult.value = ""
    try {
        testResult.value = await WebService.TestFirecrawl(form.value)
        testOk.value = true
    } catch (e) {
        testResult.value = errorMessage(e)
        testOk.value = false
    } finally {
        testing.value = false
    }
}
</script>

<template>
    <div class="space-y-6">
        <div>
            <h2 class="font-heading text-lg font-bold">Pesquisa na web</h2>
            <p class="text-sm text-muted-foreground">
                Permite ao Copilot pesquisar fatos atuais e ler links, no chat e
                antes de escrever roteiros e ideias.
            </p>
        </div>

        <div
            class="grid gap-3 rounded-xl border border-border/60 bg-card p-5 shadow-xs sm:grid-cols-2"
        >
            <div>
                <p class="text-xs font-medium text-muted-foreground">
                    Pesquisa
                </p>
                <p class="font-semibold">
                    {{ store.web.searchVia || "Indisponível" }}
                </p>
            </div>
            <div>
                <p class="text-xs font-medium text-muted-foreground">
                    Leitura de links
                </p>
                <p class="font-semibold">
                    {{ store.web.fetchVia || "Indisponível" }}
                </p>
            </div>
        </div>

        <form class="space-y-5" @submit.prevent="save">
            <div
                class="flex items-start justify-between gap-4 rounded-xl border border-border/60 bg-card p-5 shadow-xs"
            >
                <div class="space-y-1">
                    <Label for="native-search">Usar a busca do provedor</Label>
                    <p class="text-sm text-muted-foreground">
                        Anthropic, Google e OpenAI têm pesquisa própria, usada
                        quando o provedor em uso for um deles. Nos outros, ou se
                        ela falhar, o Firecrawl é usado.
                    </p>
                </div>
                <Switch id="native-search" v-model="form.nativeSearch" />
            </div>

            <div
                class="space-y-4 rounded-xl border border-border/60 bg-card p-5 shadow-xs"
            >
                <div class="flex items-center gap-2">
                    <Globe class="size-4" />
                    <h3 class="font-semibold">Firecrawl</h3>
                </div>
                <p class="text-sm text-muted-foreground">
                    Pesquisa e leitura de páginas para qualquer provedor,
                    inclusive modelos locais. Usa créditos da sua conta
                    <button
                        type="button"
                        class="cursor-pointer font-medium text-primary underline-offset-4 hover:underline"
                        @click="Browser.OpenURL('https://www.firecrawl.dev')"
                    >
                        firecrawl.dev</button
                    >; também funciona com uma instância própria.
                </p>
                <div class="space-y-2">
                    <Label for="firecrawl-key">Chave de API</Label>
                    <Input
                        id="firecrawl-key"
                        v-model="form.firecrawlApiKey"
                        type="password"
                        autocomplete="off"
                        placeholder="fc-..."
                    />
                </div>
                <div class="space-y-2">
                    <Label for="firecrawl-url">URL base (opcional)</Label>
                    <Input
                        id="firecrawl-url"
                        v-model="form.firecrawlBaseUrl"
                        placeholder="https://api.firecrawl.dev ou http://localhost:3002"
                    />
                    <p class="text-xs text-muted-foreground">
                        Só para instância própria, que pode funcionar sem chave.
                    </p>
                </div>
                <p
                    v-if="testResult"
                    :class="
                        cn(
                            'rounded-lg p-3 text-sm break-words',
                            testOk
                                ? 'bg-confirm/10 text-confirm'
                                : 'bg-destructive/10 text-destructive',
                        )
                    "
                >
                    {{ testResult }}
                </p>
                <Separator />
                <div class="flex items-center justify-between gap-2">
                    <Button
                        type="button"
                        variant="ghost"
                        :disabled="testing"
                        @click="test"
                    >
                        {{ testing ? "Testando…" : "Testar Firecrawl" }}
                    </Button>
                    <Button type="submit" :disabled="saving">
                        Salvar configurações de web
                    </Button>
                </div>
            </div>
        </form>
    </div>
</template>
