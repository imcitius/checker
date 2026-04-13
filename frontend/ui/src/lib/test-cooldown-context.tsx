import { createContext, useCallback, useContext, useEffect, useRef, useState, type ReactNode } from 'react'

interface TestCooldownContextValue {
  /** Seconds remaining on the global (429) cooldown. 0 = no cooldown. */
  globalCooldownRemaining: number
  /** Start a global cooldown (called on 429 rate-limit response). */
  startGlobalCooldown: (retryAfterSeconds: number) => void
}

const TestCooldownContext = createContext<TestCooldownContextValue | null>(null)

/**
 * Provides shared global test-cooldown state.
 * Wrap your app (or the section containing both Dashboard and CheckEditDrawer)
 * so that a 429 rate-limit in one component disables test buttons everywhere.
 */
export function TestCooldownProvider({ children }: { children: ReactNode }) {
  const [remaining, setRemaining] = useState(0)
  const intervalRef = useRef<ReturnType<typeof setInterval> | null>(null)

  const startGlobalCooldown = useCallback((retryAfterSeconds: number) => {
    const until = Date.now() + retryAfterSeconds * 1000
    setRemaining(retryAfterSeconds)
    if (intervalRef.current) clearInterval(intervalRef.current)
    intervalRef.current = setInterval(() => {
      const r = Math.ceil((until - Date.now()) / 1000)
      if (r <= 0) {
        setRemaining(0)
        if (intervalRef.current) {
          clearInterval(intervalRef.current)
          intervalRef.current = null
        }
      } else {
        setRemaining(r)
      }
    }, 1000)
  }, [])

  useEffect(() => {
    return () => {
      if (intervalRef.current) clearInterval(intervalRef.current)
    }
  }, [])

  return (
    <TestCooldownContext.Provider value={{ globalCooldownRemaining: remaining, startGlobalCooldown }}>
      {children}
    </TestCooldownContext.Provider>
  )
}

/**
 * Consume the shared test-cooldown context.
 * Falls back to a local no-op if no provider is present (backwards compatible).
 */
export function useTestCooldown(): TestCooldownContextValue {
  const ctx = useContext(TestCooldownContext)
  if (ctx) return ctx
  // Fallback: no provider — return inert values (won't share state).
  // This keeps CheckEditDrawer working in standalone mode.
  return { globalCooldownRemaining: 0, startGlobalCooldown: () => {} }
}
