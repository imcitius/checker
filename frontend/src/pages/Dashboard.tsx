import { useState, useCallback, useEffect, useMemo, useRef } from 'react'
import { useSearchParams } from 'react-router-dom'
import { TooltipProvider } from '@/components/ui/tooltip'
import { Button } from '@/components/ui/button'
import { TopBar } from '@/components/TopBar'
import { MetricsRow } from '@/components/MetricsRow'
import { CheckList } from '@/components/CheckList'
import { HealthMap } from '@/components/HealthMap'
import { EventLog } from '@/components/EventLog'
import { StatusBar } from '@/components/StatusBar'
import { useChecks } from '@/hooks/useChecks'
import { useEventLog } from '@/hooks/useEventLog'
import { useKeyboard } from '@/hooks/useKeyboard'
import { api } from '@/lib/api'
import type { Check } from '@/lib/websocket'
import { LayoutGrid, List } from 'lucide-react'

const COLLAPSED_KEY = 'checker-collapsed-groups'
const VIEW_MODE_KEY = 'checker-view-mode'

type ViewMode = 'list' | 'grid'

function loadCollapsed(): Set<string> {
  try {
    const val = localStorage.getItem(COLLAPSED_KEY)
    return val ? new Set(JSON.parse(val)) : new Set()
  } catch {
    return new Set()
  }
}

function saveCollapsed(set: Set<string>) {
  localStorage.setItem(COLLAPSED_KEY, JSON.stringify([...set]))
}

function loadViewMode(): ViewMode {
  return (localStorage.getItem(VIEW_MODE_KEY) as ViewMode) || 'list'
}

export function Dashboard() {
  const { checks, previousChecks, stats, wsStatus, getGrouped } = useChecks()
  const { entries } = useEventLog(checks, previousChecks)

  // View mode
  const [viewMode, setViewMode] = useState<ViewMode>(loadViewMode)

  const handleSetViewMode = (mode: ViewMode) => {
    setViewMode(mode)
    localStorage.setItem(VIEW_MODE_KEY, mode)
  }

  // Filters
  const [search, setSearch] = useState('')
  const [statusFilter, setStatusFilter] = useState('all')
  const [projectFilter, setProjectFilter] = useState('all')
  const [typeFilter, setTypeFilter] = useState('all')
  const searchRef = useRef<HTMLInputElement>(null)

  // Selection
  const [searchParams, setSearchParams] = useSearchParams()
  const [selectedUUID, setSelectedUUID] = useState<string | null>(null)
  const [expandedUUID, setExpandedUUID] = useState<string | null>(null)

  // Handle command-palette deep-link: ?check=UUID
  useEffect(() => {
    const checkUUID = searchParams.get('check')
    if (checkUUID && checks.length > 0) {
      setSelectedUUID(checkUUID)
      setExpandedUUID(checkUUID)
      setSearchParams((prev) => {
        const next = new URLSearchParams(prev)
        next.delete('check')
        return next
      }, { replace: true })
    }
  }, [searchParams, checks.length, setSearchParams])

  // Group collapse state
  const [collapsedGroups, setCollapsedGroups] = useState<Set<string>>(loadCollapsed)

  // Derive unique projects and types from checks
  const projects = useMemo(
    () => [...new Set(checks.map((c) => c.Project).filter(Boolean))].sort(),
    [checks]
  )
  const checkTypes = useMemo(
    () => [...new Set(checks.map((c) => c.CheckType).filter(Boolean))].sort(),
    [checks]
  )

  // Filter checks
  const filtered = useMemo(() => {
    return checks.filter((c) => {
      if (search) {
        const q = search.toLowerCase()
        const match =
          c.Name.toLowerCase().includes(q) ||
          c.UUID.toLowerCase().includes(q) ||
          c.Host.toLowerCase().includes(q) ||
          c.URL.toLowerCase().includes(q) ||
          c.Project.toLowerCase().includes(q)
        if (!match) return false
      }
      if (statusFilter !== 'all') {
        if (statusFilter === 'healthy' && (!c.Enabled || !c.LastResult)) return false
        if (statusFilter === 'unhealthy' && (!c.Enabled || c.LastResult)) return false
        if (statusFilter === 'disabled' && c.Enabled) return false
        if (statusFilter === 'silenced' && !c.IsSilenced) return false
      }
      if (projectFilter !== 'all' && c.Project !== projectFilter) return false
      if (typeFilter !== 'all' && c.CheckType !== typeFilter) return false
      return true
    })
  }, [checks, search, statusFilter, projectFilter, typeFilter])

  const groups = useMemo(() => getGrouped(filtered), [getGrouped, filtered])

  // Flat list of visible check UUIDs for keyboard navigation
  const visibleUUIDs = useMemo(() => {
    const uuids: string[] = []
    for (const group of groups) {
      if (!collapsedGroups.has(group.name)) {
        for (const c of group.checks) uuids.push(c.UUID)
      }
    }
    return uuids
  }, [groups, collapsedGroups])

  const toggleGroup = useCallback((name: string) => {
    setCollapsedGroups((prev) => {
      const next = new Set(prev)
      if (next.has(name)) next.delete(name)
      else next.add(name)
      saveCollapsed(next)
      return next
    })
  }, [])

  const handleToggleCheck = useCallback(async (uuid: string, enabled: boolean) => {
    try {
      await api.toggleCheck(uuid, enabled)
    } catch (err) {
      console.error('Failed to toggle check:', err)
    }
  }, [])

  const handleSelectCheck = useCallback(
    (uuid: string) => {
      if (selectedUUID === uuid) {
        setExpandedUUID((prev) => (prev === uuid ? null : uuid))
      } else {
        setSelectedUUID(uuid)
      }
    },
    [selectedUUID]
  )

  // Find the group that the selected check belongs to
  const selectedGroup = useMemo(() => {
    if (!selectedUUID) return null
    for (const g of groups) {
      if (g.checks.some((c) => c.UUID === selectedUUID)) return g.name
    }
    return null
  }, [selectedUUID, groups])

  // Keyboard actions
  useKeyboard({
    onNavigateDown: () => {
      if (visibleUUIDs.length === 0) return
      const idx = selectedUUID ? visibleUUIDs.indexOf(selectedUUID) : -1
      const next = Math.min(idx + 1, visibleUUIDs.length - 1)
      setSelectedUUID(visibleUUIDs[next])
    },
    onNavigateUp: () => {
      if (visibleUUIDs.length === 0) return
      const idx = selectedUUID ? visibleUUIDs.indexOf(selectedUUID) : visibleUUIDs.length
      const next = Math.max(idx - 1, 0)
      setSelectedUUID(visibleUUIDs[next])
    },
    onExpand: () => {
      if (selectedUUID) {
        setExpandedUUID((prev) => (prev === selectedUUID ? null : selectedUUID))
      }
    },
    onCollapse: () => {
      setExpandedUUID(null)
    },
    onFocusSearch: () => {
      searchRef.current?.focus()
    },
    onToggleGroup: () => {
      if (selectedGroup) toggleGroup(selectedGroup)
    },
    onCommandPalette: () => {
      // Handled globally by CommandPalette component
    },
  })

  return (
    <TooltipProvider delayDuration={300}>
      <div className="min-h-screen pb-8">
        <TopBar
          search={search}
          onSearchChange={setSearch}
          statusFilter={statusFilter}
          onStatusFilterChange={setStatusFilter}
          projectFilter={projectFilter}
          onProjectFilterChange={setProjectFilter}
          typeFilter={typeFilter}
          onTypeFilterChange={setTypeFilter}
          projects={projects}
          checkTypes={checkTypes}
          searchRef={searchRef}
        />

        <main className="mx-auto max-w-[1600px] px-4 py-4 space-y-4">
          <MetricsRow stats={stats} />

          {/* View mode toggle */}
          <div className="flex items-center justify-between">
            <div className="text-xs text-muted-foreground">
              {filtered.length} check{filtered.length !== 1 ? 's' : ''}
              {search || statusFilter !== 'all' || projectFilter !== 'all' || typeFilter !== 'all'
                ? ' (filtered)'
                : ''}
            </div>
            <div className="flex items-center gap-1 border rounded-md p-0.5 bg-muted/50">
              <Button
                variant={viewMode === 'list' ? 'secondary' : 'ghost'}
                size="sm"
                className="h-6 px-2 text-xs"
                onClick={() => handleSetViewMode('list')}
              >
                <List className="h-3.5 w-3.5 mr-1" />
                List
              </Button>
              <Button
                variant={viewMode === 'grid' ? 'secondary' : 'ghost'}
                size="sm"
                className="h-6 px-2 text-xs"
                onClick={() => handleSetViewMode('grid')}
              >
                <LayoutGrid className="h-3.5 w-3.5 mr-1" />
                Map
              </Button>
            </div>
          </div>

          {viewMode === 'list' ? (
            <CheckList
              groups={groups}
              collapsedGroups={collapsedGroups}
              onToggleGroup={toggleGroup}
              selectedUUID={selectedUUID}
              expandedUUID={expandedUUID}
              onSelectCheck={handleSelectCheck}
              onToggleCheck={handleToggleCheck}
            />
          ) : (
            <HealthMap groups={groups} onSelectCheck={handleSelectCheck} />
          )}

          <EventLog entries={entries} />
        </main>

        <StatusBar wsStatus={wsStatus} />

      </div>
    </TooltipProvider>
  )
}
