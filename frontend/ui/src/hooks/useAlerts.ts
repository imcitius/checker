import { useState, useEffect, useCallback, useRef } from 'react'
import { WebSocketManager, type WSMessage } from '@/lib/websocket'
import { api, type AlertEvent, type AlertSilence } from '@/lib/api'
import type { WSStatus } from '@/hooks/useChecks'

const PAGE_SIZE = 50

export function useAlerts() {
  const [alerts, setAlerts] = useState<AlertEvent[]>([])
  const [total, setTotal] = useState(0)
  const [offset, setOffset] = useState(0)
  const [loading, setLoading] = useState(true)
  const [projectFilter, setProjectFilter] = useState('all')
  const [statusFilter, setStatusFilter] = useState('all')
  const [since, setSince] = useState<string | undefined>(undefined)
  const [until, setUntil] = useState<string | undefined>(undefined)

  const [silences, setSilences] = useState<AlertSilence[]>([])
  const [silencesLoading, setSilencesLoading] = useState(true)

  const [wsStatus, setWsStatus] = useState<WSStatus>('disconnected')
  const wsRef = useRef<WebSocketManager | null>(null)

  // Track recently added alert IDs for highlight animation
  const [recentAlertIds, setRecentAlertIds] = useState<Set<number>>(new Set())

  const fetchAlerts = useCallback(async (newOffset?: number) => {
    setLoading(true)
    try {
      const effectiveOffset = newOffset ?? offset
      const result = await api.getAlerts({
        limit: PAGE_SIZE,
        offset: effectiveOffset,
        project: projectFilter !== 'all' ? projectFilter : undefined,
        status: statusFilter !== 'all' ? statusFilter : undefined,
        since,
        until,
      })
      setAlerts(result.alerts || [])
      setTotal(result.total)
    } catch (err) {
      console.error('Failed to fetch alerts:', err)
    } finally {
      setLoading(false)
    }
  }, [offset, projectFilter, statusFilter, since, until])

  const fetchSilences = useCallback(async () => {
    setSilencesLoading(true)
    try {
      const result = await api.getSilences()
      setSilences(result.silences || [])
    } catch (err) {
      console.error('Failed to fetch silences:', err)
    } finally {
      setSilencesLoading(false)
    }
  }, [])

  // WebSocket handler for real-time alert updates
  const handleMessage = useCallback((msg: WSMessage) => {
    if (msg.type === 'alert_new') {
      const newAlert: AlertEvent = {
        ID: msg.alert.ID,
        CheckUUID: msg.alert.CheckUUID,
        CheckName: msg.alert.CheckName,
        Project: msg.alert.Project,
        GroupName: msg.alert.GroupName,
        CheckType: msg.alert.CheckType,
        Message: msg.alert.Message,
        AlertType: msg.alert.AlertType,
        CreatedAt: msg.alert.CreatedAt,
        ResolvedAt: msg.alert.ResolvedAt,
        IsResolved: msg.alert.IsResolved,
      }
      setAlerts((prev) => [newAlert, ...prev])
      setTotal((prev) => prev + 1)

      // Highlight the new alert briefly
      setRecentAlertIds((prev) => {
        const next = new Set(prev)
        next.add(newAlert.ID)
        return next
      })
      setTimeout(() => {
        setRecentAlertIds((prev) => {
          const next = new Set(prev)
          next.delete(newAlert.ID)
          return next
        })
      }, 3000)
    } else if (msg.type === 'alert_resolved') {
      setAlerts((prev) =>
        prev.map((a) =>
          a.CheckUUID === msg.check_uuid && !a.IsResolved
            ? { ...a, IsResolved: true, ResolvedAt: new Date().toISOString() }
            : a
        )
      )
    }
  }, [])

  // Connect WebSocket
  useEffect(() => {
    const ws = new WebSocketManager(handleMessage, setWsStatus)
    wsRef.current = ws
    ws.connect()
    return () => ws.disconnect()
  }, [handleMessage])

  // Fetch data when filters change
  useEffect(() => {
    setOffset(0)
    fetchAlerts(0)
  }, [projectFilter, statusFilter, since, until]) // eslint-disable-line react-hooks/exhaustive-deps

  // Fetch alerts when offset changes
  useEffect(() => {
    fetchAlerts()
  }, [offset]) // eslint-disable-line react-hooks/exhaustive-deps

  // Initial fetch silences
  useEffect(() => {
    fetchSilences()
  }, [fetchSilences])

  const goToPage = useCallback((page: number) => {
    setOffset(page * PAGE_SIZE)
  }, [])

  const currentPage = Math.floor(offset / PAGE_SIZE)
  const totalPages = Math.ceil(total / PAGE_SIZE)

  return {
    alerts,
    total,
    loading,
    projectFilter,
    setProjectFilter,
    statusFilter,
    setStatusFilter,
    since,
    setSince,
    until,
    setUntil,
    silences,
    silencesLoading,
    wsStatus,
    recentAlertIds,
    currentPage,
    totalPages,
    goToPage,
    fetchAlerts,
    fetchSilences,
    pageSize: PAGE_SIZE,
  }
}
