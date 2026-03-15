import { useState, useEffect, useCallback, useRef } from 'react'
import { WebSocketManager, type Check, type WSMessage } from '@/lib/websocket'

export type WSStatus = 'connected' | 'disconnected' | 'connecting'

export interface CheckStats {
  total: number
  healthy: number
  unhealthy: number
  disabled: number
  silenced: number
}

export interface ProjectGroup {
  name: string
  checks: Check[]
  healthyCount: number
  failingCount: number
}

export function useChecks() {
  const [checks, setChecks] = useState<Map<string, Check>>(new Map())
  const [wsStatus, setWsStatus] = useState<WSStatus>('disconnected')
  const wsRef = useRef<WebSocketManager | null>(null)
  const prevChecksRef = useRef<Map<string, Check>>(new Map())

  const handleMessage = useCallback((msg: WSMessage) => {
    if (msg.type === 'checks') {
      setChecks((prev) => {
        prevChecksRef.current = prev
        const next = new Map<string, Check>()
        for (const check of msg.checks || []) {
          next.set(check.UUID, check)
        }
        return next
      })
    } else if (msg.type === 'update') {
      setChecks((prev) => {
        prevChecksRef.current = prev
        const next = new Map(prev)
        next.set(msg.check.UUID, msg.check)
        return next
      })
    }
  }, [])

  useEffect(() => {
    const ws = new WebSocketManager(handleMessage, setWsStatus)
    wsRef.current = ws
    ws.connect()
    return () => ws.disconnect()
  }, [handleMessage])

  const checksArray = Array.from(checks.values())

  const stats: CheckStats = {
    total: checksArray.length,
    healthy: checksArray.filter((c) => c.Enabled && c.LastResult).length,
    unhealthy: checksArray.filter((c) => c.Enabled && !c.LastResult).length,
    disabled: checksArray.filter((c) => !c.Enabled).length,
    silenced: checksArray.filter((c) => c.IsSilenced).length,
  }

  const getGrouped = useCallback(
    (filtered: Check[]): ProjectGroup[] => {
      const groups = new Map<string, Check[]>()
      for (const check of filtered) {
        const project = check.Project || 'default'
        if (!groups.has(project)) groups.set(project, [])
        groups.get(project)!.push(check)
      }
      return Array.from(groups.entries())
        .map(([name, checks]) => ({
          name,
          checks,
          healthyCount: checks.filter((c) => c.Enabled && c.LastResult).length,
          failingCount: checks.filter((c) => c.Enabled && !c.LastResult).length,
        }))
        .sort((a, b) => {
          // Failing projects first
          if (a.failingCount > 0 && b.failingCount === 0) return -1
          if (b.failingCount > 0 && a.failingCount === 0) return 1
          return a.name.localeCompare(b.name)
        })
    },
    []
  )

  const previousChecks = prevChecksRef.current

  return { checks: checksArray, checksMap: checks, previousChecks, stats, wsStatus, getGrouped }
}
