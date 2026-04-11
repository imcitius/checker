import { useEffect, useRef } from 'react'
import { ScrollArea } from '@/components/ui/scroll-area'
import type { EventLogEntry } from '@/hooks/useEventLog'
import { cn } from '@/lib/utils'

interface EventLogProps {
  entries: EventLogEntry[]
}

function relativeTime(date: Date): string {
  const seconds = Math.floor((Date.now() - date.getTime()) / 1000)
  if (seconds < 5) return 'just now'
  if (seconds < 60) return `${seconds}s ago`
  const minutes = Math.floor(seconds / 60)
  if (minutes < 60) return `${minutes}m ago`
  const hours = Math.floor(minutes / 60)
  if (hours < 24) return `${hours}h ago`
  const days = Math.floor(hours / 24)
  return `${days}d ago`
}

function statusLabel(status: EventLogEntry['status']): string {
  switch (status) {
    case 'healthy': return 'HEALTHY'
    case 'unhealthy': return 'FAILING'
    case 'disabled': return 'DISABLED'
    case 'enabled': return 'ENABLED'
  }
}

export function EventLog({ entries }: EventLogProps) {
  const bottomRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [entries.length])

  if (entries.length === 0) {
    return (
      <div className="rounded-lg border bg-card p-3">
        <div className="text-xs text-muted-foreground font-mono">
          <span className="text-disabled">{'>'}_</span> Waiting for state transitions...
        </div>
      </div>
    )
  }

  return (
    <div className="rounded-lg border bg-card overflow-hidden">
      <div className="px-3 py-1.5 border-b bg-muted/50">
        <span className="text-xs text-muted-foreground font-mono">
          <span className="text-disabled">{'>'}_</span> Event Log
        </span>
      </div>
      <ScrollArea className="h-[140px]">
        <div className="p-2 space-y-0.5">
          {entries.map((entry) => (
            <div key={entry.id} className="flex items-start gap-1.5 font-mono text-xs animate-slide-in">
              <span className="text-disabled shrink-0">{'>'}_</span>
              {/* Relative timestamp */}
              <span className="text-muted-foreground shrink-0 w-[56px] text-right">
                {relativeTime(entry.timestamp)}
              </span>
              {/* Check name + project */}
              <span className="text-foreground shrink-0 truncate max-w-[130px]">
                {entry.checkName}
                {entry.project && (
                  <span className="text-muted-foreground"> ({entry.project})</span>
                )}
              </span>
              <span className="text-muted-foreground shrink-0">&mdash;</span>
              {/* old → new state transition */}
              <span className="shrink-0 flex items-center gap-1">
                {entry.previousStatus && (
                  <>
                    <span className={cn(
                      'opacity-60',
                      entry.previousStatus === 'healthy' && 'text-healthy',
                      entry.previousStatus === 'unhealthy' && 'text-unhealthy',
                      (entry.previousStatus === 'disabled' || entry.previousStatus === 'enabled') && 'text-disabled',
                    )}>
                      {statusLabel(entry.previousStatus as EventLogEntry['status'])}
                    </span>
                    <span className="text-muted-foreground">&rarr;</span>
                  </>
                )}
                <span
                  className={cn(
                    'font-semibold',
                    entry.status === 'healthy' && 'text-healthy',
                    entry.status === 'unhealthy' && 'text-unhealthy',
                    entry.status === 'disabled' && 'text-disabled',
                    entry.status === 'enabled' && 'text-info'
                  )}
                >
                  {statusLabel(entry.status)}
                </span>
              </span>
              {/* Error / info message */}
              {entry.message && (
                <>
                  <span className="text-muted-foreground shrink-0">&mdash;</span>
                  <span className={cn(
                    'truncate',
                    entry.status === 'unhealthy' ? 'text-unhealthy/75' : 'text-muted-foreground'
                  )}>
                    {entry.message}
                  </span>
                </>
              )}
            </div>
          ))}
          <div ref={bottomRef} />
        </div>
      </ScrollArea>
    </div>
  )
}
