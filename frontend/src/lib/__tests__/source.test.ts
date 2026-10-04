import { describe, expect, it } from 'vitest'
import { SOURCE_COLORS, getSourceColor, getSourceInitials } from '@/lib/source'

describe('source helpers', () => {
  it('builds up to two uppercase initials', () => {
    expect(getSourceInitials('nova makedonija')).toBe('NM')
    expect(getSourceInitials('Meta')).toBe('M')
    expect(getSourceInitials('a b c')).toBe('AB')
  })

  it('picks a stable colour from the palette', () => {
    expect(getSourceColor('Meta')).toBe(getSourceColor('Meta'))
    expect(SOURCE_COLORS).toContain(getSourceColor('Anything'))
  })
})
