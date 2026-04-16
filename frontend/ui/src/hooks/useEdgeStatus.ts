import { useState, useEffect, useCallback, useRef } from 'react'
import { api } from '@/lib/api'

/** Set of on-premises probe region names that are currently disconnected or stale. */
export function useEdgeStatus() {
  const [offlineRegions, setOfflineRegions] = useState<Set<string>>(new Set())
  const mountedRef = useRef(true)

  const fetchStatus = useCallback(async () => {
    try {
      const resp = await api.getEdgeInstances()
      if (!resp || !mountedRef.current) return
      const offline = new Set<string>()
      for (const inst of resp.edge_instances) {
        if (inst.status !== 'connected') {
          offline.add(inst.region)
        }
      }
      // Remove regions that also have a connected instance
      for (const inst of resp.edge_instances) {
        if (inst.status === 'connected') {
          offline.delete(inst.region)
        }
      }
      setOfflineRegions(offline)
    } catch {
      // API not available (e.g. standalone mode) — no edge instances
    }
  }, [])

  useEffect(() => {
    mountedRef.current = true
    fetchStatus()
    // Refresh every 30s to pick up status changes
    const interval = setInterval(fetchStatus, 30_000)
    return () => {
      mountedRef.current = false
      clearInterval(interval)
    }
  }, [fetchStatus])

  return { offlineRegions }
}
