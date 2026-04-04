import { render, screen } from '@testing-library/react'
import { describe, it, expect, beforeAll } from 'vitest'
import { EventLog } from './EventLog'
import type { EventLogEntry } from '@/hooks/useEventLog'

beforeAll(() => {
  Element.prototype.scrollIntoView = () => {}
})

describe('EventLog', () => {
  it('shows waiting message when entries are empty', () => {
    render(<EventLog entries={[]} />)
    expect(screen.getByText(/Waiting for state transitions/)).toBeInTheDocument()
  })

  it('renders event log header when entries exist', () => {
    const entries: EventLogEntry[] = [
      {
        id: 1,
        timestamp: new Date('2026-04-04T10:00:00'),
        checkName: 'API Health',
        checkUUID: 'uuid-1',
        status: 'healthy',
        message: 'Response OK',
      },
    ]
    render(<EventLog entries={entries} />)
    expect(screen.getByText('Event Log')).toBeInTheDocument()
  })

  it('renders check names and statuses', () => {
    const entries: EventLogEntry[] = [
      {
        id: 1,
        timestamp: new Date('2026-04-04T10:00:00'),
        checkName: 'API Health',
        checkUUID: 'uuid-1',
        status: 'healthy',
      },
      {
        id: 2,
        timestamp: new Date('2026-04-04T10:01:00'),
        checkName: 'DB Check',
        checkUUID: 'uuid-2',
        status: 'unhealthy',
        message: 'Connection timeout',
      },
    ]
    render(<EventLog entries={entries} />)
    expect(screen.getByText('API Health')).toBeInTheDocument()
    expect(screen.getByText('DB Check')).toBeInTheDocument()
    expect(screen.getByText('healthy')).toBeInTheDocument()
    expect(screen.getByText('unhealthy')).toBeInTheDocument()
  })

  it('displays optional message when provided', () => {
    const entries: EventLogEntry[] = [
      {
        id: 1,
        timestamp: new Date('2026-04-04T10:00:00'),
        checkName: 'Redis',
        checkUUID: 'uuid-3',
        status: 'unhealthy',
        message: 'Connection refused',
      },
    ]
    render(<EventLog entries={entries} />)
    expect(screen.getByText('Connection refused')).toBeInTheDocument()
  })

  it('applies correct color class for healthy status', () => {
    const entries: EventLogEntry[] = [
      {
        id: 1,
        timestamp: new Date('2026-04-04T10:00:00'),
        checkName: 'Check1',
        checkUUID: 'uuid-1',
        status: 'healthy',
      },
    ]
    render(<EventLog entries={entries} />)
    const statusEl = screen.getByText('healthy')
    expect(statusEl.className).toContain('text-healthy')
  })
})
