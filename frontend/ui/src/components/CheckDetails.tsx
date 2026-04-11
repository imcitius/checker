import { useState, useEffect, type ReactNode } from 'react'
import type { Check } from '@/lib/websocket'
import { api, type RegionResult } from '@/lib/api'
import { Badge } from '@/components/ui/badge'
import { Separator } from '@/components/ui/separator'
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip'
import { relativeTime } from '@/lib/utils'

/** Render function type for extra badges displayed in CheckDetails */
export type ExtraBadgeRenderer = (check: Check) => ReactNode

export interface CheckDetailsProps {
  check: Check

  // --- Customization props ---
  /** Extra badges rendered at the bottom of the details panel */
  extraBadges?: ExtraBadgeRenderer
}

export function CheckDetails({ check, extraBadges }: CheckDetailsProps) {
  const [regions, setRegions] = useState<RegionResult[]>([])

  useEffect(() => {
    api.getCheckRegions(check.UUID).then(setRegions).catch(() => setRegions([]))
  }, [check.UUID, check.LastExec])

  return (
    <div className="px-6 py-3 bg-muted/30 border-t border-border/50 animate-slide-in">
      <div className="grid grid-cols-2 md:grid-cols-4 gap-3 text-xs">
        <div>
          <span className="text-muted-foreground">UUID</span>
          <div className="font-mono text-foreground mt-0.5 truncate">{check.UUID}</div>
        </div>
        <div>
          <span className="text-muted-foreground">Project</span>
          <div className="text-foreground mt-0.5">{check.Project}</div>
        </div>
        <div>
          <span className="text-muted-foreground">Group</span>
          <div className="text-foreground mt-0.5">{check.Healthcheck}</div>
        </div>
        <div>
          <span className="text-muted-foreground">Type</span>
          <div className="mt-0.5">
            <Badge variant="secondary" className="text-[10px]">{check.CheckType}</Badge>
          </div>
        </div>
        <div>
          <span className="text-muted-foreground">Target</span>
          <div className="font-mono text-foreground mt-0.5 truncate">{check.URL || check.Host || '\u2014'}</div>
        </div>
        <div>
          <span className="text-muted-foreground">Frequency</span>
          <div className="font-mono text-foreground mt-0.5">{check.Periodicity || '\u2014'}</div>
        </div>
        <div>
          <span className="text-muted-foreground">Last Run</span>
          <div className="font-mono text-foreground mt-0.5">{check.LastExec}</div>
        </div>
        <div>
          <span className="text-muted-foreground">Last Alert</span>
          <div className="font-mono text-foreground mt-0.5">{check.LastPing}</div>
        </div>
      </div>

      {check.Message && (
        <>
          <Separator className="my-2" />
          <div className="text-xs">
            <span className="text-muted-foreground">Message</span>
            <div
              className={`font-mono mt-0.5 p-2 rounded text-xs ${
                check.LastResult
                  ? 'bg-healthy/5 text-healthy border border-healthy/20'
                  : 'bg-unhealthy/5 text-unhealthy border border-unhealthy/20'
              }`}
            >
              {check.Message}
            </div>
          </div>
        </>
      )}

      {regions.length > 0 && (
        <>
          <Separator className="my-2" />
          <div className="text-xs">
            <span className="text-muted-foreground">Regions</span>
            <div className="flex flex-wrap gap-1.5 mt-1">
              {regions.map((r) => (
                <Tooltip key={r.region}>
                  <TooltipTrigger asChild>
                    <Badge
                      variant={r.is_healthy ? 'healthy' : 'unhealthy'}
                      className="text-[10px] px-1.5 py-0 cursor-default"
                    >
                      {r.region}
                    </Badge>
                  </TooltipTrigger>
                  <TooltipContent>
                    <div className="text-xs">
                      <div className="font-semibold">{r.is_healthy ? 'Healthy' : 'Failing'}</div>
                      {r.message && <div className="font-mono mt-0.5">{r.message}</div>}
                      <div className="text-muted-foreground mt-0.5">{relativeTime(r.created_at)}</div>
                    </div>
                  </TooltipContent>
                </Tooltip>
              ))}
            </div>
          </div>
        </>
      )}

      {check.IsSilenced && (
        <div className="mt-2">
          <Badge variant="warning" className="text-[10px]">Alerts Silenced</Badge>
        </div>
      )}

      {extraBadges && (
        <div className="mt-2">
          {extraBadges(check)}
        </div>
      )}
    </div>
  )
}
