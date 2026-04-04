import { render, screen, waitFor } from '@testing-library/react'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { VersionBadge } from './VersionBadge'

describe('VersionBadge', () => {
  beforeEach(() => {
    vi.restoreAllMocks()
  })

  it('shows "build: unknown" on fetch error', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockRejectedValue(new Error('Network error'))
    )

    render(<VersionBadge />)

    await waitFor(() => {
      expect(screen.getByText('build: unknown')).toBeInTheDocument()
    })
  })

  it('renders nothing while loading', () => {
    // Never-resolving fetch to simulate loading state
    vi.stubGlobal(
      'fetch',
      vi.fn().mockReturnValue(new Promise(() => {}))
    )

    const { container } = render(<VersionBadge />)
    // Should render nothing (null) while loading and no error
    expect(container.textContent).toBe('')
  })

  it('displays short SHA on successful fetch', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue({
        ok: true,
        json: () =>
          Promise.resolve({
            version: 'abcdef1234567890',
            build_time: '2026-04-01T12:00:00Z',
            frontend_version: 'unknown',
          }),
      })
    )

    render(<VersionBadge />)

    await waitFor(() => {
      expect(screen.getByText('abcdef1')).toBeInTheDocument()
    })
  })

  it('shows error state on non-ok response', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue({
        ok: false,
        statusText: 'Internal Server Error',
      })
    )

    render(<VersionBadge />)

    await waitFor(() => {
      expect(screen.getByText('build: unknown')).toBeInTheDocument()
    })
  })
})
