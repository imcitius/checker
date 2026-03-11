import { useState, useCallback, useRef, useEffect } from 'react'
import type { Check } from '@/lib/websocket'

export interface EventLogEntry {
  id: number
  timestamp: Date
  checkName: string
  checkUUID: string
  status: 'healthy' | 'unhealthy' | 'disabled' | 'enabled'
  message?: string
}

const MAX_ENTRIES = 100

export function useEventLog(
  checks: Check[],
  previousChecks: Map<string, Check>
) {
  const [entries, setEntries] = useState<EventLogEntry[]>([])
  const idCounter = useRef(0)
  const initialized = useRef(false)

  const addEntry = useCallback((entry: Omit<EventLogEntry, 'id'>) => {
    idCounter.current++
    setEntries((prev) => {
      const next = [...prev, { ...entry, id: idCounter.current }]
      return next.length > MAX_ENTRIES ? next.slice(-MAX_ENTRIES) : next
    })
  }, [])

  useEffect(() => {
    if (checks.length === 0) return
    // Skip first load — no transitions to detect
    if (!initialized.current) {
      initialized.current = true
      return
    }
    if (previousChecks.size === 0) return

    for (const check of checks) {
      const prev = previousChecks.get(check.UUID)
      if (!prev) continue

      // Status transition
      if (prev.LastResult !== check.LastResult && check.Enabled) {
        addEntry({
          timestamp: new Date(),
          checkName: check.Name,
          checkUUID: check.UUID,
          status: check.LastResult ? 'healthy' : 'unhealthy',
          message: check.Message,
        })
      }

      // Enable/disable transition
      if (prev.Enabled !== check.Enabled) {
        addEntry({
          timestamp: new Date(),
          checkName: check.Name,
          checkUUID: check.UUID,
          status: check.Enabled ? 'enabled' : 'disabled',
        })
      }
    }
  }, [checks, previousChecks, addEntry])

  return { entries }
}
