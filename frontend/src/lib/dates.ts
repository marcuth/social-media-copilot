// Schedule values are local wall-clock strings "YYYY-MM-DDTHH:MM".

const pad = (n: number): string => String(n).padStart(2, "0")

export function dayKey(d: Date): string {
    return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`
}

export function toSchedule(d: Date): string {
    return `${dayKey(d)}T${pad(d.getHours())}:${pad(d.getMinutes())}`
}

export function parseSchedule(s: string | null | undefined): Date | null {
    if (!s) return null
    const [date, time = "09:00"] = s.split("T")
    const [y, m, d] = date.split("-").map(Number)
    const [hh, mm] = time.split(":").map(Number)
    if (!y || !m || !d) return null
    return new Date(y, m - 1, d, hh || 0, mm || 0)
}

export function addDays(d: Date, n: number): Date {
    const out = new Date(d)
    out.setDate(out.getDate() + n)
    return out
}

export function startOfWeek(d: Date): Date {
    const out = new Date(d.getFullYear(), d.getMonth(), d.getDate())
    const weekday = (out.getDay() + 6) % 7 // Monday first
    return addDays(out, -weekday)
}

/** Six Monday-first weeks covering the month of `month`. */
export function monthGrid(month: Date): Date[] {
    const first = new Date(month.getFullYear(), month.getMonth(), 1)
    const start = startOfWeek(first)
    return Array.from({ length: 42 }, (_, i) => addDays(start, i))
}

const fmtTime = new Intl.DateTimeFormat("pt-BR", { hour: "2-digit", minute: "2-digit" })
const fmtDay = new Intl.DateTimeFormat("pt-BR", { weekday: "short", day: "numeric", month: "short" })
const fmtLong = new Intl.DateTimeFormat("pt-BR", { weekday: "long", day: "numeric", month: "long" })
const fmtMonth = new Intl.DateTimeFormat("pt-BR", { month: "long", year: "numeric" })

export const formatTime = (d: Date): string => fmtTime.format(d)
export const formatDay = (d: Date): string => fmtDay.format(d).replace(/\./g, "")
export const formatLongDay = (d: Date): string => fmtLong.format(d)
export const formatMonth = (d: Date): string => fmtMonth.format(d)

export function formatSchedule(s: string | null | undefined): string {
    const d = parseSchedule(s)
    if (!d) return "Sem data"
    return `${formatDay(d)}, ${formatTime(d)}`
}

export function isOverdue(s: string | null | undefined, status: string): boolean {
    const d = parseSchedule(s)
    return !!d && status !== "published" && d.getTime() < Date.now()
}

export const WEEKDAYS = ["seg", "ter", "qua", "qui", "sex", "sáb", "dom"]
