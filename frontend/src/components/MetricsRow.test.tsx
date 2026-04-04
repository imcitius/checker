import { render, screen, act } from '@testing-library/react'
import { describe, it, expect, vi } from 'vitest'
import { MetricsRow } from './MetricsRow'
import type { CheckStats } from '@/hooks/useChecks'

describe('MetricsRow', () => {
  const stats: CheckStats = {
    total: 25,
    healthy: 20,
    unhealthy: 3,
    disabled: 2,
    silenced: 1,
  }

  beforeEach(() => {
    vi.useFakeTimers()
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  it('renders all four metric labels', () => {
    render(<MetricsRow stats={stats} />)
    expect(screen.getByText('Total')).toBeInTheDocument()
    expect(screen.getByText('Healthy')).toBeInTheDocument()
    expect(screen.getByText('Failing')).toBeInTheDocument()
    expect(screen.getByText('Disabled')).toBeInTheDocument()
  })

  it('displays animated count values after delays', async () => {
    render(<MetricsRow stats={stats} />)

    // Initially all show 0
    const zeros = screen.getAllByText('0')
    expect(zeros.length).toBe(4)

    // Advance past all animation delays (last card delay = 3 * 80ms = 240ms)
    await act(async () => {
      vi.advanceTimersByTime(300)
    })

    expect(screen.getByText('25')).toBeInTheDocument()
    expect(screen.getByText('20')).toBeInTheDocument()
    expect(screen.getByText('3')).toBeInTheDocument()
    expect(screen.getByText('2')).toBeInTheDocument()
  })

  it('renders zero stats correctly', () => {
    const zeroStats: CheckStats = {
      total: 0,
      healthy: 0,
      unhealthy: 0,
      disabled: 0,
      silenced: 0,
    }
    render(<MetricsRow stats={zeroStats} />)
    // All should be 0 even after animation
    vi.advanceTimersByTime(500)
    const zeros = screen.getAllByText('0')
    expect(zeros.length).toBe(4)
  })
})
