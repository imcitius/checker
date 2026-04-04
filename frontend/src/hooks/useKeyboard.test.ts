import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { renderHook } from '@testing-library/react'
import { useKeyboard } from './useKeyboard'

function makeActions() {
  return {
    onNavigateDown: vi.fn(),
    onNavigateUp: vi.fn(),
    onExpand: vi.fn(),
    onCollapse: vi.fn(),
    onFocusSearch: vi.fn(),
    onToggleGroup: vi.fn(),
    onCommandPalette: vi.fn(),
  }
}

function fireKey(key: string, opts: Partial<KeyboardEventInit> = {}) {
  window.dispatchEvent(new KeyboardEvent('keydown', { key, bubbles: true, ...opts }))
}

describe('useKeyboard', () => {
  let actions: ReturnType<typeof makeActions>

  beforeEach(() => {
    actions = makeActions()
  })

  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('calls onNavigateDown on "j"', () => {
    renderHook(() => useKeyboard(actions))
    fireKey('j')
    expect(actions.onNavigateDown).toHaveBeenCalledOnce()
  })

  it('calls onNavigateUp on "k"', () => {
    renderHook(() => useKeyboard(actions))
    fireKey('k')
    expect(actions.onNavigateUp).toHaveBeenCalledOnce()
  })

  it('calls onExpand on "Enter"', () => {
    renderHook(() => useKeyboard(actions))
    fireKey('Enter')
    expect(actions.onExpand).toHaveBeenCalledOnce()
  })

  it('calls onFocusSearch on "/"', () => {
    renderHook(() => useKeyboard(actions))
    fireKey('/')
    expect(actions.onFocusSearch).toHaveBeenCalledOnce()
  })

  it('calls onToggleGroup on "g"', () => {
    renderHook(() => useKeyboard(actions))
    fireKey('g')
    expect(actions.onToggleGroup).toHaveBeenCalledOnce()
  })

  it('calls onCommandPalette on Cmd+K', () => {
    renderHook(() => useKeyboard(actions))
    fireKey('k', { metaKey: true })
    expect(actions.onCommandPalette).toHaveBeenCalledOnce()
  })

  it('calls onCommandPalette on Ctrl+K', () => {
    renderHook(() => useKeyboard(actions))
    fireKey('k', { ctrlKey: true })
    expect(actions.onCommandPalette).toHaveBeenCalledOnce()
  })

  it('calls onCollapse on Escape', () => {
    renderHook(() => useKeyboard(actions))
    fireKey('Escape')
    expect(actions.onCollapse).toHaveBeenCalledOnce()
  })

  it('does not call navigation shortcuts when target is an input', () => {
    renderHook(() => useKeyboard(actions))

    const input = document.createElement('input')
    document.body.appendChild(input)
    input.focus()

    input.dispatchEvent(new KeyboardEvent('keydown', { key: 'j', bubbles: true }))
    expect(actions.onNavigateDown).not.toHaveBeenCalled()

    document.body.removeChild(input)
  })

  it('Cmd+K works even when focused on input', () => {
    renderHook(() => useKeyboard(actions))

    const input = document.createElement('input')
    document.body.appendChild(input)
    input.focus()

    // Cmd+K dispatched on window still fires
    fireKey('k', { metaKey: true })
    expect(actions.onCommandPalette).toHaveBeenCalledOnce()

    document.body.removeChild(input)
  })

  it('cleans up listener on unmount', () => {
    const spy = vi.spyOn(window, 'removeEventListener')
    const { unmount } = renderHook(() => useKeyboard(actions))

    unmount()
    expect(spy).toHaveBeenCalledWith('keydown', expect.any(Function))
  })
})
