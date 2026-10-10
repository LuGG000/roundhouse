import type { ColorSlot, Colors } from '../types'

/** The colors the style page edits, in the order it lists them. */
export const COLOR_SLOTS: readonly { id: ColorSlot; label: string }[] = [
  { id: 'background', label: 'background' },
  { id: 'text', label: 'text' },
  { id: 'accent', label: 'accent' },
  { id: 'border', label: 'borders' },
]

/** The color names offered in the pickers; any other color is typed in. */
export const NAMED_COLORS = ['white', 'black', 'gray', 'red', 'green', 'yellow', 'blue', 'magenta', 'cyan'] as const

export const NO_COLORS: Colors = { background: '', text: '', accent: '', border: '' }

/**
 * What an empty setting draws. Theme keys follow the person's Claude theme. No
 * entry means the engine's own background or the theme's text color.
 */
export const DEFAULT_COLOR: Record<ColorSlot, string | undefined> = {
  background: undefined,
  text: undefined,
  accent: 'claude',
  border: 'subtle',
}

export const PALETTES = ['default', 'dark', 'light'] as const
export type PaletteName = (typeof PALETTES)[number]

const PALETTE_COLORS: Record<PaletteName, Colors> = {
  default: NO_COLORS,
  dark: { background: '#1e1e1e', text: '#d4d4d4', accent: '#d97757', border: '#555555' },
  light: { background: '#f5f5f5', text: '#222222', accent: '#b5502d', border: '#bbbbbb' },
}

/** The colors a named palette sets, all four slots at once. */
export function paletteColors(name: PaletteName): Colors {
  return { ...PALETTE_COLORS[name] }
}

/** What the pane draws with: every slot resolved, undefined where the engine's own applies. */
export type Palette = Record<ColorSlot, string | undefined>

const unset = (value: string | undefined): string | undefined => (value === undefined || value.trim() === '' ? undefined : value.trim())

export function palette(colors: Colors): Palette {
  return {
    background: unset(colors.background),
    text: unset(colors.text),
    accent: unset(colors.accent) ?? DEFAULT_COLOR.accent,
    border: unset(colors.border) ?? DEFAULT_COLOR.border,
  }
}

/** What a slot shows on the style page: the set value, or what the default draws. */
export function describeColor(value: string, slot: ColorSlot): string {
  const set = unset(value)
  if (set) {
    return set
  }

  return `default (${DEFAULT_COLOR[slot] ?? (slot === 'background' ? 'Claude background' : 'Claude text color')})`
}

/** Reads the colors kept in the store; anything that is not a text per slot is left empty. */
export function parseColors(value: unknown): Colors {
  const stored = (value && typeof value === 'object' ? value : {}) as Record<string, unknown>
  const text = (slot: ColorSlot) => (typeof stored[slot] === 'string' ? (stored[slot] as string).trim() : '')

  return { background: text('background'), text: text('text'), accent: text('accent'), border: text('border') }
}

/** Words that clear a color, so the default applies. */
const CLEARS = ['default', 'theme']

/**
 * Reads what was typed into the color field. A single color (`#00ff88`, `red`)
 * is for the chosen slot, `<slot> <color>` names its slot; a bare six-digit hex
 * gets its `#`. Null when it is neither.
 */
export function parseColorEntry(text: string, chosen: ColorSlot): { slot: ColorSlot; value: string } | null {
  const words = text.trim().split(/\s+/).filter(Boolean)
  const named = words.length === 2 ? COLOR_SLOTS.find(s => s.id === words[0]?.toLowerCase()) : undefined
  const color = words.length === 2 ? (named ? words[1] : undefined) : words.length === 1 ? words[0] : undefined
  if (!color) {
    return null
  }
  const value = CLEARS.includes(color.toLowerCase()) ? '' : /^[0-9a-f]{6}$/i.test(color) ? `#${color}` : color

  return { slot: named?.id ?? chosen, value }
}
