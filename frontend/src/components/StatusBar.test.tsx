import { render, screen } from '@testing-library/react'
import { describe, it, expect } from 'vitest'
import { StatusBar } from './StatusBar'

describe('StatusBar', () => {
  it('displays connected status', () => {
    render(<StatusBar wsStatus="connected" />)
    expect(screen.getByText('connected')).toBeInTheDocument()
    expect(screen.getByText('ws:ok')).toBeInTheDocument()
  })

  it('displays disconnected status', () => {
    render(<StatusBar wsStatus="disconnected" />)
    expect(screen.getByText('disconnected')).toBeInTheDocument()
    expect(screen.getByText('ws:err')).toBeInTheDocument()
  })

  it('displays connecting status', () => {
    render(<StatusBar wsStatus="connecting" />)
    expect(screen.getByText('connecting')).toBeInTheDocument()
    expect(screen.getByText('ws:...')).toBeInTheDocument()
  })

  it('shows refresh interval', () => {
    render(<StatusBar wsStatus="connected" />)
    expect(screen.getByText('refresh 30s')).toBeInTheDocument()
  })

  it('applies correct color class for connected status', () => {
    render(<StatusBar wsStatus="connected" />)
    const statusText = screen.getByText('connected')
    expect(statusText.className).toContain('text-healthy')
  })
})
