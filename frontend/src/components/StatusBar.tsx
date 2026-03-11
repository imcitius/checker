import type { WSStatus } from '@/hooks/useChecks'
import { cn } from '@/lib/utils'

interface StatusBarProps {
  wsStatus: WSStatus
}

export function StatusBar({ wsStatus }: StatusBarProps) {
  return (
    <div className="fixed bottom-0 left-0 right-0 border-t bg-[hsl(215_25%_9%)] z-40">
      <div className="mx-auto max-w-[1600px] px-4 py-1 flex items-center gap-3 text-xs font-mono text-muted-foreground">
        <span className="text-disabled">{'>'}_</span>
        <span
          className={cn(
            'flex items-center gap-1.5',
            wsStatus === 'connected' && 'text-healthy',
            wsStatus === 'disconnected' && 'text-unhealthy',
            wsStatus === 'connecting' && 'text-warning'
          )}
        >
          <span
            className={cn(
              'inline-block h-1.5 w-1.5 rounded-full',
              wsStatus === 'connected' && 'bg-healthy',
              wsStatus === 'disconnected' && 'bg-unhealthy',
              wsStatus === 'connecting' && 'bg-warning animate-pulse'
            )}
          />
          {wsStatus}
        </span>
        <span className="text-border">·</span>
        <span>ws:{wsStatus === 'connected' ? 'ok' : wsStatus === 'connecting' ? '...' : 'err'}</span>
        <span className="text-border">·</span>
        <span>refresh 30s</span>
        <span className="ml-auto text-[10px] hidden sm:inline">
          <kbd>j</kbd>/<kbd>k</kbd> navigate
          <span className="mx-1">·</span>
          <kbd>Enter</kbd> expand
          <span className="mx-1">·</span>
          <kbd>/</kbd> search
          <span className="mx-1">·</span>
          <kbd>⌘K</kbd> palette
        </span>
      </div>
    </div>
  )
}
