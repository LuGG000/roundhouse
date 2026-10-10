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

declare module 'claude-code' {
  interface PluginState {
    'roundhouse-mod': { snapshot: Snapshot | null }
  }
}
