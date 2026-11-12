<script setup lang="ts">
import { computed, ref, watch } from "vue"
import { useRouter } from "vue-router"
import { BookOpen, Check, Lightbulb, Plus, Wand2 } from "@lucide/vue"

import AppSelect from "@/components/AppSelect.vue"
import PageHeader from "@/components/PageHeader.vue"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { AIService, PostService, isCancel, type Idea, type SkillInfo } from "@/lib/api"
import { formatLabel, profileById, store, themesOf } from "@/lib/store"
import { toast, toastError } from "@/lib/toast"

const router = useRouter()

const profileId = ref<number | null>(store.activeProfileId ?? store.profiles[0]?.id ?? null)
const themeId = ref<number | null>(null)
const format = ref("")
const count = ref(6)
const instructions = ref("")
const ideas = ref<(Idea & { added?: number })[]>([])
const busy = ref(false)

const profileOptions = computed(() => store.profiles.map((p) => ({ value: p.id as number | null, label: p.name })))
const themeOptions = computed(() => [
    { value: null, label: "Qualquer tema" },
    ...themesOf(profileId.value).map((t) => ({ value: t.id, label: t.name })),
])
const formatOptions = computed(() => [
    { value: "", label: "Variado" },
    ...store.catalog.formats.map((f) => ({ value: f.id, label: f.label })),
])
watch(profileId, () => (themeId.value = null))

const skillsUsed = ref<SkillInfo[]>([])
watch(
    [profileId, format],
    async () => {
        if (!profileId.value) return
        skillsUsed.value = (await AIService.MatchingSkills("ideas", profileId.value, [], format.value).catch(() => [])) ?? []
    },
    { immediate: true },
)

async function generate(): Promise<void> {
    if (!profileId.value) return
    if (!store.providers.length) {
        toast("Configure um provedor de IA em Configurações para gerar ideias.", "error")
        return
    }
    busy.value = true
    try {
        ideas.value =
            (await AIService.GenerateIdeas({
                profileId: profileId.value,
                themeId: themeId.value,
                count: count.value,
                format: format.value,
                platforms: [],
                instructions: instructions.value,
            })) ?? []
        if (!ideas.value.length) toast("A IA não trouxe ideias desta vez. Tente de novo.", "error")
    } catch (e) {
        if (!isCancel(e)) toastError(e)
    } finally {
        busy.value = false
    }
}

async function add(idea: Idea & { added?: number }, openAfter: boolean): Promise<void> {
    const profile = profileById(profileId.value)
    if (!profile) return
    try {
        const post = await PostService.SavePost({
            id: 0,
            profileId: profile.id,
            themeId: themeId.value,
            title: idea.title,
            description: "",
            script: "",
            notes: [idea.hook && `Gancho: ${idea.hook}`, idea.summary].filter(Boolean).join("\n\n"),
            tags: [],
            hashtags: [],
            format: idea.format,
            status: "idea",
            scheduledAt: null,
            publishedAt: null,
            postponeCount: 0,
            platforms: profile.platforms.map((p) => ({ platform: p.platform, status: "pending", url: "", publishedAt: null })),
            createdAt: "",
            updatedAt: "",
        })
        idea.added = post.id
        if (openAfter) router.push({ name: "post", params: { id: post.id } })
        else toast("Ideia adicionada à produção.")
    } catch (e) {
        toastError(e)
    }
}
</script>

<template>
    <div class="flex h-full flex-col">
        <PageHeader
            title="Ideias com IA"
            description="Peça ideias para um perfil e mande as boas direto para a produção"
            :icon="Lightbulb"
        />

        <div class="flex-1 overflow-y-auto px-8 py-6">
            <div class="mx-auto max-w-4xl space-y-6">
                <div
                    v-if="!store.profiles.length"
                    class="flex flex-col items-center justify-center gap-3 rounded-xl border border-dashed border-border/60 bg-card/40 p-8 py-16 text-center"
                >
                    <Lightbulb class="size-8 text-muted-foreground" />
                    <h2 class="font-heading text-lg font-bold">Crie um perfil primeiro</h2>
                    <p class="max-w-md text-sm text-muted-foreground">
                        As ideias partem do nicho, do público e do tom de voz do perfil.
                    </p>
                    <Button as-child><RouterLink to="/profiles?new=1">Criar perfil</RouterLink></Button>
                </div>

                <template v-else>
                    <form class="space-y-4 rounded-xl border border-border/60 bg-card p-6 shadow-xs" @submit.prevent="generate">
                        <div class="grid gap-4 md:grid-cols-[1fr_1fr_1fr_110px]">
                            <div class="space-y-2">
                                <Label>Perfil</Label>
                                <AppSelect v-model="profileId" :options="profileOptions" label="Perfil" />
                            </div>
                            <div class="space-y-2">
                                <Label>Tema</Label>
                                <AppSelect v-model="themeId" :options="themeOptions" label="Tema" />
                            </div>
                            <div class="space-y-2">
                                <Label>Formato</Label>
                                <AppSelect v-model="format" :options="formatOptions" label="Formato" />
                            </div>
                            <div class="space-y-2">
                                <Label for="count">Quantidade</Label>
                                <Input id="count" v-model.number="count" type="number" min="1" max="20" />
                            </div>
                        </div>
                        <div v-if="skillsUsed.length" class="flex flex-wrap items-center gap-1.5 text-xs text-muted-foreground">
                            <BookOpen class="size-3.5" /> Seguindo:
                            <Badge v-for="s in skillsUsed" :key="s.id" variant="secondary" :title="s.description">{{ s.name }}</Badge>
                        </div>
                        <div class="flex items-end gap-3">
                            <div class="flex-1 space-y-2">
                                <Label for="direction">Direção (opcional)</Label>
                                <Input
                                    id="direction"
                                    v-model="instructions"
                                    placeholder="Ex.: ideias para a Black Friday, com bastidores"
                                />
                            </div>
                            <Button type="submit" :disabled="busy || !profileId">
                                <Wand2 /> {{ busy ? "Pensando…" : "Gerar ideias" }}
                            </Button>
                        </div>
                    </form>

                    <ol v-if="ideas.length" class="space-y-3">
                        <li
                            v-for="(idea, i) in ideas"
                            :key="i"
                            class="flex justify-between gap-6 rounded-xl border border-border/60 bg-card p-5 shadow-xs"
                        >
                            <div class="max-w-2xl space-y-1.5">
                                <h2 class="font-heading text-base font-bold">{{ idea.title }}</h2>
                                <p v-if="idea.hook" class="text-sm text-muted-foreground italic">“{{ idea.hook }}”</p>
                                <p class="text-sm">{{ idea.summary }}</p>
                                <Badge variant="secondary">{{ formatLabel(idea.format) }}</Badge>
                            </div>
                            <div class="flex shrink-0 flex-col items-stretch gap-2">
                                <template v-if="idea.added">
                                    <span class="flex items-center gap-1.5 text-sm font-semibold text-confirm">
                                        <Check class="size-4" /> Na produção
                                    </span>
                                    <Button variant="outline" size="sm" as-child>
                                        <RouterLink :to="{ name: 'post', params: { id: idea.added } }">Abrir</RouterLink>
                                    </Button>
                                </template>
                                <template v-else>
                                    <Button size="sm" @click="add(idea, false)"><Plus /> Adicionar</Button>
                                    <Button size="sm" variant="ghost" @click="add(idea, true)">Adicionar e abrir</Button>
                                </template>
                            </div>
                        </li>
                    </ol>
                    <div
                        v-else-if="!busy"
                        class="flex flex-col items-center justify-center gap-3 rounded-xl border border-dashed border-border/60 bg-card/40 p-8 py-16 text-center"
                    >
                        <Lightbulb class="size-8 text-muted-foreground" />
                        <p class="max-w-md text-sm text-muted-foreground">
                            As ideias aparecem aqui. Elas entram na produção como “Ideia”, com o gancho nas anotações.
                        </p>
                    </div>
                </template>
            </div>
        </div>
    </div>
</template>
