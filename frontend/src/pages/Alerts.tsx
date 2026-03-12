import { useState, useEffect, useMemo } from 'react'
import { TooltipProvider } from '@/components/ui/tooltip'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from '@/components/ui/collapsible'
import { StatusBar } from '@/components/StatusBar'
import { useAlerts } from '@/hooks/useAlerts'
import { api } from '@/lib/api'
import {
  Bell,
  BellOff,
  CheckCircle2,
  XCircle,
  ChevronDown,
  ChevronLeft,
  ChevronRight,
  RefreshCw,
  VolumeX,
  Volume2,
  Clock,
  Shield,
  Trash2,
  Settings,
  LogOut,
  User,
} from 'lucide-react'
import { Link } from 'react-router-dom'

function timeAgo(dateStr: string): string {
  const now = Date.now()
  const then = new Date(dateStr).getTime()
  const diffMs = now - then
  if (diffMs < 0) return 'just now'

  const seconds = Math.floor(diffMs / 1000)
  if (seconds < 60) return `${seconds}s ago`
  const minutes = Math.floor(seconds / 60)
  if (minutes < 60) return `${minutes}m ago`
  const hours = Math.floor(minutes / 60)
  if (hours < 24) return `${hours}h ago`
  const days = Math.floor(hours / 24)
  return `${days}d ago`
}

function formatDuration(startStr: string, endStr: string | null): string {
  const start = new Date(startStr).getTime()
  const end = endStr ? new Date(endStr).getTime() : Date.now()
  const diffMs = end - start
  if (diffMs < 0) return '-'

  const seconds = Math.floor(diffMs / 1000)
  if (seconds < 60) return `${seconds}s`
  const minutes = Math.floor(seconds / 60)
  if (minutes < 60) return `${minutes}m`
  const hours = Math.floor(minutes / 60)
  if (hours < 24) return `${hours}h ${minutes % 60}m`
  const days = Math.floor(hours / 24)
  return `${days}d ${hours % 24}h`
}

function formatExpiresAt(dateStr: string | null): string {
  if (!dateStr) return 'Indefinite'
  const expires = new Date(dateStr)
  const now = new Date()
  if (expires <= now) return 'Expired'
  return timeAgo(dateStr).replace(' ago', ' remaining').replace('just now', 'expiring now')
}

const SILENCE_DURATIONS = [
  { label: '30 minutes', value: '30m' },
  { label: '1 hour', value: '1h' },
  { label: '4 hours', value: '4h' },
  { label: '8 hours', value: '8h' },
  { label: '24 hours', value: '24h' },
  { label: 'Indefinite', value: 'indefinite' },
]

export function Alerts() {
  const {
    alerts,
    total,
    loading,
    projectFilter,
    setProjectFilter,
    statusFilter,
    setStatusFilter,
    silences,
    silencesLoading,
    wsStatus,
    recentAlertIds,
    currentPage,
    totalPages,
    goToPage,
    fetchAlerts,
    fetchSilences,
  } = useAlerts()

  const [projects, setProjects] = useState<string[]>([])
  const [silencesOpen, setSilencesOpen] = useState(true)

  // Fetch projects list for filters
  useEffect(() => {
    api.getProjects().then(setProjects).catch(() => {})
  }, [])

  // Build a set of silenced check UUIDs and project names for quick lookup
  const silencedChecks = useMemo(() => {
    const set = new Set<string>()
    for (const s of silences) {
      if (s.Active) {
        set.add(s.Target)
      }
    }
    return set
  }, [silences])

  const isCheckSilenced = (checkUUID: string, project: string) => {
    return silencedChecks.has(checkUUID) || silencedChecks.has(project)
  }

  const handleSilenceCheck = async (checkUUID: string, duration: string) => {
    try {
      await api.createSilence({ scope: 'check', target: checkUUID, duration })
      fetchSilences()
    } catch (err) {
      console.error('Failed to silence check:', err)
    }
  }

  const handleSilenceProject = async (project: string, duration: string) => {
    try {
      await api.createSilence({ scope: 'project', target: project, duration })
      fetchSilences()
    } catch (err) {
      console.error('Failed to silence project:', err)
    }
  }

  const handleUnsilence = async (id: number) => {
    try {
      await api.deleteSilence(id)
      fetchSilences()
    } catch (err) {
      console.error('Failed to unsilence:', err)
    }
  }

  return (
    <TooltipProvider delayDuration={300}>
      <div className="min-h-screen pb-8">
        {/* Navigation Bar */}
        <header className="sticky top-0 z-40 border-b bg-[hsl(215_25%_11%)] scanline-bg">
          <div className="mx-auto max-w-[1600px] px-4 py-2">
            <div className="flex items-center gap-4">
              {/* Logo */}
              <Link to="/" className="flex items-center gap-2 shrink-0">
                <div className="h-7 w-7 rounded bg-healthy/20 flex items-center justify-center">
                  <span className="text-healthy font-mono font-bold text-sm">C</span>
                </div>
                <span className="font-semibold text-foreground hidden sm:inline">Checker</span>
              </Link>

              {/* Nav */}
              <nav className="flex items-center gap-1 shrink-0">
                <Link to="/">
                  <Button variant="ghost" size="sm">
                    Dashboard
                  </Button>
                </Link>
                <Link to="/manage">
                  <Button variant="ghost" size="sm">
                    <Settings className="h-4 w-4 mr-1" />
                    Manage
                  </Button>
                </Link>
                <Link to="/alerts">
                  <Button variant="secondary" size="sm">
                    <Bell className="h-4 w-4 mr-1" />
                    Alerts
                  </Button>
                </Link>
              </nav>

              <div className="flex-1" />

              {/* Filters */}
              <div className="hidden lg:flex items-center gap-2">
                <Select value={statusFilter} onValueChange={setStatusFilter}>
                  <SelectTrigger className="h-8 w-[120px] text-xs">
                    <SelectValue placeholder="Status" />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="all">All Status</SelectItem>
                    <SelectItem value="active">Active</SelectItem>
                    <SelectItem value="resolved">Resolved</SelectItem>
                  </SelectContent>
                </Select>

                <Select value={projectFilter} onValueChange={setProjectFilter}>
                  <SelectTrigger className="h-8 w-[130px] text-xs">
                    <SelectValue placeholder="Project" />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="all">All Projects</SelectItem>
                    {projects.map((p) => (
                      <SelectItem key={p} value={p}>
                        {p}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>

              {/* User menu */}
              <div className="flex items-center gap-2 shrink-0">
                <DropdownMenu>
                  <DropdownMenuTrigger asChild>
                    <Button variant="ghost" size="icon" className="h-8 w-8">
                      <User className="h-4 w-4" />
                    </Button>
                  </DropdownMenuTrigger>
                  <DropdownMenuContent align="end">
                    <DropdownMenuItem asChild>
                      <a href="/auth/logout">
                        <LogOut className="mr-2 h-4 w-4" />
                        Logout
                      </a>
                    </DropdownMenuItem>
                  </DropdownMenuContent>
                </DropdownMenu>
              </div>
            </div>
          </div>
        </header>

        <main className="mx-auto max-w-[1600px] px-4 py-4 space-y-4">
          {/* Page header */}
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-2">
              <Bell className="h-5 w-5 text-muted-foreground" />
              <h2 className="text-lg font-semibold">Alert History</h2>
              <Badge variant="secondary" className="text-xs">
                {total} total
              </Badge>
            </div>
            <Button
              variant="outline"
              size="sm"
              onClick={() => fetchAlerts()}
              disabled={loading}
            >
              <RefreshCw className={`h-4 w-4 mr-1 ${loading ? 'animate-spin' : ''}`} />
              Refresh
            </Button>
          </div>

          {/* Mobile filters */}
          <div className="flex lg:hidden items-center gap-2">
            <Select value={statusFilter} onValueChange={setStatusFilter}>
              <SelectTrigger className="h-8 text-xs flex-1">
                <SelectValue placeholder="Status" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">All Status</SelectItem>
                <SelectItem value="active">Active</SelectItem>
                <SelectItem value="resolved">Resolved</SelectItem>
              </SelectContent>
            </Select>

            <Select value={projectFilter} onValueChange={setProjectFilter}>
              <SelectTrigger className="h-8 text-xs flex-1">
                <SelectValue placeholder="Project" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">All Projects</SelectItem>
                {projects.map((p) => (
                  <SelectItem key={p} value={p}>
                    {p}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>

          {/* Alert history table */}
          <div className="rounded-lg border bg-card overflow-hidden">
            <div className="overflow-x-auto">
              <table className="w-full text-sm">
                <thead>
                  <tr className="border-b bg-[hsl(215_14%_10%)] text-muted-foreground text-xs">
                    <th className="text-left px-3 py-2 font-medium w-10">Status</th>
                    <th className="text-left px-3 py-2 font-medium">Check Name</th>
                    <th className="text-left px-3 py-2 font-medium hidden md:table-cell">Project</th>
                    <th className="text-left px-3 py-2 font-medium hidden lg:table-cell">Message</th>
                    <th className="text-left px-3 py-2 font-medium">Time</th>
                    <th className="text-left px-3 py-2 font-medium hidden sm:table-cell">Duration</th>
                    <th className="text-right px-3 py-2 font-medium">Actions</th>
                  </tr>
                </thead>
                <tbody>
                  {loading && alerts.length === 0 ? (
                    <tr>
                      <td colSpan={7} className="text-center py-8 text-muted-foreground">
                        Loading alerts...
                      </td>
                    </tr>
                  ) : alerts.length === 0 ? (
                    <tr>
                      <td colSpan={7} className="text-center py-8 text-muted-foreground">
                        <div className="flex flex-col items-center gap-2">
                          <CheckCircle2 className="h-8 w-8 text-healthy/50" />
                          <span>No alerts found</span>
                        </div>
                      </td>
                    </tr>
                  ) : (
                    alerts.map((alert) => {
                      const isActive = !alert.IsResolved
                      const isRecent = recentAlertIds.has(alert.ID)
                      const isSilenced = isCheckSilenced(alert.CheckUUID, alert.Project)

                      return (
                        <tr
                          key={alert.ID}
                          className={`
                            border-b border-border/50 transition-all duration-500
                            ${isActive
                              ? 'bg-unhealthy/5 hover:bg-unhealthy/10'
                              : 'opacity-60 hover:opacity-80 hover:bg-[hsl(215_14%_14%)]'
                            }
                            ${isRecent ? 'bg-warning/10 animate-pulse' : ''}
                          `}
                        >
                          {/* Status icon */}
                          <td className="px-3 py-2">
                            {isActive ? (
                              <XCircle className="h-4 w-4 text-unhealthy" />
                            ) : (
                              <CheckCircle2 className="h-4 w-4 text-healthy" />
                            )}
                          </td>

                          {/* Check name */}
                          <td className="px-3 py-2">
                            <div className="flex items-center gap-1.5">
                              <span className="font-medium">{alert.CheckName}</span>
                              {isSilenced && (
                                <Badge variant="warning" className="text-[10px] py-0">
                                  <VolumeX className="h-3 w-3 mr-0.5" />
                                  Silenced
                                </Badge>
                              )}
                            </div>
                            <div className="text-[10px] text-muted-foreground font-mono md:hidden">
                              {alert.Project}
                            </div>
                          </td>

                          {/* Project */}
                          <td className="px-3 py-2 text-muted-foreground hidden md:table-cell">
                            {alert.Project}
                          </td>

                          {/* Message */}
                          <td className="px-3 py-2 text-muted-foreground hidden lg:table-cell max-w-[300px] truncate">
                            {alert.Message || '-'}
                          </td>

                          {/* Time */}
                          <td className="px-3 py-2 text-muted-foreground whitespace-nowrap">
                            {timeAgo(alert.CreatedAt)}
                          </td>

                          {/* Duration */}
                          <td className="px-3 py-2 text-muted-foreground font-mono text-xs hidden sm:table-cell">
                            {formatDuration(alert.CreatedAt, alert.ResolvedAt)}
                          </td>

                          {/* Actions */}
                          <td className="px-3 py-2 text-right">
                            {isActive && !isSilenced && (
                              <DropdownMenu>
                                <DropdownMenuTrigger asChild>
                                  <Button variant="ghost" size="sm" className="h-7 text-xs">
                                    <VolumeX className="h-3.5 w-3.5 mr-1" />
                                    Silence
                                  </Button>
                                </DropdownMenuTrigger>
                                <DropdownMenuContent align="end" className="w-48">
                                  <div className="px-2 py-1.5 text-xs font-medium text-muted-foreground">
                                    Silence Check
                                  </div>
                                  {SILENCE_DURATIONS.map((d) => (
                                    <DropdownMenuItem
                                      key={d.value}
                                      onClick={() => handleSilenceCheck(alert.CheckUUID, d.value)}
                                    >
                                      <Clock className="h-3.5 w-3.5 mr-2" />
                                      {d.label}
                                    </DropdownMenuItem>
                                  ))}
                                  <DropdownMenuSeparator />
                                  <div className="px-2 py-1.5 text-xs font-medium text-muted-foreground">
                                    Silence Project
                                  </div>
                                  {SILENCE_DURATIONS.map((d) => (
                                    <DropdownMenuItem
                                      key={`proj-${d.value}`}
                                      onClick={() => handleSilenceProject(alert.Project, d.value)}
                                    >
                                      <Shield className="h-3.5 w-3.5 mr-2" />
                                      {d.label}
                                    </DropdownMenuItem>
                                  ))}
                                </DropdownMenuContent>
                              </DropdownMenu>
                            )}
                            {isSilenced && isActive && (
                              <Badge variant="warning" className="text-[10px]">
                                Silenced
                              </Badge>
                            )}
                          </td>
                        </tr>
                      )
                    })
                  )}
                </tbody>
              </table>
            </div>
          </div>

          {/* Pagination */}
          {totalPages > 1 && (
            <div className="flex items-center justify-between">
              <span className="text-xs text-muted-foreground">
                Page {currentPage + 1} of {totalPages}
              </span>
              <div className="flex items-center gap-1">
                <Button
                  variant="outline"
                  size="sm"
                  disabled={currentPage === 0}
                  onClick={() => goToPage(currentPage - 1)}
                >
                  <ChevronLeft className="h-4 w-4" />
                  Previous
                </Button>
                <Button
                  variant="outline"
                  size="sm"
                  disabled={currentPage >= totalPages - 1}
                  onClick={() => goToPage(currentPage + 1)}
                >
                  Next
                  <ChevronRight className="h-4 w-4" />
                </Button>
              </div>
            </div>
          )}

          {/* Active Silences Section */}
          <Collapsible open={silencesOpen} onOpenChange={setSilencesOpen}>
            <CollapsibleTrigger asChild>
              <button className="flex items-center gap-2 w-full text-left group">
                {silencesOpen ? (
                  <ChevronDown className="h-4 w-4 text-muted-foreground" />
                ) : (
                  <ChevronRight className="h-4 w-4 text-muted-foreground" />
                )}
                <BellOff className="h-4 w-4 text-muted-foreground" />
                <h3 className="text-sm font-semibold">Active Silences</h3>
                <Badge variant="secondary" className="text-[10px]">
                  {silences.length}
                </Badge>
              </button>
            </CollapsibleTrigger>
            <CollapsibleContent>
              <div className="mt-2 rounded-lg border bg-card overflow-hidden">
                <div className="overflow-x-auto">
                  <table className="w-full text-sm">
                    <thead>
                      <tr className="border-b bg-[hsl(215_14%_10%)] text-muted-foreground text-xs">
                        <th className="text-left px-3 py-2 font-medium">Scope</th>
                        <th className="text-left px-3 py-2 font-medium">Target</th>
                        <th className="text-left px-3 py-2 font-medium hidden sm:table-cell">Silenced By</th>
                        <th className="text-left px-3 py-2 font-medium hidden md:table-cell">Reason</th>
                        <th className="text-left px-3 py-2 font-medium">Expires</th>
                        <th className="text-right px-3 py-2 font-medium">Actions</th>
                      </tr>
                    </thead>
                    <tbody>
                      {silencesLoading ? (
                        <tr>
                          <td colSpan={6} className="text-center py-6 text-muted-foreground">
                            Loading silences...
                          </td>
                        </tr>
                      ) : silences.length === 0 ? (
                        <tr>
                          <td colSpan={6} className="text-center py-6 text-muted-foreground">
                            <div className="flex flex-col items-center gap-2">
                              <Volume2 className="h-6 w-6 text-muted-foreground/50" />
                              <span>No active silences</span>
                            </div>
                          </td>
                        </tr>
                      ) : (
                        silences.map((silence) => (
                          <tr
                            key={silence.ID}
                            className="border-b border-border/50 hover:bg-[hsl(215_14%_14%)] transition-colors"
                          >
                            <td className="px-3 py-2">
                              <Badge
                                variant={silence.Scope === 'project' ? 'info' : 'secondary'}
                                className="text-[10px]"
                              >
                                {silence.Scope === 'project' ? (
                                  <Shield className="h-3 w-3 mr-0.5" />
                                ) : (
                                  <Bell className="h-3 w-3 mr-0.5" />
                                )}
                                {silence.Scope}
                              </Badge>
                            </td>
                            <td className="px-3 py-2 font-medium">
                              {silence.Target}
                            </td>
                            <td className="px-3 py-2 text-muted-foreground hidden sm:table-cell">
                              {silence.SilencedBy || '-'}
                            </td>
                            <td className="px-3 py-2 text-muted-foreground hidden md:table-cell max-w-[200px] truncate">
                              {silence.Reason || '-'}
                            </td>
                            <td className="px-3 py-2 text-muted-foreground whitespace-nowrap">
                              {formatExpiresAt(silence.ExpiresAt)}
                            </td>
                            <td className="px-3 py-2 text-right">
                              <Button
                                variant="ghost"
                                size="sm"
                                className="h-7 text-xs text-unhealthy hover:text-unhealthy"
                                onClick={() => handleUnsilence(silence.ID)}
                              >
                                <Trash2 className="h-3.5 w-3.5 mr-1" />
                                Unsilence
                              </Button>
                            </td>
                          </tr>
                        ))
                      )}
                    </tbody>
                  </table>
                </div>
              </div>
            </CollapsibleContent>
          </Collapsible>
        </main>

        <StatusBar wsStatus={wsStatus} />
      </div>
    </TooltipProvider>
  )
}
