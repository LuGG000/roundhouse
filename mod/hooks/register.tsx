import { atom, read, update } from 'claude-code'
import type { EngineInterface, Register } from 'claude-code'

import type { Chat, Colors, Page, Snapshot } from '../types'
import { ago, oneLine, parseList } from './chats'
import {
  COLOR_SLOTS,
  DEFAULT_COLOR,
  NAMED_COLORS,
  NO_COLORS,
  PALETTES,
  describeColor,
  palette,
  paletteColors,
  parseColorEntry,
  parseColors,
} from './palette'
import type { PaletteName } from './palette'

const PANE = 'roundhouse'
const COLORS_KEY = 'colors'

const snapshot = atom({ plugin: 'roundhouse-mod', key: 'snapshot' } as const, null)
const page = atom({ plugin: 'roundhouse-mod', key: 'page' } as const, 'chats')
const slotAtom = atom({ plugin: 'roundhouse-mod', key: 'slot' } as const, 'background')
const colorsAtom = atom({ plugin: 'roundhouse-mod', key: 'colors' } as const, NO_COLORS)

type Engine = EngineInterface

/** Lines a chat takes in the pane: title, buttons and a gap. */
const CHAT_ROWS = 3
/** Lines of the pane around the chats: header, buttons and the note about more chats. */
const CHROME_ROWS = 5

/** Runs `roundhouse` with `argv`; the text of a refusal when it fails. */
async function run($: Engine, argv: string[], timeoutMs = 10_000): Promise<{ stdout: string } | { error: string }> {
  try {
    const result = await $.process.run(['roundhouse', ...argv], { timeoutMs })

    return result.exitCode === 0
      ? { stdout: result.stdout }
      : { error: result.stderr.trim() || `roundhouse ${argv[0]} failed (exit code ${result.exitCode})` }
  } catch {
    return { error: 'roundhouse was not found in PATH' }
  }
}

/** Reads the chats of the session's folder from the CLI; a missing command is a snapshot, not an error. */
async function load($: Engine): Promise<Snapshot> {
  const cwd = await $.session.cwd()
  const result = await run($, ['list', '--json', '--folder', cwd])
  if ('error' in result) {
    return { kind: 'unavailable', reason: result.error }
  }
  const parsed = parseList(result.stdout)

  return parsed.kind === 'ok' ? { kind: 'ok', cwd, sessionId: await $.session.id(), chats: parsed.chats } : parsed
}

async function refresh($: Engine): Promise<void> {
  const current = await load($)
  await update($, snapshot, () => current)
}

/** Reads the chats again when the pane is open, so it follows a switch or a new chat without a press. */
async function refreshIfOpen($: Engine): Promise<void> {
  if ((await $.ui.panes()).some(pane => pane.id === PANE)) {
    await refresh($)
  }
}

async function loadColors($: Engine): Promise<void> {
  const colors = parseColors(await $.store.get(COLORS_KEY))
  await update($, colorsAtom, () => colors)
}

async function saveColors($: Engine, colors: Colors): Promise<void> {
  await $.store.set(COLORS_KEY, colors)
  await update($, colorsAtom, () => colors)
}

/** Continues the conversation in this session with the chosen chat, as if `/resume <id>` was typed. */
async function switchTo($: Engine, chat: Chat): Promise<void> {
  try {
    await $.command.run({ command: 'resume', args: chat.id })
  } catch (error) {
    $.ui.toast(`Could not resume: ${error instanceof Error ? error.message : String(error)}`)
  }
  await refresh($)
}

/** Opens the chat in a new terminal window, in its folder, through `roundhouse open --terminal`. */
async function openWindow($: Engine, chat: Chat): Promise<void> {
  const result = await run($, ['open', chat.id, '--terminal'])
  $.ui.toast('error' in result ? result.error : `Opened "${oneLine(chat.title)}" in a new terminal.`)
}

async function startNewChat($: Engine): Promise<void> {
  try {
    await $.command.run({ command: 'clear' })
  } catch (error) {
    $.ui.toast(`Could not start a new chat: ${error instanceof Error ? error.message : String(error)}`)
  }
  await refresh($)
}

async function open($: Engine, to: Page): Promise<void> {
  await update($, page, () => to)
  await refresh($)
  await $.ui.open({ id: PANE, title: 'roundhouse' })
}

export const register: Register = on => {
  on('session.start', async ($, e, next) => {
    await $.command.register({
      name: 'roundhouse',
      description: 'Chats of this folder: switch to one or open it in a new terminal',
      argumentHint: '[style]',
    })
    await loadColors($)
    const started = await next(e)
    void refreshIfOpen($)

    return started
  })

  on('turn.complete', async ($, e, next) => {
    void refreshIfOpen($)

    return next(e)
  })

  on('command.run', { command: 'roundhouse' }, async ($, e) => {
    await loadColors($)
    const to: Page = e.args.trim() === 'style' ? 'style' : 'chats'
    await open($, to)

    return { text: to === 'style' ? 'Style page opened.' : 'Roundhouse pane opened.' }
  })

  on('ui.render', { component: 'Pane', requestId: PANE }, async ($, e) => {
    const elements = $.ui.resolve(e)
    const { Box, Button, Text } = elements
    // Mobile has no text fields; there any other color is set with the command.
    const Input = 'Input' in elements ? elements.Input : null
    const current = await read($, snapshot)
    const shown = await read($, page)
    const chosenId = await read($, slotAtom)
    const chosen = COLOR_SLOTS.find(s => s.id === chosenId) ?? COLOR_SLOTS[0]
    const colors = await read($, colorsAtom)
    const p = palette(colors)
    const now = await $.clock.now()
    const rows = e.viewport?.rows ?? 24

    // A background has to fill the whole pane, not just the height of its content.
    const fill = p.background ? { backgroundColor: p.background, width: e.props.bodyColumns, minHeight: e.props.scroll.bodyRows } : {}
    const dim = (text: string) => (
      <Text color={p.text} dimColor>
        {text}
      </Text>
    )
    const toPage = (key: string, label: string, to: Page) => <Button key={key} label={label} onPress={() => update($, page, () => to)} />
    // The engine's own close mark is small; this one is a full button.
    const closeButton = <Button key="close" label="Close" role="dismiss" onPress={() => $.ui.close({ id: PANE })} />

    if (shown === 'style') {
      const choose = (name: PaletteName) => <Button key={`palette-${name}`} label={name} onPress={() => saveColors($, paletteColors(name))} />
      const pick = (slot: typeof chosen.id, value: string) => saveColors($, { ...colors, [slot]: value === 'default' ? '' : value })
      const custom = async (text: string) => {
        const parsed = parseColorEntry(text, chosen.id)
        if (parsed) {
          await saveColors($, { ...colors, [parsed.slot]: parsed.value })
        } else {
          $.ui.toast('Type a hex code such as #00ff88 or a color name, or "<slot> <color>".')
        }
      }
      const chosenValue = colors[chosen.id]

      return (
        <Box flexDirection="column" gap={1} {...fill}>
          <Box gap={1} flexWrap="wrap">
            <Text bold color={p.accent}>
              Style
            </Text>
            {toPage('back', 'Back', 'chats')}
            {closeButton}
          </Box>
          <Box flexDirection="column" borderStyle="round" borderColor={p.border} paddingX={1}>
            <Text bold color={p.text}>
              Palette
            </Text>
            <Box gap={1}>{PALETTES.map(choose)}</Box>
            {dim('Only this pane changes, never your Claude theme or other mods.')}
          </Box>
          <Box flexDirection="column" borderStyle="round" borderColor={p.border} paddingX={1}>
            <Text bold color={p.text}>
              Colors
            </Text>
            {dim('Pick a color to change, then a new value:')}
            <Box gap={1} flexWrap="wrap">
              {COLOR_SLOTS.map(slot => (
                <Button
                  key={`slot-${slot.id}`}
                  label={slot.label}
                  variant={slot.id === chosen.id ? 'primary' : undefined}
                  onPress={() => update($, slotAtom, () => slot.id)}
                />
              ))}
            </Box>
            <Box flexDirection="column">
              {COLOR_SLOTS.map(slot => {
                const value = colors[slot.id]
                const isChosen = slot.id === chosen.id

                return (
                  <Box key={`value-${slot.id}`} gap={1}>
                    <Text color={value.trim() || DEFAULT_COLOR[slot.id]}>■</Text>
                    <Text bold={isChosen} color={p.text}>{`${isChosen ? '▸' : ' '} ${slot.label.padEnd(10)}`}</Text>
                    {dim(describeColor(value, slot.id))}
                  </Box>
                )
              })}
            </Box>
            <Box gap={1} flexWrap="wrap">
              {['default', ...NAMED_COLORS].map(name => {
                const isCurrent = name === 'default' ? chosenValue === '' : chosenValue.toLowerCase() === name

                return (
                  <Button key={`pick-${name}`} variant={isCurrent ? 'primary' : undefined} onPress={() => pick(chosen.id, name)}>
                    {isCurrent ? '✓ ' : ''}
                    <Text color={name === 'default' ? DEFAULT_COLOR[chosen.id] : name}>■</Text> {name}
                  </Button>
                )
              })}
            </Box>
            {Input && (
              <Input
                key="custom-color"
                label={`Hex for ${chosen.label} `}
                placeholder="#rrggbb, a color name, or default"
                submitLabel="apply"
                onSubmit={custom}
              />
            )}
            {dim(
              Input
                ? `Applies to "${chosen.label}". Other slot: "<slot> <color>" with ${COLOR_SLOTS.map(s => s.id).join(', ')}.`
                : 'Without a text field: /roundhouse style on a surface that has one.',
            )}
          </Box>
        </Box>
      )
    }

    const header = (
      <Box gap={1} flexWrap="wrap">
        <Text bold color={p.accent}>
          roundhouse
        </Text>
        <Button key="new-chat" label="New chat" variant="primary" onPress={() => startNewChat($)} />
        <Button key="refresh" label="Refresh" onPress={() => refresh($)} />
        {toPage('to-style', 'Style', 'style')}
        {closeButton}
      </Box>
    )

    if (current === null) {
      return (
        <Box flexDirection="column" {...fill}>
          {header}
          {dim('Press Refresh to load the chats.')}
        </Box>
      )
    }

    if (current.kind === 'unavailable') {
      return (
        <Box flexDirection="column" {...fill}>
          {header}
          <Text color={p.text}>roundhouse is not available: {current.reason}</Text>
          {dim('Install it from https://github.com/LuGG000/roundhouse and put it on your PATH.')}
        </Box>
      )
    }

    if (current.kind === 'incompatible') {
      return (
        <Box flexDirection="column" {...fill}>
          {header}
          <Text color={p.text}>
            roundhouse reports list schema {current.schema === null ? 'unknown' : current.schema}, which this mod does not read.
          </Text>
          {dim('Update roundhouse and this mod to matching versions.')}
        </Box>
      )
    }

    const room = Math.max(1, Math.floor((rows - CHROME_ROWS) / CHAT_ROWS))
    const shownChats = current.chats.slice(0, room)
    const hidden = current.chats.length - shownChats.length

    return (
      <Box flexDirection="column" {...fill}>
        {header}
        <Text color={p.text} dimColor wrap="truncate-start">
          {current.cwd}
        </Text>
        {current.chats.length === 0 && dim('No chats in this folder yet.')}
        {shownChats.map(chat => {
          const isCurrent = chat.id === current.sessionId

          return (
            <Box key={`chat-${chat.id}`} flexDirection="column" marginTop={1}>
              <Text wrap="truncate-end">
                <Text bold color={p.text}>
                  {oneLine(chat.title)}
                </Text>
                {isCurrent ? (
                  <Text color={p.accent}>{'  this chat'}</Text>
                ) : (
                  <Text color={p.text} dimColor>
                    {'  '}
                    {ago(chat.modified, now)}
                  </Text>
                )}
              </Text>
              <Box flexDirection="row" gap={1}>
                {!isCurrent && <Button key={`switch-${chat.id}`} label="Switch here" onPress={() => switchTo($, chat)} />}
                <Button key={`window-${chat.id}`} label="New window" onPress={() => openWindow($, chat)} />
              </Box>
            </Box>
          )
        })}
        {hidden > 0 && dim(`and ${hidden} more; run roundhouse for the full list`)}
      </Box>
    )
  })
}
