import { memo, type ReactNode } from 'react'
import type { Check } from '@/lib/websocket'
import { StatusDot } from '@/components/StatusDot'
import { Badge } from '@/components/ui/badge'
import { ChevronDown, ChevronRight } from 'lucide-react'
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip'
import { relativeTime } from '@/lib/utils'
import { cn } from '@/lib/utils'

/** Render function type for extra badges displayed in a CheckRow */
export type ExtraBadgeRenderer = (check: Check) => ReactNode

export interface CheckRowProps {
  check: Check
  isSelected: boolean
  isExpanded: boolean
  onSelect: () => void

  // --- Customization props ---
  /** Extra badges rendered after the type badge */
  extraBadges?: ExtraBadgeRenderer
}

function getTypeBadgeVariant(type: string): 'info' | 'database' | 'warning' | 'secondary' {
  if (type.includes('pgsql') || type.includes('mysql')) return 'database'
  if (type === 'http') return 'info'
  if (type === 'icmp') return 'warning'
  return 'secondary'
}

export const CheckRow = memo(function CheckRow({
  check,
  isSelected,
  isExpanded,
  onSelect,
  extraBadges,
}: CheckRowProps) {
  const hasPartialSilence = !check.IsSilenced && check.SilencedChannels && check.SilencedChannels.length > 0

  const statusText = !check.Enabled
    ? 'Disabled'
    : check.LastResult
      ? 'Healthy'
      : 'FAILING'

  return (
    <div
      id={`check-${check.UUID}`}
      className={cn(
        'flex items-center gap-3 px-3 py-1.5 cursor-pointer transition-colors text-sm',
        'hover:bg-muted/50',
        isSelected && 'bg-muted ring-1 ring-info/30',
        !check.Enabled && 'opacity-60'
      )}
      onClick={onSelect}
    >
      {/* Status dot */}
      <StatusDot
        healthy={check.LastResult}
        enabled={check.Enabled}
        silenced={check.IsSilenced}
        partiallySilenced={hasPartialSilence}
      />

      {/* Name */}
      <Tooltip>
        <TooltipTrigger asChild>
          <span className="font-medium truncate min-w-0 w-[180px] shrink-0">{check.Name}</span>
        </TooltipTrigger>
        <TooltipContent>
          <span className="font-mono text-xs">{check.UUID}</span>
        </TooltipContent>
      </Tooltip>

      {/* Type badge */}
      <Badge variant={getTypeBadgeVariant(check.CheckType)} className="shrink-0 text-[10px] px-1.5 py-0">
        {check.CheckType}
      </Badge>

      {/* Extra badges */}
      {extraBadges && extraBadges(check)}

      {/* Status text */}
      <span
        className={cn(
          'font-mono text-xs w-[70px] shrink-0',
          !check.Enabled
            ? 'text-disabled'
            : check.LastResult
              ? 'text-healthy'
              : 'text-unhealthy font-semibold'
        )}
      >
        {statusText}
      </span>

      {/* Host / URL */}
      <Tooltip>
        <TooltipTrigger asChild>
          <span
            className="font-mono text-xs text-muted-foreground break-all min-w-0 flex-1 cursor-pointer hover:text-foreground transition-colors"
            onDoubleClick={(e) => {
              e.stopPropagation()
              const target = check.URL || check.Host || ''
              if (target) {
                navigator.clipboard.writeText(target)
              }
            }}
          >
            {check.URL || check.Host || '\u2014'}
          </span>
        </TooltipTrigger>
        <TooltipContent>Double-click to copy</TooltipContent>
      </Tooltip>

      {/* Frequency */}
      <span className="font-mono text-xs text-muted-foreground w-[40px] shrink-0 text-right">
        {check.Periodicity || '\u2014'}
      </span>

      {/* Last run */}
      <Tooltip>
        <TooltipTrigger asChild>
          <span className="font-mono text-xs text-muted-foreground w-[60px] shrink-0 text-right">
            {relativeTime(check.LastExec)}
          </span>
        </TooltipTrigger>
        <TooltipContent>{check.LastExec}</TooltipContent>
      </Tooltip>

      {/* Expand indicator */}
      {isExpanded ? (
        <ChevronDown className="h-4 w-4 text-muted-foreground shrink-0" />
      ) : (
        <ChevronRight className="h-4 w-4 text-muted-foreground shrink-0" />
      )}
    </div>
  )
})
