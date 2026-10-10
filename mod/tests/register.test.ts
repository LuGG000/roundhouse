import { expect, mock, test } from 'claude-code/testing'
import type { Engine } from 'claude-code/testing'
import type { On } from 'claude-code'

const PANE = {
  title: 'Chats',
  isFocused: false,
  bodyColumns: 80,
  placement: 'inline',
  scroll: { offset: 0, bodyRows: 20 },
  view: {},
} as const

const LIST = JSON.stringify({
  schema: 1,
  sessions: [
    { id: 'current-1', title: 'This very chat', last_prompt: 'now', modified: '2026-10-11T11:59:50Z' },
    { id: 'old-2', title: 'Older chat', last_prompt: 'before', modified: '2026-10-11T09:00:00Z' },
  ],
})

type Run = { exitCode: number; stdout: string; stderr: string }

/** The world beneath the plugin: a clock, a session, `/resume` and `/clear`, and the roundhouse command. */
function world(on: On, answer: (argv: readonly string[]) => Run | null) {
  const calls: string[][] = []
  const commands: { command: string; args: string }[] = []
  mock.clock(on, { now: Date.parse('2026-10-11T12:00:00Z') })
  on('session.id', () => ({ value: 'current-1' }))
  on('session.cwd', () => ({ value: '/work/repo' }))
  on('ui.open', () => ({ value: { isPlaced: true } }))
  on('command.run', (_$, e) => {
    commands.push({ command: e.command, args: e.args })

    return { text: 'ok' }
  })
  on('process.run', (_$, e) => {
    calls.push([...e.argv])
    const run = answer(e.argv)

    return run ? { value: { ...run, isStdoutTruncated: false, isStderrTruncated: false } } : { deny: 'spawn roundhouse ENOENT' }
  })

  return { calls, commands }
}

/** Runs a slash command as typed in the composer. */
function runCommand($: Engine, command: string, args = '') {
  return $.command.run({ command, args, origin: { kind: 'composer' }, presentation: { isFullscreen: false, columns: 80 } })
}

const ok = (stdout: string): Run => ({ exitCode: 0, stdout, stderr: '' })

async function mount($: Engine, surface: 'terminal' | 'desktop' = 'terminal') {
  return $.ui.mount({ plugin: 'roundhouse-mod', surface, component: 'Pane', props: PANE, requestId: 'roundhouse' })
}

test('the pane lists the chats of the folder and marks the current one', async ($, on) => {
  const { calls } = world(on, () => ok(LIST))
  await runCommand($, 'roundhouse')
  expect(calls[0]).toEqual(['roundhouse', 'list', '--json', '--folder', '/work/repo'])

  for (const surface of ['terminal', 'desktop'] as const) {
    const ui = await mount($, surface)
    expect(await ui.find({ type: 'Text', text: /This very chat/ })).toBeDefined()
    expect(await ui.find({ type: 'Text', text: /this chat/ })).toBeDefined()
    expect(await ui.find({ type: 'Text', text: /Older chat/ })).toBeDefined()
    expect(await ui.find({ type: 'Text', text: /3h ago/ })).toBeDefined()
    expect(await ui.find({ key: 'switch-current-1' })).toBeUndefined()
    expect(await ui.find({ key: 'switch-old-2' })).toBeDefined()
    expect(await ui.find({ key: 'window-current-1' })).toBeDefined()
    await ui.unmount()
  }
})

test('Switch here resumes the chat in this session', async ($, on) => {
  const { commands } = world(on, () => ok(LIST))
  await runCommand($, 'roundhouse')

  const ui = await mount($)
  await ui.press({ key: 'switch-old-2' })
  expect(commands).toContainEqual({ command: 'resume', args: 'old-2' })
  await ui.unmount()
})

test('New chat clears the session', async ($, on) => {
  const { commands } = world(on, () => ok(LIST))
  await runCommand($, 'roundhouse')

  const ui = await mount($)
  await ui.press({ key: 'new-chat' })
  expect(commands).toContainEqual({ command: 'clear', args: '' })
  await ui.unmount()
})

test('New window asks roundhouse to open the chat in a terminal', async ($, on) => {
  const { calls } = world(on, () => ok(LIST))
  await runCommand($, 'roundhouse')

  const ui = await mount($)
  await ui.press({ key: 'window-old-2' })
  expect(calls).toContainEqual(['roundhouse', 'open', 'old-2', '--terminal'])
  await ui.unmount()
})

test('Refresh reads the folder again', async ($, on) => {
  const { calls } = world(on, () => ok(LIST))
  await runCommand($, 'roundhouse')

  const ui = await mount($)
  await ui.press({ key: 'refresh' })
  expect(calls.filter(argv => argv[1] === 'list')).toHaveLength(2)
  await ui.unmount()
})

test('a folder without chats says so', async ($, on) => {
  world(on, () => ok(JSON.stringify({ schema: 1, sessions: [] })))
  await runCommand($, 'roundhouse')

  const ui = await mount($)
  expect(await ui.find({ type: 'Text', text: /No chats in this folder/ })).toBeDefined()
  await ui.unmount()
})

test('a missing roundhouse is explained in the pane and nowhere else', async ($, on) => {
  world(on, () => null)
  await runCommand($, 'roundhouse')

  const ui = await mount($)
  expect(await ui.find({ type: 'Text', text: /not available/ })).toBeDefined()
  expect(await ui.find({ key: 'refresh' })).toBeDefined()
  await ui.unmount()
})

test('an incompatible schema is explained in the pane', async ($, on) => {
  world(on, () => ok(JSON.stringify({ schema: 7, sessions: [] })))
  await runCommand($, 'roundhouse')

  const ui = await mount($)
  expect(await ui.find({ type: 'Text', text: /schema 7/ })).toBeDefined()
  await ui.unmount()
})

test('a failing roundhouse shows its message', async ($, on) => {
  world(on, () => ({ exitCode: 1, stdout: '', stderr: 'roundhouse: unreadable config\n' }))
  await runCommand($, 'roundhouse')

  const ui = await mount($)
  expect(await ui.find({ type: 'Text', text: /unreadable config/ })).toBeDefined()
  await ui.unmount()
})
