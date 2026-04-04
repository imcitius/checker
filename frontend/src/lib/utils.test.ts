import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { cn, relativeTime, formatTime } from './utils'

describe('cn', () => {
  it('merges class names', () => {
    expect(cn('foo', 'bar')).toBe('foo bar')
  })

  it('handles conditional classes', () => {
    expect(cn('base', false && 'hidden', 'visible')).toBe('base visible')
  })

  it('deduplicates tailwind classes', () => {
    expect(cn('p-4', 'p-2')).toBe('p-2')
  })

  it('handles empty inputs', () => {
    expect(cn()).toBe('')
  })

  it('handles undefined and null', () => {
    expect(cn('a', undefined, null, 'b')).toBe('a b')
  })

  it('merges conflicting tailwind utilities', () => {
    expect(cn('text-red-500', 'text-blue-500')).toBe('text-blue-500')
  })
})

describe('relativeTime', () => {
  let nowSpy: ReturnType<typeof vi.spyOn>

  beforeEach(() => {
    nowSpy = vi.spyOn(Date, 'now')
  })

  afterEach(() => {
    nowSpy.mockRestore()
    vi.useRealTimers()
  })

  it('returns "Never" for empty string', () => {
    expect(relativeTime('')).toBe('Never')
  })

  it('returns "Never" for "Never" string', () => {
    expect(relativeTime('Never')).toBe('Never')
  })

  it('returns original string for invalid date', () => {
    expect(relativeTime('not-a-date')).toBe('not-a-date')
  })

  it('returns "just now" for < 5 seconds ago', () => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date('2026-01-01T12:00:03Z'))
    expect(relativeTime('2026-01-01T12:00:00Z')).toBe('just now')
  })

  it('returns seconds ago for < 60 seconds', () => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date('2026-01-01T12:00:30Z'))
    expect(relativeTime('2026-01-01T12:00:00Z')).toBe('30s ago')
  })

  it('returns minutes ago for < 60 minutes', () => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date('2026-01-01T12:05:00Z'))
    expect(relativeTime('2026-01-01T12:00:00Z')).toBe('5m ago')
  })

  it('returns hours ago for < 24 hours', () => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date('2026-01-01T15:00:00Z'))
    expect(relativeTime('2026-01-01T12:00:00Z')).toBe('3h ago')
  })

  it('returns days ago for >= 24 hours', () => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date('2026-01-03T12:00:00Z'))
    expect(relativeTime('2026-01-01T12:00:00Z')).toBe('2d ago')
  })
})

describe('formatTime', () => {
  it('returns "Never" for empty string', () => {
    expect(formatTime('')).toBe('Never')
  })

  it('returns "Never" for "Never" string', () => {
    expect(formatTime('Never')).toBe('Never')
  })

  it('returns original string for invalid date', () => {
    expect(formatTime('garbage')).toBe('garbage')
  })

  it('formats a valid date as 24h time string', () => {
    // toLocaleTimeString with hour12:false produces locale-dependent output
    // but we can verify it returns something that looks like a time
    const result = formatTime('2026-01-01T14:30:45Z')
    expect(result).toMatch(/\d{1,2}:\d{2}:\d{2}/)
  })
})
