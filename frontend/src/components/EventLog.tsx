import { useEffect, useRef } from 'react'
import { ScrollArea } from '@/components/ui/scroll-area'
import type { EventLogEntry } from '@/hooks/useEventLog'
import { cn } from '@/lib/utils'

interface EventLogProps {
  entries: EventLogEntry[]
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
            <div key={entry.id} className="flex items-center gap-2 font-mono text-xs animate-slide-in">
              <span className="text-disabled shrink-0">{'>'}_</span>
              <span className="text-muted-foreground shrink-0">
                {entry.timestamp.toLocaleTimeString('en-US', { hour12: false })}
              </span>
              <span className="text-foreground truncate shrink-0 max-w-[100px] sm:max-w-[140px] sm:w-[140px]">{entry.checkName}</span>
              <span className="text-muted-foreground shrink-0">&rarr;</span>
              <span
                className={cn(
                  'font-semibold shrink-0',
                  entry.status === 'healthy' && 'text-healthy',
                  entry.status === 'unhealthy' && 'text-unhealthy',
                  entry.status === 'disabled' && 'text-disabled',
                  entry.status === 'enabled' && 'text-info'
                )}
              >
                {entry.status}
              </span>
              {entry.message && (
                <span className="text-muted-foreground truncate">{entry.message}</span>
              )}
            </div>
          ))}
          <div ref={bottomRef} />
        </div>
      </ScrollArea>
    </div>
  )
}
