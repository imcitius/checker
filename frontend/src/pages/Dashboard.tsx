import { useState, useCallback, useMemo, useRef } from 'react'
import { TooltipProvider } from '@/components/ui/tooltip'
import {
  CommandDialog,
  CommandInput,
  CommandList,
  CommandEmpty,
  CommandGroup,
  CommandItem,
  CommandShortcut,
} from '@/components/ui/command'
import { TopBar } from '@/components/TopBar'
import { MetricsRow } from '@/components/MetricsRow'
import { CheckList } from '@/components/CheckList'
import { EventLog } from '@/components/EventLog'
import { StatusBar } from '@/components/StatusBar'
import { useChecks } from '@/hooks/useChecks'
import { useEventLog } from '@/hooks/useEventLog'
import { useKeyboard } from '@/hooks/useKeyboard'
import type { Check } from '@/lib/websocket'

const COLLAPSED_KEY = 'checker-collapsed-groups'

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

export function Dashboard() {
  const { checks, previousChecks, stats, wsStatus, getGrouped } = useChecks()
  const { entries } = useEventLog(checks, previousChecks)

  // Filters
  const [search, setSearch] = useState('')
  const [statusFilter, setStatusFilter] = useState('all')
  const [projectFilter, setProjectFilter] = useState('all')
  const [typeFilter, setTypeFilter] = useState('all')
  const searchRef = useRef<HTMLInputElement>(null)

  // Selection
  const [selectedUUID, setSelectedUUID] = useState<string | null>(null)
  const [expandedUUID, setExpandedUUID] = useState<string | null>(null)

  // Group collapse state
  const [collapsedGroups, setCollapsedGroups] = useState<Set<string>>(loadCollapsed)

  // Command palette
  const [commandOpen, setCommandOpen] = useState(false)

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
      await fetch(`/api/check-definitions/${uuid}/toggle`, {
        method: 'PATCH',
        credentials: 'include',
      })
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
      setCommandOpen((prev) => !prev)
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
          onOpenCommandPalette={() => setCommandOpen(true)}
        />

        <main className="mx-auto max-w-[1600px] px-4 py-4 space-y-4">
          <MetricsRow stats={stats} />

          <CheckList
            groups={groups}
            collapsedGroups={collapsedGroups}
            onToggleGroup={toggleGroup}
            selectedUUID={selectedUUID}
            expandedUUID={expandedUUID}
            onSelectCheck={handleSelectCheck}
            onToggleCheck={handleToggleCheck}
          />

          <EventLog entries={entries} />
        </main>

        <StatusBar wsStatus={wsStatus} />

        {/* Command Palette */}
        <CommandDialog open={commandOpen} onOpenChange={setCommandOpen}>
          <CommandInput placeholder="Type a command or search..." />
          <CommandList>
            <CommandEmpty>No results found.</CommandEmpty>
            <CommandGroup heading="Navigation">
              <CommandItem onSelect={() => { searchRef.current?.focus(); setCommandOpen(false) }}>
                Focus Search
                <CommandShortcut>/</CommandShortcut>
              </CommandItem>
            </CommandGroup>
            <CommandGroup heading="Shortcuts">
              <CommandItem disabled>
                Navigate Down
                <CommandShortcut>j</CommandShortcut>
              </CommandItem>
              <CommandItem disabled>
                Navigate Up
                <CommandShortcut>k</CommandShortcut>
              </CommandItem>
              <CommandItem disabled>
                Expand/Collapse Check
                <CommandShortcut>Enter</CommandShortcut>
              </CommandItem>
              <CommandItem disabled>
                Close Details
                <CommandShortcut>Esc</CommandShortcut>
              </CommandItem>
              <CommandItem disabled>
                Toggle Group
                <CommandShortcut>g</CommandShortcut>
              </CommandItem>
            </CommandGroup>
            <CommandGroup heading="Filters">
              <CommandItem onSelect={() => { setStatusFilter('unhealthy'); setCommandOpen(false) }}>
                Show Only Failing
              </CommandItem>
              <CommandItem onSelect={() => { setStatusFilter('all'); setSearch(''); setProjectFilter('all'); setTypeFilter('all'); setCommandOpen(false) }}>
                Clear All Filters
              </CommandItem>
            </CommandGroup>
          </CommandList>
        </CommandDialog>
      </div>
    </TooltipProvider>
  )
}
