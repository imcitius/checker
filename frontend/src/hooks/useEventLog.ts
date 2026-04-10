import { useState, useCallback, useRef, useEffect } from 'react'
import type { Check } from '@/lib/websocket'
import { api } from '@/lib/api'

export interface EventLogEntry {
  id: number
  timestamp: Date
  checkName: string
  checkUUID: string
  project?: string
  previousStatus?: string
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
  const historicalLoaded = useRef(false)

  const addEntry = useCallback((entry: Omit<EventLogEntry, 'id'>) => {
    idCounter.current++
    setEntries((prev) => {
      const next = [...prev, { ...entry, id: idCounter.current }]
      return next.length > MAX_ENTRIES ? next.slice(-MAX_ENTRIES) : next
    })
  }, [])

  // Load recent alert history from API on mount
  useEffect(() => {
    if (historicalLoaded.current) return
    historicalLoaded.current = true

    api.getAlerts({ limit: 20 }).then((response) => {
      if (!response.alerts || response.alerts.length === 0) return

      // Sort oldest-first so newest ends up at the bottom of the log
      const sorted = [...response.alerts].sort(
        (a, b) => new Date(a.CreatedAt).getTime() - new Date(b.CreatedAt).getTime()
      )

      setEntries((prev) => {
        const historicalEntries: EventLogEntry[] = sorted.map((alert) => {
          idCounter.current++
          const isRecovery = alert.IsResolved
          return {
            id: idCounter.current,
            timestamp: new Date(alert.CreatedAt),
            checkName: alert.CheckName,
            checkUUID: alert.CheckUUID,
            project: alert.Project || undefined,
            previousStatus: isRecovery ? 'unhealthy' : 'healthy',
            status: isRecovery ? 'healthy' : 'unhealthy',
            message: alert.Message || undefined,
          }
        })

        // Prepend historical entries before any live entries already in state
        const combined = [...historicalEntries, ...prev]
        return combined.length > MAX_ENTRIES ? combined.slice(-MAX_ENTRIES) : combined
      })
    }).catch(() => {
      // Silently ignore errors — historical data is best-effort
    })
  }, [])

  // Detect real-time state transitions from WebSocket updates
  useEffect(() => {
    if (checks.length === 0) return
    // Skip first load — no transitions to detect yet
    if (!initialized.current) {
      initialized.current = true
      return
    }
    if (previousChecks.size === 0) return

    for (const check of checks) {
      const prev = previousChecks.get(check.UUID)
      if (!prev) continue

      // Status transition (healthy ↔ unhealthy)
      if (prev.LastResult !== check.LastResult && check.Enabled) {
        addEntry({
          timestamp: new Date(),
          checkName: check.Name,
          checkUUID: check.UUID,
          project: check.Project || undefined,
          previousStatus: prev.LastResult ? 'healthy' : 'unhealthy',
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
          project: check.Project || undefined,
          previousStatus: prev.Enabled ? 'enabled' : 'disabled',
          status: check.Enabled ? 'enabled' : 'disabled',
        })
      }
    }
  }, [checks, previousChecks, addEntry])

  return { entries }
}
