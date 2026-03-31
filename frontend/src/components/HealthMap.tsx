import { memo } from 'react'
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip'
import type { Check } from '@/lib/websocket'
import type { ProjectGroup } from '@/hooks/useChecks'
import { cn } from '@/lib/utils'

interface HealthMapProps {
  groups: ProjectGroup[]
  onSelectCheck: (uuid: string) => void
}

const HealthTile = memo(function HealthTile({
  check,
  onSelect,
}: {
  check: Check
  onSelect: () => void
}) {
  const hasPartialSilence = !check.IsSilenced && check.SilencedChannels && check.SilencedChannels.length > 0

  const status = !check.Enabled
    ? 'disabled'
    : check.IsSilenced
      ? 'silenced'
      : hasPartialSilence
        ? 'partial'
        : check.LastResult
          ? 'healthy'
          : 'unhealthy'

  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <button
          className={cn(
            'h-10 w-10 sm:h-8 sm:w-8 rounded-sm transition-all cursor-pointer border',
            'hover:scale-110 hover:z-10',
            status === 'healthy' && 'bg-healthy/80 border-healthy/40 hover:bg-healthy',
            status === 'unhealthy' && 'bg-unhealthy/80 border-unhealthy/40 hover:bg-unhealthy animate-pulse-unhealthy',
            status === 'disabled' && 'bg-disabled/30 border-disabled/20 hover:bg-disabled/50',
            status === 'silenced' && 'bg-warning/40 border-warning/30 hover:bg-warning/60',
            status === 'partial' && 'bg-warning/25 border-warning/20 hover:bg-warning/40'
          )}
          onClick={onSelect}
        />
      </TooltipTrigger>
      <TooltipContent side="top" className="max-w-[300px]">
        <div className="space-y-1">
          <div className="font-medium text-xs">{check.Name}</div>
          <div className="text-[10px] text-muted-foreground">
            {check.CheckType} · {check.URL || check.Host || '—'}
          </div>
          <div
            className={cn(
              'text-[10px] font-mono',
              status === 'healthy' && 'text-healthy',
              status === 'unhealthy' && 'text-unhealthy',
              status === 'disabled' && 'text-disabled',
              (status === 'silenced' || status === 'partial') && 'text-warning'
            )}
          >
            {status === 'healthy' ? 'Healthy' : status === 'unhealthy' ? 'FAILING' : status === 'disabled' ? 'Disabled' : status === 'partial' ? `Partially silenced (${check.SilencedChannels?.join(', ')})` : 'Silenced'}
          </div>
          {check.Message && status === 'unhealthy' && (
            <div className="text-[10px] text-unhealthy/80 truncate">{check.Message}</div>
          )}
        </div>
      </TooltipContent>
    </Tooltip>
  )
})

export function HealthMap({ groups, onSelectCheck }: HealthMapProps) {
  if (groups.length === 0) {
    return (
      <div className="text-center py-12 text-muted-foreground text-sm">
        No checks found
      </div>
    )
  }

  return (
    <div className="space-y-4">
      {groups.map((group) => (
        <div key={group.name} className="rounded-lg border bg-card p-3">
          <div className="flex items-center gap-2 mb-2">
            <span className="font-semibold text-sm text-foreground">{group.name}</span>
            <span className="text-xs text-muted-foreground">
              {group.checks.length} check{group.checks.length !== 1 ? 's' : ''}
            </span>
            {group.failingCount > 0 && (
              <span className="text-[10px] text-unhealthy font-medium">
                {group.failingCount} failing
              </span>
            )}
          </div>
          <div className="grid grid-cols-[repeat(auto-fill,2.75rem)] sm:grid-cols-[repeat(auto-fill,2rem)] gap-1.5">
            {group.checks.map((check) => (
              <HealthTile
                key={check.UUID}
                check={check}
                onSelect={() => onSelectCheck(check.UUID)}
              />
            ))}
          </div>
        </div>
      ))}
    </div>
  )
}
