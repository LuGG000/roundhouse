import type { Chat, UpdateInfo } from '../types'

/** The `list --json` schema this mod reads. */
export const LIST_SCHEMA = 1

export type Parsed =
  | { kind: 'ok'; chats: Chat[]; version: string | null; update: UpdateInfo | null }
  | { kind: 'incompatible'; schema: number | null }

const isText = (value: unknown): value is string => typeof value === 'string'

/** Reads the output of `roundhouse list --json`; chats without an id or a title are left out. */
export function parseList(stdout: string): Parsed {
  let data: unknown
  try {
    data = JSON.parse(stdout)
  } catch {
    return { kind: 'incompatible', schema: null }
  }
  const list = data as { schema?: unknown; sessions?: unknown; version?: unknown; update?: unknown }
  if (list.schema !== LIST_SCHEMA) {
    return { kind: 'incompatible', schema: typeof list.schema === 'number' ? list.schema : null }
  }
  if (!Array.isArray(list.sessions)) {
    return { kind: 'incompatible', schema: LIST_SCHEMA }
  }

  const chats: Chat[] = []
  for (const item of list.sessions as Record<string, unknown>[]) {
    if (isText(item.id) && isText(item.title) && item.title !== '') {
      chats.push({
        id: item.id,
        title: item.title,
        last_prompt: isText(item.last_prompt) ? item.last_prompt : '',
        modified: isText(item.modified) ? item.modified : '',
      })
    }
  }

  return { kind: 'ok', chats, version: isText(list.version) ? list.version : null, update: parseUpdate(list.update) }
}

/** A coarse relative time like `3h ago`; the empty string when `modified` is not a time. */
export function ago(modified: string, now: number): string {
  const then = Date.parse(modified)
  if (Number.isNaN(then)) {
    return ''
  }
  const seconds = Math.max(0, Math.round((now - then) / 1000))
  if (seconds < 60) return 'just now'
  if (seconds < 3600) return `${Math.floor(seconds / 60)}m ago`
  if (seconds < 86_400) return `${Math.floor(seconds / 3600)}h ago`
  if (seconds < 30 * 86_400) return `${Math.floor(seconds / 86_400)}d ago`

  return modified.slice(0, 10)
}

/** Collapses line breaks and runs of spaces, so a title fits one line. */
export function oneLine(text: string): string {
  return text.split(/\s+/).filter(Boolean).join(' ')
}

/** The `update` field of `list --json`: a newer release, or null for none or anything unreadable. */
function parseUpdate(value: unknown): UpdateInfo | null {
  const update = value as { version?: unknown; url?: unknown } | null
  if (update && isText(update.version) && update.version !== '') {
    return { version: update.version, url: isText(update.url) ? update.url : '' }
  }

  return null
}
