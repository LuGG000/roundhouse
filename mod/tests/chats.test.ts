import { expect, test } from 'claude-code/testing'

import { ago, oneLine, parseList } from '../hooks/chats'

const NOW = Date.parse('2026-10-11T12:00:00Z')

test('parseList reads the chats and leaves out those without an id or a title', () => {
  const parsed = parseList(
    JSON.stringify({
      schema: 1,
      sessions: [
        { id: 'a', title: 'First', last_prompt: 'hello', modified: '2026-10-11T11:00:00Z', cwd: '/w', size: 1 },
        { id: 'b', title: '' },
        { title: 'no id' },
        { id: 'c', title: 'Third' },
      ],
    }),
  )

  expect(parsed).toEqual({
    kind: 'ok',
    chats: [
      { id: 'a', title: 'First', last_prompt: 'hello', modified: '2026-10-11T11:00:00Z' },
      { id: 'c', title: 'Third', last_prompt: '', modified: '' },
    ],
  })
})

test('parseList refuses another schema and garbage', () => {
  expect(parseList(JSON.stringify({ schema: 7, sessions: [] }))).toEqual({ kind: 'incompatible', schema: 7 })
  expect(parseList(JSON.stringify({ sessions: [] }))).toEqual({ kind: 'incompatible', schema: null })
  expect(parseList('not json')).toEqual({ kind: 'incompatible', schema: null })
  expect(parseList(JSON.stringify({ schema: 1 }))).toEqual({ kind: 'incompatible', schema: 1 })
})

test('ago is coarse and survives bad input', () => {
  expect(ago('2026-10-11T11:59:30Z', NOW)).toBe('just now')
  expect(ago('2026-10-11T11:55:00Z', NOW)).toBe('5m ago')
  expect(ago('2026-10-11T09:00:00Z', NOW)).toBe('3h ago')
  expect(ago('2026-10-09T12:00:00Z', NOW)).toBe('2d ago')
  expect(ago('2026-08-01T12:00:00Z', NOW)).toBe('2026-08-01')
  expect(ago('2026-10-11T13:00:00Z', NOW)).toBe('just now')
  expect(ago('', NOW)).toBe('')
})

test('oneLine collapses line breaks and spaces', () => {
  expect(oneLine('  fix\n  the   build ')).toBe('fix the build')
})
