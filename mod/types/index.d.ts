/** One chat, from `roundhouse list --json`. */
export type Chat = {
  id: string
  title: string
  last_prompt: string
  /** RFC 3339 time of the last change of the transcript. */
  modified: string
}

/** What the mod knows about the chats of the session's folder. */
export type Snapshot =
  | { kind: 'ok'; cwd: string; sessionId: string; chats: Chat[] }
  /** The command is missing from PATH or failed. */
  | { kind: 'unavailable'; reason: string }
  /** The command reports a schema this mod does not read. */
  | { kind: 'incompatible'; schema: number | null }

/** The color the style page edits. */
export type ColorSlot = 'background' | 'text' | 'accent' | 'border'

/** The pane colors; the empty string means the default of the slot. */
export type Colors = Record<ColorSlot, string>

/** The pages of the /roundhouse pane. */
export type Page = 'chats' | 'style'

declare module 'claude-code' {
  interface PluginState {
    'roundhouse-mod': { snapshot: Snapshot | null; page: Page; slot: ColorSlot; colors: Colors }
  }
}
