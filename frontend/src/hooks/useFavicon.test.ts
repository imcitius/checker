import { describe, it, expect, beforeEach, afterEach } from 'vitest'
import { renderHook } from '@testing-library/react'
import { useFavicon } from './useFavicon'

describe('useFavicon', () => {
  let originalTitle: string

  beforeEach(() => {
    originalTitle = document.title
    // Remove any existing favicon link
    document.querySelector('link[rel="icon"]')?.remove()
  })

  afterEach(() => {
    document.title = originalTitle
    document.querySelector('link[rel="icon"]')?.remove()
  })

  it('does nothing when totalEnabled is 0', () => {
    renderHook(() => useFavicon(0, 0))

    const link = document.querySelector<HTMLLinkElement>('link[rel="icon"]')
    expect(link).toBeNull()
  })

  it('sets healthy favicon when unhealthyCount is 0', () => {
    renderHook(() => useFavicon(0, 5))

    const link = document.querySelector<HTMLLinkElement>('link[rel="icon"]')
    expect(link).not.toBeNull()
    expect(link!.type).toBe('image/svg+xml')
    expect(link!.href).toContain('%2322c55e') // green color encoded
  })

  it('sets unhealthy favicon when unhealthyCount > 0', () => {
    renderHook(() => useFavicon(3, 10))

    const link = document.querySelector<HTMLLinkElement>('link[rel="icon"]')
    expect(link).not.toBeNull()
    expect(link!.href).toContain('%23ef4444') // red color encoded
  })

  it('sets title to "Checker" when all healthy', () => {
    renderHook(() => useFavicon(0, 5))

    expect(document.title).toBe('Checker')
  })

  it('sets title with failing count when unhealthy', () => {
    renderHook(() => useFavicon(3, 10))

    expect(document.title).toBe('(3 failing) Checker')
  })

  it('reuses existing link element', () => {
    const existingLink = document.createElement('link')
    existingLink.rel = 'icon'
    document.head.appendChild(existingLink)

    renderHook(() => useFavicon(0, 5))

    const links = document.querySelectorAll('link[rel="icon"]')
    expect(links.length).toBe(1)
  })

  it('updates when unhealthyCount changes', () => {
    const { rerender } = renderHook(
      ({ unhealthy, total }) => useFavicon(unhealthy, total),
      { initialProps: { unhealthy: 0, total: 5 } }
    )

    expect(document.title).toBe('Checker')

    rerender({ unhealthy: 2, total: 5 })
    expect(document.title).toBe('(2 failing) Checker')
  })
})
