import { memo } from 'react'
import type { Check } from '@/lib/websocket'
import { StatusDot } from '@/components/StatusDot'
import { Badge } from '@/components/ui/badge'
import { Switch } from '@/components/ui/switch'
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip'
import { relativeTime } from '@/lib/utils'
import { cn } from '@/lib/utils'

interface CheckRowProps {
  check: Check
  isSelected: boolean
  isExpanded: boolean
  onSelect: () => void
  onToggle: (uuid: string, enabled: boolean) => void
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
  onToggle,
}: CheckRowProps) {
  const statusText = !check.Enabled
    ? 'Disabled'
    : check.LastResult
      ? 'Healthy'
      : 'FAILING'

  return (
    <div
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
          <span className="font-mono text-xs text-muted-foreground truncate min-w-0 flex-1">
            {check.URL || check.Host || '—'}
          </span>
        </TooltipTrigger>
        <TooltipContent>{check.URL || check.Host || 'No target'}</TooltipContent>
      </Tooltip>

      {/* Frequency */}
      <span className="font-mono text-xs text-muted-foreground w-[40px] shrink-0 text-right">
        {check.Periodicity || '—'}
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

      {/* Toggle */}
      <div className="shrink-0" onClick={(e) => e.stopPropagation()}>
        <Switch
          checked={check.Enabled}
          onCheckedChange={(checked) => onToggle(check.UUID, checked)}
          className="scale-75"
        />
      </div>

      {/* Expand indicator */}
      <span className={cn('text-muted-foreground text-xs transition-transform shrink-0', isExpanded && 'rotate-180')}>
        ▾
      </span>
    </div>
  )
})
