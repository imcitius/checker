import { describe, it, expect } from 'vitest'
import { renderHook } from '@testing-library/react'
import { useEventLog } from './useEventLog'
import type { Check } from '@/lib/websocket'

function makeCheck(overrides: Partial<Check> = {}): Check {
  return {
    ID: '1',
    Name: 'test-check',
    Project: 'default',
    Healthcheck: '',
    LastResult: true,
    LastExec: '',
    LastPing: '',
    Enabled: true,
    UUID: 'uuid-1',
    CheckType: 'http',
    Message: '',
    Host: '',
    Periodicity: '30s',
    URL: '',
    IsSilenced: false,
    ...overrides,
  }
}

describe('useEventLog', () => {
  it('returns empty entries initially', () => {
    const { result } = renderHook(() =>
      useEventLog([], new Map())
    )
    expect(result.current.entries).toEqual([])
  })

  it('does not add entries on first load (initialization)', () => {
    const checks = [makeCheck()]
    const prev = new Map([['uuid-1', makeCheck({ LastResult: false })]])

    const { result } = renderHook(() =>
      useEventLog(checks, prev)
    )
    // First render sets initialized=true but skips processing
    expect(result.current.entries).toEqual([])
  })

  it('detects status transition after initialization', () => {
    const prev = new Map([['uuid-1', makeCheck({ LastResult: true })]])
    const checks = [makeCheck({ LastResult: true })]

    const { result, rerender } = renderHook(
      ({ c, p }) => useEventLog(c, p),
      { initialProps: { c: checks, p: prev } }
    )

    // First render initializes, no entries
    expect(result.current.entries).toEqual([])

    // Now simulate a status change
    const newPrev = new Map([['uuid-1', makeCheck({ LastResult: true })]])
    const newChecks = [makeCheck({ LastResult: false })]

    rerender({ c: newChecks, p: newPrev })
    expect(result.current.entries).toHaveLength(1)
    expect(result.current.entries[0].status).toBe('unhealthy')
  })

  it('detects enable/disable transition', () => {
    const prev = new Map([['uuid-1', makeCheck({ Enabled: true })]])
    const checks = [makeCheck({ Enabled: true })]

    const { result, rerender } = renderHook(
      ({ c, p }) => useEventLog(c, p),
      { initialProps: { c: checks, p: prev } }
    )

    // Initialization
    expect(result.current.entries).toEqual([])

    // Disable the check
    const newPrev = new Map([['uuid-1', makeCheck({ Enabled: true })]])
    const newChecks = [makeCheck({ Enabled: false })]

    rerender({ c: newChecks, p: newPrev })
    expect(result.current.entries).toHaveLength(1)
    expect(result.current.entries[0].status).toBe('disabled')
  })

  it('skips when previousChecks is empty', () => {
    const checks = [makeCheck()]

    const { result, rerender } = renderHook(
      ({ c, p }) => useEventLog(c, p),
      { initialProps: { c: checks, p: new Map([['uuid-1', makeCheck()]]) } }
    )

    // Pass empty prev on rerender
    rerender({ c: checks, p: new Map() })
    expect(result.current.entries).toEqual([])
  })

  it('skips when check has no previous entry', () => {
    const checks = [makeCheck()]
    const prev = new Map([['uuid-1', makeCheck()]])

    const { result, rerender } = renderHook(
      ({ c, p }) => useEventLog(c, p),
      { initialProps: { c: checks, p: prev } }
    )

    // Add a new check with different UUID that has no previous
    const newChecks = [makeCheck({ UUID: 'uuid-2', Name: 'new' })]
    rerender({ c: newChecks, p: prev })
    expect(result.current.entries).toEqual([])
  })
})
