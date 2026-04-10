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
import { useFavicon } from '@/hooks/useFavicon'
import { useKeyboard } from '@/hooks/useKeyboard'
import { api } from '@/lib/api'
import type { Check } from '@/lib/websocket'
import { LayoutGrid, List, ArrowUp, ArrowDown, ArrowUpDown } from 'lucide-react'

const COLLAPSED_KEY = 'checker-collapsed-groups'
const VIEW_MODE_KEY = 'checker-view-mode'
const SORT_KEY = 'checker-dashboard-sort'

type ViewMode = 'list' | 'grid'
type SortColumn = 'name' | 'type' | 'status' | 'host' | 'frequency'
type SortDirection = 'asc' | 'desc'

const VALID_SORT_COLUMNS: readonly string[] = ['name', 'type', 'status', 'host', 'frequency'] as const
const VALID_SORT_DIRECTIONS: readonly string[] = ['asc', 'desc'] as const

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

function loadSort(): { column: SortColumn | null; direction: SortDirection } {
  try {
    const val = localStorage.getItem(SORT_KEY)
    if (!val) return { column: null, direction: 'asc' }
    const parsed = JSON.parse(val)
    return {
      column: parsed.column && VALID_SORT_COLUMNS.includes(parsed.column) ? parsed.column : null,
      direction: parsed.direction && VALID_SORT_DIRECTIONS.includes(parsed.direction) ? parsed.direction : 'asc',
    }
  } catch {
    return { column: null, direction: 'asc' }
  }
}

function saveSort(column: SortColumn | null, direction: SortDirection) {
  if (column) {
    localStorage.setItem(SORT_KEY, JSON.stringify({ column, direction }))
  } else {
    localStorage.removeItem(SORT_KEY)
  }
}

export function Dashboard() {
  const { checks, previousChecks, stats, wsStatus, getGrouped } = useChecks()
  const { entries } = useEventLog(checks, previousChecks)
  useFavicon(stats.unhealthy, stats.total - stats.disabled)

  // View mode
  const [viewMode, setViewMode] = useState<ViewMode>(loadViewMode)

  // Sort state — persisted in localStorage
  const [sortState, setSortState] = useState(loadSort)
  const sortColumn = sortState.column
  const sortDirection = sortState.direction

  const handleSort = useCallback((column: SortColumn) => {
    setSortState((prev) => {
      let next: { column: SortColumn | null; direction: SortDirection }
      if (prev.column === column) {
        if (prev.direction === 'asc') {
          next = { column, direction: 'desc' }
        } else {
          next = { column: null, direction: 'asc' }
        }
      } else {
        next = { column, direction: 'asc' }
      }
      saveSort(next.column, next.direction)
      return next
    })
  }, [])

  const sortChecks = useCallback((checks: Check[]): Check[] => {
    if (!sortColumn) return checks
    return [...checks].sort((a, b) => {
      let cmp = 0
      switch (sortColumn) {
        case 'name':
          cmp = a.Name.toLowerCase().localeCompare(b.Name.toLowerCase())
          break
        case 'type':
          cmp = a.CheckType.toLowerCase().localeCompare(b.CheckType.toLowerCase())
          break
        case 'status': {
          const statusA = !a.Enabled ? 2 : a.LastResult ? 0 : 1
          const statusB = !b.Enabled ? 2 : b.LastResult ? 0 : 1
          cmp = statusA - statusB
          break
        }
        case 'host':
          cmp = (a.URL || a.Host || '').toLowerCase().localeCompare((b.URL || b.Host || '').toLowerCase())
          break
        case 'frequency':
          cmp = a.Periodicity.localeCompare(b.Periodicity)
          break
      }
      return sortDirection === 'asc' ? cmp : -cmp
    })
  }, [sortColumn, sortDirection])

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

  const groups = useMemo(() => getGrouped(filtered, sortChecks), [getGrouped, filtered, sortChecks])

  // Flat list of visible check UUIDs for keyboard navigation
  const visibleUUIDs = useMemo(() => {
    const uuids: string[] = []
    for (const group of groups) {
      const projectKey = `p:${group.name}`
      if (collapsedGroups.has(projectKey)) continue
      for (const sg of group.subGroups) {
        const sgKey = `g:${group.name}/${sg.name}`
        if (collapsedGroups.has(sgKey)) continue
        for (const c of sg.checks) uuids.push(c.UUID)
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

  // Find the project key that the selected check belongs to (for keyboard toggle)
  const selectedGroup = useMemo(() => {
    if (!selectedUUID) return null
    for (const g of groups) {
      if (g.checks.some((c) => c.UUID === selectedUUID)) return `p:${g.name}`
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
      <div className="min-h-screen bg-background pb-8">
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
          <MetricsRow
            stats={stats}
            failingChecks={checks.filter((c) => c.Enabled && !c.LastResult)}
            onSelectCheck={(uuid) => {
              setSelectedUUID(uuid)
              setExpandedUUID(uuid)
              // Expand the project group containing this check if collapsed
              const group = groups.find((g) => g.checks.some((c) => c.UUID === uuid))
              if (group) {
                const projectKey = `p:${group.name}`
                setCollapsedGroups((prev) => {
                  if (prev.has(projectKey)) {
                    const next = new Set(prev)
                    next.delete(projectKey)
                    saveCollapsed(next)
                    return next
                  }
                  return prev
                })
              }
              // Scroll to the check after a short delay to allow render
              setTimeout(() => {
                document.getElementById(`check-${uuid}`)?.scrollIntoView({ behavior: 'smooth', block: 'center' })
              }, 100)
            }}
          />

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
              sortColumn={sortColumn}
              sortDirection={sortDirection}
              onSort={handleSort}
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
