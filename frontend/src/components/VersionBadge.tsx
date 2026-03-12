import { useEffect, useState } from 'react'
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from '@/components/ui/tooltip'

interface VersionInfo {
  version: string
  build_time: string
  frontend_version: string
}

export function VersionBadge() {
  const [info, setInfo] = useState<VersionInfo | null>(null)
  const [error, setError] = useState(false)

  const clientSha = import.meta.env.VITE_GIT_SHA as string | undefined

  useEffect(() => {
    fetch('/api/version')
      .then((res) => {
        if (!res.ok) throw new Error(res.statusText)
        return res.json()
      })
      .then((data: VersionInfo) => setInfo(data))
      .catch(() => setError(true))
  }, [])

  if (error) {
    return (
      <div className="fixed bottom-2 right-2 z-50 select-none text-xs text-muted-foreground/40">
        build: unknown
      </div>
    )
  }

  if (!info) return null

  const shortSha = info.version ? info.version.slice(0, 7) : 'unknown'
  const serverFrontendSha = info.frontend_version || 'unknown'
  const clientShort = clientSha ? clientSha.slice(0, 7) : 'unknown'

  // Mismatch: the Go-embedded frontend SHA doesn't match the client bundle's SHA
  const hasMismatch =
    clientSha &&
    serverFrontendSha !== 'unknown' &&
    clientSha !== serverFrontendSha

  return (
    <TooltipProvider>
      <Tooltip>
        <TooltipTrigger asChild>
          <div
            className={`fixed bottom-2 right-2 z-50 cursor-default select-none rounded px-1.5 py-0.5 font-mono text-xs transition-opacity hover:opacity-100 ${
              hasMismatch
                ? 'bg-orange-500/10 text-orange-500/70'
                : 'text-muted-foreground/40'
            }`}
          >
            {shortSha}
            {hasMismatch && ' ⚠'}
          </div>
        </TooltipTrigger>
        <TooltipContent side="top" align="end" className="max-w-xs font-mono text-xs">
          <div className="space-y-1">
            <div>
              <span className="text-muted-foreground">Server SHA: </span>
              {info.version || 'unknown'}
            </div>
            <div>
              <span className="text-muted-foreground">Build time: </span>
              {info.build_time || 'unknown'}
            </div>
            <div>
              <span className="text-muted-foreground">Embedded frontend: </span>
              {serverFrontendSha}
            </div>
            <div>
              <span className="text-muted-foreground">Client bundle: </span>
              {clientSha || 'unknown'}
            </div>
            {hasMismatch && (
              <div className="mt-1 text-orange-400">
                ⚠ Frontend version mismatch — client bundle differs from server-embedded version
              </div>
            )}
            {!hasMismatch && clientSha && serverFrontendSha !== 'unknown' && (
              <div className="mt-1 text-green-400">✓ Frontend versions match</div>
            )}
          </div>
        </TooltipContent>
      </Tooltip>
    </TooltipProvider>
  )
}
