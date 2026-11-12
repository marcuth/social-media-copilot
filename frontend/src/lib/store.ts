import { reactive } from "vue"

import {
    CatalogService,
    ProfileService,
    ProviderService,
    normProfile,
    type AIProvider,
    type Option,
    type Platform,
    type Profile,
    type ProfilePlatform,
    type Theme,
} from "./api"

type ProfileN = Profile & { platforms: ProfilePlatform[] }

interface CatalogN {
    platforms: Platform[]
    formats: Option[]
    statuses: Option[]
}

const ACTIVE_KEY = "smc:active-profile"

export const store = reactive({
    ready: false,
    catalog: { platforms: [], formats: [], statuses: [] } as CatalogN,
    profiles: [] as ProfileN[],
    themes: [] as Theme[],
    providers: [] as AIProvider[],
    /** Global profile filter; null shows every profile. */
    activeProfileId: (Number(localStorage.getItem(ACTIVE_KEY)) || null) as number | null,
})

export async function loadAll(): Promise<void> {
    const [catalog] = await Promise.all([CatalogService.GetCatalog(), refreshProfiles(), refreshProviders()])
    store.catalog = {
        platforms: catalog.platforms ?? [],
        formats: catalog.formats ?? [],
        statuses: catalog.statuses ?? [],
    }
    if (store.activeProfileId && !store.profiles.some((p) => p.id === store.activeProfileId)) {
        setActiveProfile(null)
    }
    store.ready = true
}

export async function refreshProfiles(): Promise<void> {
    const [profiles, themes] = await Promise.all([ProfileService.ListProfiles(), ProfileService.ListThemes(0)])
    store.profiles = (profiles ?? []).map(normProfile)
    store.themes = themes ?? []
}

export async function refreshProviders(): Promise<void> {
    store.providers = (await ProviderService.ListProviders()) ?? []
}

export function setActiveProfile(id: number | null): void {
    store.activeProfileId = id
    if (id) localStorage.setItem(ACTIVE_KEY, String(id))
    else localStorage.removeItem(ACTIVE_KEY)
}

export function platformInfo(id: string): Platform {
    return store.catalog.platforms.find((p) => p.id === id) ?? { id, name: id, color: "#807e96", guidance: "" }
}

export const profileById = (id: number | null | undefined): ProfileN | undefined =>
    store.profiles.find((p) => p.id === id)

export const themeById = (id: number | null | undefined): Theme | undefined => store.themes.find((t) => t.id === id)

export const themesOf = (profileId: number | null | undefined): Theme[] =>
    store.themes.filter((t) => t.profileId === profileId)

export const formatLabel = (id: string): string => store.catalog.formats.find((f) => f.id === id)?.label ?? id

export const statusLabel = (id: string): string => store.catalog.statuses.find((s) => s.id === id)?.label ?? id

/** Colour per pipeline stage, from cold (idea) to done (published). */
export const STATUS_COLORS: Record<string, string> = {
    idea: "#9b98b3",
    scripting: "#7c6ce0",
    production: "#d9822b",
    editing: "#c2569b",
    ready: "#3f51d9",
    published: "#1d8f63",
}

export const PROFILE_COLORS = ["#3F51D9", "#E8336D", "#1D8F63", "#D9822B", "#7C6CE0", "#0F8FB3", "#B8404F", "#5B6B2E"]
