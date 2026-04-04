import { render } from '@testing-library/react'
import { describe, it, expect } from 'vitest'
import { StatusDot } from './StatusDot'

describe('StatusDot', () => {
  it('renders a disabled dot when not enabled', () => {
    const { container } = render(<StatusDot healthy={true} enabled={false} />)
    const dot = container.querySelector('span')
    expect(dot).toBeInTheDocument()
    expect(dot?.className).toContain('bg-disabled')
  })

  it('renders a healthy dot when enabled and healthy', () => {
    const { container } = render(<StatusDot healthy={true} enabled={true} />)
    const dot = container.querySelector('span')
    expect(dot?.className).toContain('bg-healthy')
    expect(dot?.className).toContain('animate-pulse-healthy')
  })

  it('renders an unhealthy dot when enabled and not healthy', () => {
    const { container } = render(<StatusDot healthy={false} enabled={true} />)
    const dot = container.querySelector('span')
    expect(dot?.className).toContain('bg-unhealthy')
    expect(dot?.className).toContain('animate-pulse-unhealthy')
  })

  it('renders a silenced healthy dot with reduced opacity', () => {
    const { container } = render(<StatusDot healthy={true} enabled={true} silenced={true} />)
    const dot = container.querySelector('span')
    expect(dot?.className).toContain('bg-healthy')
    expect(dot?.className).toContain('opacity-50')
  })

  it('renders a silenced unhealthy dot as warning', () => {
    const { container } = render(<StatusDot healthy={false} enabled={true} silenced={true} />)
    const dot = container.querySelector('span')
    expect(dot?.className).toContain('bg-warning')
  })

  it('renders small size when size=sm', () => {
    const { container } = render(<StatusDot healthy={true} enabled={true} size="sm" />)
    const dot = container.querySelector('span')
    expect(dot?.className).toContain('h-2')
    expect(dot?.className).toContain('w-2')
  })

  it('renders partially silenced healthy dot', () => {
    const { container } = render(
      <StatusDot healthy={true} enabled={true} partiallySilenced={true} />
    )
    const dot = container.querySelector('span')
    expect(dot?.className).toContain('bg-healthy')
    expect(dot?.className).toContain('opacity-70')
  })
})
