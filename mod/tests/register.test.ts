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
function world(on: On, answer: (argv: readonly string[]) => Run | null, entries: Record<string, unknown> = {}) {
  const calls: string[][] = []
  const commands: { command: string; args: string }[] = []
  mock.clock(on, { now: Date.parse('2026-10-11T12:00:00Z') })
  const store = new Map<string, unknown>(Object.entries(entries))
  on('store.get', (_$, e) => ({ value: store.get(e.key) }))
  on('store.set', (_$, e) => {
    store.set(e.key, JSON.parse(JSON.stringify(e.value)))

    return { value: undefined }
  })
  const session = { id: 'current-1' }
  on('session.id', () => ({ value: session.id }))
  on('session.cwd', () => ({ value: '/work/repo' }))
  on('ui.open', () => ({ value: { isPlaced: true } }))
  on('command.run', (_$, e) => {
    commands.push({ command: e.command, args: e.args })
    if (e.command === 'resume') session.id = e.args.trim()

    return { text: 'ok' }
  })
  on('process.run', (_$, e) => {
    calls.push([...e.argv])
    const run = answer(e.argv)

    return run ? { value: { ...run, isStdoutTruncated: false, isStderrTruncated: false } } : { deny: 'spawn roundhouse ENOENT' }
  })

  return { calls, commands, session, store }
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

test('Switch here resumes the chat in this session and reads the folder again', async ($, on) => {
  const { commands } = world(on, () => ok(LIST))
  await runCommand($, 'roundhouse')

  const ui = await mount($)
  await ui.press({ key: 'switch-old-2' })
  expect(commands).toContainEqual({ command: 'resume', args: 'old-2' })

  // No Refresh press needed: the pane already marks the new chat as the current one.
  expect(await ui.find({ key: 'switch-old-2' })).toBeUndefined()
  expect(await ui.find({ key: 'switch-current-1' })).toBeDefined()
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

test('the style page sets a palette and keeps it in the store', async ($, on) => {
  const { store } = world(on, () => ok(LIST))
  await runCommand($, 'roundhouse')

  const ui = await mount($)
  await ui.press({ key: 'to-style' })
  expect(await ui.find({ type: 'Text', text: /Palette/ })).toBeDefined()
  expect(await ui.find({ type: 'Text', text: /default \(claude\)/ })).toBeDefined()

  await ui.press({ key: 'palette-dark' })
  expect(store.get('colors')).toEqual({ background: '#1e1e1e', text: '#d4d4d4', accent: '#d97757', border: '#555555' })
  expect(await ui.find({ type: 'Text', text: /#d97757/ })).toBeDefined()

  await ui.press({ key: 'palette-default' })
  expect(store.get('colors')).toEqual({ background: '', text: '', accent: '', border: '' })

  await ui.press({ key: 'back' })
  expect(await ui.find({ key: 'new-chat' })).toBeDefined()
  await ui.unmount()
})

test('a color picked for the chosen slot is saved, and a typed one too', async ($, on) => {
  const { store } = world(on, () => ok(LIST))
  await runCommand($, 'roundhouse', 'style')

  const ui = await mount($)
  await ui.press({ key: 'slot-accent' })
  await ui.press({ key: 'pick-red' })
  expect((store.get('colors') as Record<string, string>).accent).toBe('red')

  await ui.input({ key: 'custom-color', text: '00ff88' })
  expect((store.get('colors') as Record<string, string>).accent).toBe('#00ff88')

  await ui.input({ key: 'custom-color', text: 'background #101010' })
  expect((store.get('colors') as Record<string, string>).background).toBe('#101010')

  await ui.press({ key: 'pick-default' })
  expect((store.get('colors') as Record<string, string>).accent).toBe('')
  await ui.unmount()
})

test('the saved colors are read again when the command runs', async ($, on) => {
  world(on, () => ok(LIST), { colors: { background: '#202020', text: '', accent: 'cyan', border: '' } })
  await runCommand($, 'roundhouse', 'style')

  const ui = await mount($)
  expect(await ui.find({ type: 'Text', text: /#202020/ })).toBeDefined()
  expect(await ui.find({ type: 'Text', text: /cyan/ })).toBeDefined()
  await ui.unmount()
})

test('saved colors still apply when the session state started over, as after a resume', async ($, on) => {
  world(on, () => ok(LIST), { colors: { background: '#202020', text: '', accent: 'cyan', border: '' } })

  // No /roundhouse run in this session: the atoms hold only their initial values, the store holds the colors.
  const ui = await mount($)
  await ui.press({ key: 'to-style' })
  expect(await ui.find({ type: 'Text', text: /#202020/ })).toBeDefined()
  expect(await ui.find({ type: 'Text', text: /cyan/ })).toBeDefined()
  await ui.unmount()
})

const WITH_UPDATE = JSON.stringify({
  schema: 1,
  version: '0.2.0',
  update: { version: '0.3.0', url: 'https://example.test/v0.3.0' },
  sessions: JSON.parse(LIST).sessions,
})

test('the pane says when a newer roundhouse exists', async ($, on) => {
  world(on, () => ok(WITH_UPDATE))
  await runCommand($, 'roundhouse')

  for (const surface of ['terminal', 'desktop'] as const) {
    const ui = await mount($, surface)
    expect(await ui.find({ type: 'Text', text: /roundhouse 0\.3\.0 is available \(you have 0\.2\.0\)/ })).toBeDefined()
    expect(await ui.find({ type: 'Text', text: /Nothing changes before/ })).toBeDefined()
    expect(await ui.find({ key: 'update' })).toBeDefined()
    // The chats are still there.
    expect(await ui.find({ key: 'switch-old-2' })).toBeDefined()
    await ui.unmount()
  }
})

test('there is no update notice for a current roundhouse', async ($, on) => {
  world(on, () => ok(LIST))
  await runCommand($, 'roundhouse')

  const ui = await mount($)
  expect(await ui.find({ key: 'update' })).toBeUndefined()
  expect(await ui.find({ type: 'Text', text: /is available/ })).toBeUndefined()
  await ui.unmount()
})

test('Update opens a window to confirm in and installs nothing itself', async ($, on) => {
  const { calls } = world(on, () => ok(WITH_UPDATE))
  await runCommand($, 'roundhouse')

  const ui = await mount($)
  await ui.press({ key: 'update' })
  expect(calls).toContainEqual(['roundhouse', 'update', '--terminal'])
  // Never an install from the pane: no call without --terminal, no --yes.
  expect(calls.filter(argv => argv[1] === 'update').every(argv => argv.includes('--terminal'))).toBe(true)
  expect(calls.flat()).not.toContain('--yes')
  await ui.unmount()
})
