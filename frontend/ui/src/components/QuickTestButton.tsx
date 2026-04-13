import { memo, useState } from 'react'
import { Play, Loader2, Check, X, ChevronDown, ChevronUp } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip'
import { cn } from '@/lib/utils'
import type { QuickTestResult } from '@/hooks/useCheckQuickTest'

interface QuickTestButtonProps {
  uuid: string
  state: QuickTestResult
  disabled: boolean
  cooldownLabel: string | null
  onTest: (uuid: string) => void
  /** Compact mode for health map tiles */
  compact?: boolean
}

/** Summarise multi-region results into a single line. */
function summariseResults(state: QuickTestResult): {
  text: string
  variant: 'success' | 'error' | 'partial'
} | null {
  if (state.error) return { text: state.error, variant: 'error' }
  if (!state.results) return null
  const total = state.results.length
  const ok = state.results.filter((r) => r.healthy).length
  if (total === 1) {
    const r = state.results[0]
    return r.healthy
      ? { text: `OK (${r.duration_ms}ms)`, variant: 'success' }
      : { text: `Failed: ${r.message}`, variant: 'error' }
  }
  if (ok === total) return { text: `${ok}/${total} OK`, variant: 'success' }
  if (ok === 0) return { text: `0/${total} OK`, variant: 'error' }
  return { text: `${ok}/${total} OK, ${total - ok} failed`, variant: 'partial' }
}

/** Inline test button + result for list-view rows. */
export const QuickTestInline = memo(function QuickTestInline({
  uuid,
  state,
  disabled,
  cooldownLabel,
  onTest,
}: QuickTestButtonProps) {
  const [expanded, setExpanded] = useState(false)
  const summary = summariseResults(state)

  return (
    <span className="inline-flex items-center gap-1.5 shrink-0" onClick={(e) => e.stopPropagation()}>
      <Tooltip>
        <TooltipTrigger asChild>
          <Button
            variant="ghost"
            size="sm"
            className={cn(
              'h-5 w-5 p-0 rounded-sm',
              disabled && 'opacity-40'
            )}
            disabled={disabled}
            onClick={(e) => {
              e.stopPropagation()
              onTest(uuid)
            }}
          >
            {state.loading ? (
              <Loader2 className="h-3 w-3 animate-spin text-muted-foreground" />
            ) : cooldownLabel ? (
              <span className="text-[9px] text-muted-foreground font-mono">{cooldownLabel}</span>
            ) : (
              <Play className="h-3 w-3 text-muted-foreground" />
            )}
          </Button>
        </TooltipTrigger>
        <TooltipContent side="left">
          {cooldownLabel ? `Cooldown ${cooldownLabel}` : 'Test this check'}
        </TooltipContent>
      </Tooltip>

      {summary && (
        <span
          className={cn(
            'text-[10px] font-mono inline-flex items-center gap-0.5',
            summary.variant === 'success' && 'text-healthy',
            summary.variant === 'error' && 'text-unhealthy',
            summary.variant === 'partial' && 'text-warning'
          )}
        >
          {summary.variant === 'success' && <Check className="h-2.5 w-2.5" />}
          {summary.variant === 'error' && <X className="h-2.5 w-2.5" />}
          {summary.variant === 'partial' && <span className="text-[10px]">⚠</span>}
          <span className="max-w-[120px] truncate">{summary.text}</span>
          {state.results && state.results.length > 1 && (
            <button
              className="ml-0.5 text-muted-foreground hover:text-foreground"
              onClick={(e) => {
                e.stopPropagation()
                setExpanded((p) => !p)
              }}
            >
              {expanded ? <ChevronUp className="h-2.5 w-2.5" /> : <ChevronDown className="h-2.5 w-2.5" />}
            </button>
          )}
        </span>
      )}

      {expanded && state.results && state.results.length > 1 && (
        <span className="absolute right-0 top-full z-20 mt-1 rounded border bg-popover p-2 shadow-md text-[10px] font-mono space-y-0.5 min-w-[180px]">
          {state.results.map((r) => (
            <span
              key={r.region}
              className={cn(
                'block',
                r.healthy ? 'text-healthy' : 'text-unhealthy'
              )}
            >
              {r.healthy ? '✅' : '❌'} {r.region}: {r.message} ({r.duration_ms}ms)
            </span>
          ))}
        </span>
      )}
    </span>
  )
})

/** Compact test button for health map tiles (icon only, result in tooltip). */
export const QuickTestTile = memo(function QuickTestTile({
  uuid,
  state,
  disabled,
  cooldownLabel,
  onTest,
}: QuickTestButtonProps) {
  const summary = summariseResults(state)

  const tooltipContent = state.loading
    ? 'Testing…'
    : summary
      ? state.results && state.results.length > 1
        ? state.results.map((r) => `${r.healthy ? '✅' : '❌'} ${r.region}: ${r.message} (${r.duration_ms}ms)`).join('\n')
        : summary.text
      : cooldownLabel
        ? `Cooldown ${cooldownLabel}`
        : 'Test'

  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <button
          className={cn(
            'absolute inset-0 flex items-center justify-center',
            'bg-background/70 rounded-sm opacity-0 group-hover/tile:opacity-100 transition-opacity',
            disabled && 'cursor-not-allowed',
            summary && 'opacity-100 bg-background/50'
          )}
          disabled={disabled}
          onClick={(e) => {
            e.stopPropagation()
            if (!disabled) onTest(uuid)
          }}
        >
          {state.loading ? (
            <Loader2 className="h-3 w-3 animate-spin text-muted-foreground" />
          ) : summary ? (
            <span
              className={cn(
                'text-[9px] font-bold',
                summary.variant === 'success' && 'text-healthy',
                summary.variant === 'error' && 'text-unhealthy',
                summary.variant === 'partial' && 'text-warning'
              )}
            >
              {summary.variant === 'success' ? '✓' : summary.variant === 'error' ? '✗' : '⚠'}
            </span>
          ) : cooldownLabel ? (
            <span className="text-[8px] text-muted-foreground font-mono">{cooldownLabel}</span>
          ) : (
            <Play className="h-3 w-3 text-muted-foreground" />
          )}
        </button>
      </TooltipTrigger>
      <TooltipContent side="top" className="whitespace-pre-line text-[10px] max-w-[250px]">
        {tooltipContent}
      </TooltipContent>
    </Tooltip>
  )
})
