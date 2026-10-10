import { atom, read, update } from 'claude-code'
import type { EngineInterface, Register } from 'claude-code'

import type { Chat, Snapshot } from '../types'
import { ago, oneLine, parseList } from './chats'

const PANE = 'roundhouse'

const snapshot = atom({ plugin: 'roundhouse-mod', key: 'snapshot' } as const, null)

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

/** Continues the conversation in this session with the chosen chat, as if `/resume <id>` was typed. */
async function switchTo($: Engine, chat: Chat): Promise<void> {
  try {
    await $.command.run({ command: 'resume', args: chat.id })
  } catch (error) {
    $.ui.toast(`Could not resume: ${error instanceof Error ? error.message : String(error)}`)
  }
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
}

export const register: Register = on => {
  on('session.start', async ($, e, next) => {
    await $.command.register({
      name: 'roundhouse',
      description: 'Chats of this folder: switch to one or open it in a new terminal',
    })

    return next(e)
  })

  on('command.run', { command: 'roundhouse' }, async $ => {
    await refresh($)
    await $.ui.open({ id: PANE, title: 'Chats' })

    return { text: 'Roundhouse pane opened.' }
  })

  on('ui.render', { component: 'Pane', requestId: PANE }, async ($, e) => {
    const { Box, Button, Text } = $.ui.resolve(e)
    const current = await read($, snapshot)
    const now = await $.clock.now()
    const rows = e.viewport?.rows ?? 24

    const header = (
      <Box flexDirection="row" gap={1}>
        <Button key="new-chat" label="New chat" variant="primary" onPress={() => startNewChat($)} />
        <Button key="refresh" label="Refresh" onPress={() => refresh($)} />
        <Button key="close" label="Close" role="dismiss" onPress={() => $.ui.close({ id: PANE })} />
      </Box>
    )

    if (current === null) {
      return (
        <Box flexDirection="column">
          {header}
          <Text dimColor>Press Refresh to load the chats.</Text>
        </Box>
      )
    }

    if (current.kind === 'unavailable') {
      return (
        <Box flexDirection="column">
          {header}
          <Text>roundhouse is not available: {current.reason}</Text>
          <Text dimColor>Install it from https://github.com/LuGG000/roundhouse and put it on your PATH.</Text>
        </Box>
      )
    }

    if (current.kind === 'incompatible') {
      return (
        <Box flexDirection="column">
          {header}
          <Text>
            roundhouse reports list schema {current.schema === null ? 'unknown' : current.schema}, which this mod does not read.
          </Text>
          <Text dimColor>Update roundhouse and this mod to matching versions.</Text>
        </Box>
      )
    }

    const room = Math.max(1, Math.floor((rows - CHROME_ROWS) / CHAT_ROWS))
    const shown = current.chats.slice(0, room)
    const hidden = current.chats.length - shown.length

    return (
      <Box flexDirection="column">
        {header}
        <Text dimColor wrap="truncate-start">
          {current.cwd}
        </Text>
        {current.chats.length === 0 && <Text dimColor>No chats in this folder yet.</Text>}
        {shown.map(chat => {
          const isCurrent = chat.id === current.sessionId

          return (
            <Box key={`chat-${chat.id}`} flexDirection="column" marginTop={1}>
              <Text wrap="truncate-end">
                <Text bold>{oneLine(chat.title)}</Text>
                <Text dimColor>
                  {'  '}
                  {isCurrent ? 'this chat' : ago(chat.modified, now)}
                </Text>
              </Text>
              <Box flexDirection="row" gap={1}>
                {!isCurrent && <Button key={`switch-${chat.id}`} label="Switch here" onPress={() => switchTo($, chat)} />}
                <Button key={`window-${chat.id}`} label="New window" onPress={() => openWindow($, chat)} />
              </Box>
            </Box>
          )
        })}
        {hidden > 0 && <Text dimColor>and {hidden} more; run roundhouse for the full list</Text>}
      </Box>
    )
  })
}
