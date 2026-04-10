import { useEffect, useState } from 'react'
import type { CheckStats } from '@/hooks/useChecks'
import type { Check } from '@/lib/websocket'
import { Popover, PopoverTrigger, PopoverContent } from '@/components/ui/popover'
import { relativeTime } from '@/lib/utils'
import { AlertCircle } from 'lucide-react'

interface MetricsRowProps {
  stats: CheckStats
  failingChecks?: Check[]
  onSelectCheck?: (uuid: string) => void
}

function AnimatedCount({ value, delay }: { value: number; delay: number }) {
  const [display, setDisplay] = useState(0)
  const [animate, setAnimate] = useState(false)

  useEffect(() => {
    const timer = setTimeout(() => {
      setDisplay(value)
      setAnimate(true)
    }, delay)
    return () => clearTimeout(timer)
  }, [value, delay])

  useEffect(() => {
    if (!animate) return
    const t = setTimeout(() => setAnimate(false), 400)
    return () => clearTimeout(t)
  }, [animate])

  return (
    <span className={animate ? 'animate-count-up' : ''}>
      {display}
    </span>
  )
}

export function MetricsRow({ stats, failingChecks = [], onSelectCheck }: MetricsRowProps) {
  const hasFailures = stats.unhealthy > 0

  const staticCards = [
    { label: 'Total', value: stats.total, color: 'border-info/40', glow: '' },
    { label: 'Healthy', value: stats.healthy, color: 'border-healthy/40', glow: 'glow-healthy' },
    { label: 'Disabled', value: stats.disabled, color: 'border-disabled/40', glow: '' },
  ]

  return (
    <div className="grid grid-cols-2 sm:grid-cols-4 gap-3">
      {/* Total */}
      <div
        className={`rounded-lg border bg-card p-3 text-center ${staticCards[0].color} ${staticCards[0].glow}`}
      >
        <div className="text-2xl font-bold font-mono text-foreground">
          <AnimatedCount value={staticCards[0].value} delay={0} />
        </div>
        <div className="text-xs text-muted-foreground mt-0.5">{staticCards[0].label}</div>
      </div>

      {/* Healthy */}
      <div
        className={`rounded-lg border bg-card p-3 text-center ${staticCards[1].color} ${staticCards[1].glow}`}
      >
        <div className="text-2xl font-bold font-mono text-foreground">
          <AnimatedCount value={staticCards[1].value} delay={80} />
        </div>
        <div className="text-xs text-muted-foreground mt-0.5">{staticCards[1].label}</div>
      </div>

      {/* Failing — clickable popover when > 0 */}
      {hasFailures ? (
        <Popover>
          <PopoverTrigger asChild>
            <div
              className={`rounded-lg border bg-card p-3 text-center border-unhealthy/40 glow-unhealthy cursor-pointer hover:bg-muted/30 transition-colors`}
              role="button"
              tabIndex={0}
              aria-label={`${stats.unhealthy} failing checks — click to view`}
              onKeyDown={(e) => {
                if (e.key === 'Enter' || e.key === ' ') {
                  e.currentTarget.click()
                }
              }}
            >
              <div className="text-2xl font-bold font-mono text-foreground">
                <AnimatedCount value={stats.unhealthy} delay={160} />
              </div>
              <div className="text-xs text-muted-foreground mt-0.5">Failing</div>
            </div>
          </PopoverTrigger>
          <PopoverContent
            className="w-80 p-0 max-h-96 overflow-hidden flex flex-col"
            align="start"
          >
            <div className="flex items-center gap-2 px-3 py-2 border-b bg-muted/30">
              <AlertCircle className="h-4 w-4 text-destructive shrink-0" />
              <span className="text-sm font-medium">Failing Checks</span>
              <span className="ml-auto text-xs text-muted-foreground">{failingChecks.length}</span>
            </div>
            <div className="overflow-y-auto flex-1">
              {failingChecks.map((check) => (
                <button
                  key={check.UUID}
                  className="w-full text-left px-3 py-2 hover:bg-muted/50 transition-colors border-b last:border-b-0 focus:outline-none focus:bg-muted/50"
                  onClick={() => onSelectCheck?.(check.UUID)}
                >
                  <div className="flex items-start justify-between gap-2">
                    <div className="min-w-0 flex-1">
                      <div className="text-sm font-medium text-foreground truncate">{check.Name}</div>
                      {check.Project && (
                        <div className="text-xs text-muted-foreground truncate">{check.Project}</div>
                      )}
                      {check.Message && (
                        <div className="text-xs text-destructive mt-0.5 line-clamp-2">{check.Message}</div>
                      )}
                    </div>
                    <div className="text-xs text-muted-foreground shrink-0 mt-0.5">
                      {relativeTime(check.LastExec)}
                    </div>
                  </div>
                </button>
              ))}
            </div>
          </PopoverContent>
        </Popover>
      ) : (
        <div
          className={`rounded-lg border bg-card p-3 text-center border-unhealthy/40`}
        >
          <div className="text-2xl font-bold font-mono text-foreground">
            <AnimatedCount value={stats.unhealthy} delay={160} />
          </div>
          <div className="text-xs text-muted-foreground mt-0.5">Failing</div>
        </div>
      )}

      {/* Disabled */}
      <div
        className={`rounded-lg border bg-card p-3 text-center ${staticCards[2].color} ${staticCards[2].glow}`}
      >
        <div className="text-2xl font-bold font-mono text-foreground">
          <AnimatedCount value={staticCards[2].value} delay={240} />
        </div>
        <div className="text-xs text-muted-foreground mt-0.5">{staticCards[2].label}</div>
      </div>
    </div>
  )
}
