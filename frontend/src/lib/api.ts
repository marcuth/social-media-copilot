// Single entry point to the generated Wails bindings, plus small helpers that
// normalise Go nil slices (sent as null) into empty arrays.
import type * as Models from "../../bindings/github.com/marcuth/social-media-copilot/internal/models/models"
import type * as Catalog from "../../bindings/github.com/marcuth/social-media-copilot/internal/catalog/models"
import type * as Svc from "../../bindings/github.com/marcuth/social-media-copilot/internal/services/models"

export {
    AIService,
    CatalogService,
    ChatService,
    PostService,
    ProfileService,
    ProviderService,
    SkillService,
} from "../../bindings/github.com/marcuth/social-media-copilot/internal/services"
export { AiDelta } from "../../bindings/events"

export type Profile = Models.Profile
export type ProfilePlatform = Models.ProfilePlatform
export type Theme = Models.Theme
export type Post = Models.Post
export type PostPlatform = Models.PostPlatform
export type PostFilter = Models.PostFilter
export type PostponeRequest = Models.PostponeRequest
export type Postponement = Models.Postponement
export type AIProvider = Models.AIProvider
export type Stats = Models.Stats
export type Platform = Catalog.Platform
export type Option = Catalog.Option
export type CatalogData = Svc.Catalog
export type AIRequest = Svc.AIRequest
export type Metadata = Svc.Metadata
export type Idea = Svc.Idea
export type AIDelta = Svc.AIDelta
export type ChatSession = Svc.ChatSession
export type ChatMessage = Svc.ChatMessage
export type SkillInfo = Svc.SkillInfo

/** Post with arrays guaranteed non-null. */
export interface PostN extends Post {
    tags: string[]
    hashtags: string[]
    platforms: PostPlatform[]
}

export function normPost(p: Post): PostN {
    return {
        ...p,
        tags: p.tags ?? [],
        hashtags: p.hashtags ?? [],
        platforms: p.platforms ?? [],
    }
}

export function normProfile(p: Profile): Profile & { platforms: ProfilePlatform[] } {
    return { ...p, platforms: p.platforms ?? [] }
}

export function emptyFilter(patch: Partial<PostFilter> = {}): PostFilter {
    return {
        profileId: null,
        themeId: null,
        status: "",
        platform: "",
        from: "",
        to: "",
        search: "",
        unscheduled: false,
        ...patch,
    }
}

/** Human message from a rejected binding call. */
export function errorMessage(e: unknown): string {
    if (e instanceof Error) return e.message
    if (typeof e === "object" && e && "message" in e) return String((e as { message: unknown }).message)
    return String(e)
}

/** True when a cancellable call was cancelled by us. */
export function isCancel(e: unknown): boolean {
    const name = typeof e === "object" && e && "name" in e ? String((e as { name: unknown }).name) : ""
    return name === "CancelError" || /cancel/i.test(errorMessage(e))
}
