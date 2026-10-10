import { expect, test } from 'claude-code/testing'

import { NO_COLORS, describeColor, palette, paletteColors, parseColorEntry, parseColors } from '../hooks/palette'

test('palette resolves empty slots to the defaults and keeps set ones', () => {
  expect(palette(NO_COLORS)).toEqual({ background: undefined, text: undefined, accent: 'claude', border: 'subtle' })
  expect(palette({ background: ' #101010 ', text: 'white', accent: '', border: 'red' })).toEqual({
    background: '#101010',
    text: 'white',
    accent: 'claude',
    border: 'red',
  })
})

test('a named palette sets every slot, and default clears them all', () => {
  const dark = paletteColors('dark')
  expect(Object.values(dark).every(value => value !== '')).toBe(true)
  expect(paletteColors('default')).toEqual(NO_COLORS)

  // The result is a copy: editing it must not change the palette.
  dark.text = 'red'
  expect(paletteColors('dark').text).not.toBe('red')
})

test('describeColor names the default of an empty slot', () => {
  expect(describeColor('', 'accent')).toBe('default (claude)')
  expect(describeColor('', 'background')).toBe('default (Claude background)')
  expect(describeColor('', 'text')).toBe('default (Claude text color)')
  expect(describeColor('#00ff88', 'accent')).toBe('#00ff88')
})

test('parseColors keeps text values and drops everything else', () => {
  expect(parseColors({ background: '#111', text: 5, accent: ' red ', border: null, extra: 'x' })).toEqual({
    background: '#111',
    text: '',
    accent: 'red',
    border: '',
  })
  expect(parseColors(undefined)).toEqual(NO_COLORS)
  expect(parseColors('nonsense')).toEqual(NO_COLORS)
})

test('parseColorEntry takes one color for the chosen slot or a slot and a color', () => {
  expect(parseColorEntry('#00ff88', 'accent')).toEqual({ slot: 'accent', value: '#00ff88' })
  expect(parseColorEntry('00FF88', 'text')).toEqual({ slot: 'text', value: '#00FF88' })
  expect(parseColorEntry('background #1e1e1e', 'accent')).toEqual({ slot: 'background', value: '#1e1e1e' })
  expect(parseColorEntry('default', 'border')).toEqual({ slot: 'border', value: '' })
  expect(parseColorEntry('nonsense red', 'border')).toBeNull()
  expect(parseColorEntry('   ', 'border')).toBeNull()
  expect(parseColorEntry('a b c', 'border')).toBeNull()
})
