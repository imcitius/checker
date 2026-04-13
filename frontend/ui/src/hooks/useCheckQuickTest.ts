import { useState, useCallback, useRef, useEffect } from 'react'
import { api, type TestRemoteLocationResult } from '@/lib/api'
import { useTestCooldown } from '@/lib/test-cooldown-context'

const PER_CHECK_COOLDOWN_SECONDS = 10

export interface QuickTestResult {
  loading: boolean
  results: TestRemoteLocationResult[] | null
  error: string | null
  cooldown: number // seconds remaining on per-check cooldown
}

/**
 * Hook for quick-testing checks from the Dashboard.
 * Fetches the full CheckDefinition, calls test-remote, shows aggregated results.
 * Shares the global 429 cooldown with CheckEditDrawer via TestCooldownProvider.
 */
export function useCheckQuickTest() {
  const { globalCooldownRemaining, startGlobalCooldown } = useTestCooldown()

  // Per-check state keyed by UUID
  const [results, setResults] = useState<Record<string, QuickTestResult>>({})
  const cooldownIntervalsRef = useRef<Record<string, ReturnType<typeof setInterval>>>({})

  // Cleanup on unmount
  useEffect(() => {
    return () => {
      Object.values(cooldownIntervalsRef.current).forEach(clearInterval)
    }
  }, [])

  const getCheckState = useCallback(
    (uuid: string): QuickTestResult =>
      results[uuid] || { loading: false, results: null, error: null, cooldown: 0 },
    [results]
  )

  const runTest = useCallback(
    async (uuid: string) => {
      // Set loading
      setResults((prev) => ({
        ...prev,
        [uuid]: { loading: true, results: null, error: null, cooldown: 0 },
      }))

      try {
        // Fetch full check definition to get target_regions etc.
        const checkDef = await api.getCheck(uuid)
        // Call test-remote (works for both platform and on-premises regions)
        const response = await api.testCheckRemote(checkDef)
        setResults((prev) => ({
          ...prev,
          [uuid]: { loading: false, results: response.results, error: null, cooldown: PER_CHECK_COOLDOWN_SECONDS },
        }))
      } catch (err) {
        let message = err instanceof Error ? err.message : 'Unknown error'
        if (err instanceof Error) {
          try {
            const parsed = JSON.parse(err.message)
            if (parsed.retry_after) {
              const retryAfter = Number(parsed.retry_after)
              message = `Rate limited — try again in ${parsed.retry_after}s`
              if (retryAfter > 0) startGlobalCooldown(retryAfter)
            } else if (parsed.error) {
              message = parsed.error
            }
          } catch {
            // not JSON, use raw
          }
        }
        setResults((prev) => ({
          ...prev,
          [uuid]: { loading: false, results: null, error: message, cooldown: PER_CHECK_COOLDOWN_SECONDS },
        }))
      }

      // Start per-check cooldown countdown
      if (cooldownIntervalsRef.current[uuid]) {
        clearInterval(cooldownIntervalsRef.current[uuid])
      }
      cooldownIntervalsRef.current[uuid] = setInterval(() => {
        setResults((prev) => {
          const cur = prev[uuid]
          if (!cur) return prev
          const remaining = cur.cooldown - 1
          if (remaining <= 0) {
            clearInterval(cooldownIntervalsRef.current[uuid])
            delete cooldownIntervalsRef.current[uuid]
            return { ...prev, [uuid]: { ...cur, cooldown: 0 } }
          }
          return { ...prev, [uuid]: { ...cur, cooldown: remaining } }
        })
      }, 1000)
    },
    [startGlobalCooldown]
  )

  const isDisabled = useCallback(
    (uuid: string): boolean => {
      if (globalCooldownRemaining > 0) return true
      const state = results[uuid]
      return state ? state.loading || state.cooldown > 0 : false
    },
    [globalCooldownRemaining, results]
  )

  const getCooldownLabel = useCallback(
    (uuid: string): string | null => {
      if (globalCooldownRemaining > 0) return `${globalCooldownRemaining}s`
      const state = results[uuid]
      if (state?.cooldown && state.cooldown > 0) return `${state.cooldown}s`
      return null
    },
    [globalCooldownRemaining, results]
  )

  return { getCheckState, runTest, isDisabled, getCooldownLabel, globalCooldownRemaining }
}
