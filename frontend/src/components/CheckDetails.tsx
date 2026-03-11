import type { Check } from '@/lib/websocket'
import { Badge } from '@/components/ui/badge'
import { Separator } from '@/components/ui/separator'

interface CheckDetailsProps {
  check: Check
}

export function CheckDetails({ check }: CheckDetailsProps) {
  return (
    <div className="px-6 py-3 bg-[hsl(215_14%_10%)] border-t border-border/50 animate-slide-in">
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
          <div className="font-mono text-foreground mt-0.5 truncate">{check.URL || check.Host || '—'}</div>
        </div>
        <div>
          <span className="text-muted-foreground">Frequency</span>
          <div className="font-mono text-foreground mt-0.5">{check.Periodicity || '—'}</div>
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

      {check.IsSilenced && (
        <div className="mt-2">
          <Badge variant="warning" className="text-[10px]">Alerts Silenced</Badge>
        </div>
      )}
    </div>
  )
}
