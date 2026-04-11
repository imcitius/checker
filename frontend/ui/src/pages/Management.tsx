import { useState, useEffect, useCallback, useMemo } from 'react'
import { useSearchParams } from 'react-router-dom'
import { Layers } from 'lucide-react'
import { api, type CheckDefinition, type AlertChannel, type CheckDefaults } from '@/lib/api'
import { type Check } from '@/lib/websocket'
import { Button } from '@/components/ui/button'
import { StatusDot } from '@/components/StatusDot'
import { cn } from '@/lib/utils'
import { CHANNEL_TYPES } from '@/lib/channels'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from '@/components/ui/dialog'
import { Switch } from '@/components/ui/switch'
import { Badge } from '@/components/ui/badge'
import {
  Plus, Pencil, Trash2, RefreshCw, Upload, Download,
  ArrowUp, ArrowDown, ArrowUpDown, Copy, Power, PowerOff,
  CheckSquare, Square, MinusSquare, Clock, X,
  ChevronRight, ChevronDown, FolderOpen, BellOff, Bell,
} from 'lucide-react'
import { TooltipProvider } from '@/components/ui/tooltip'
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip'
import { TopBar } from '@/components/TopBar'
import { StatusBar } from '@/components/StatusBar'
import { useTopBarConfig } from '@/lib/topbar-context'
import { useChecks } from '@/hooks/useChecks'
import { useRef } from 'react'
import { ImportDialog } from '@/components/ImportDialog'
import { CheckEditDrawer } from '@/components/CheckEditDrawer'
import { Input } from '@/components/ui/input'
import { api as apiClient } from '@/lib/api'
import { toast } from 'sonner'

type SortColumn = 'name' | 'type' | 'group' | 'duration' | 'enabled' | 'status'
type SortDirection = 'asc' | 'desc'

const VALID_SORT_COLUMNS: readonly string[] = ['name', 'type', 'group', 'duration', 'enabled', 'status'] as const
const VALID_SORT_DIRECTIONS: readonly string[] = ['asc', 'desc'] as const
const COLLAPSED_KEY = 'checker-manage-collapsed'
const SORT_KEY = 'checker-manage-sort'

function parseSortColumn(value: string | null): SortColumn | null {
  if (value && VALID_SORT_COLUMNS.includes(value)) return value as SortColumn
  return null
}

function parseSortDirection(value: string | null): SortDirection {
  if (value && VALID_SORT_DIRECTIONS.includes(value)) return value as SortDirection
  return 'asc'
}

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

function loadSort(): { column: SortColumn | null; direction: SortDirection } {
  try {
    const val = localStorage.getItem(SORT_KEY)
    if (!val) return { column: null, direction: 'asc' }
    const parsed = JSON.parse(val)
    return {
      column: parseSortColumn(parsed.column),
      direction: parseSortDirection(parsed.direction),
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

const EMPTY_FORM: Partial<CheckDefinition> = {
  name: '',
  project: '',
  group_name: '',
  type: 'http',
  description: '',
  enabled: true,
  duration: '1m',
  url: '',
  timeout: '10s',
  host: '',
  port: 0,
}

interface SubGroup {
  group: string
  checks: CheckDefinition[]
  enabledCount: number
  disabledCount: number
}

interface ProjectGroup {
  project: string
  subGroups: SubGroup[]
  checks: CheckDefinition[]  // all checks flat (for selection helpers)
  enabledCount: number
  disabledCount: number
}

// Returns a numeric priority for sorting by status (lower = more urgent)
function getStatusPriority(def: CheckDefinition, liveCheck: Check | undefined): number {
  if (!def.enabled) return 5
  if (!liveCheck) return 2 // pending (no live data yet)
  if (liveCheck.IsSilenced) return liveCheck.LastResult ? 3 : 1
  if (!liveCheck.LastResult) return 0 // failing
  return 4 // ok
}

// Returns label + dot props for a check definition given live websocket data
function getCheckStatus(def: CheckDefinition, liveCheck: Check | undefined) {
  if (!def.enabled) {
    return { label: 'Disabled', healthy: false, enabled: false, silenced: false }
  }
  if (!liveCheck) {
    return { label: 'Pending', healthy: false, enabled: true, silenced: false, pending: true }
  }
  if (liveCheck.IsSilenced) {
    return { label: 'Silenced', healthy: liveCheck.LastResult, enabled: true, silenced: true }
  }
  if (liveCheck.LastResult) {
    return { label: 'OK', healthy: true, enabled: true, silenced: false }
  }
  return { label: 'Failing', healthy: false, enabled: true, silenced: false }
}

function AlertChannelBadges({ channels, defaultChannels }: { channels?: string[]; defaultChannels?: string[] }) {
  const hasExplicit = channels && channels.length > 0
  const hasDefault = defaultChannels && defaultChannels.length > 0

  if (!hasExplicit && !hasDefault) {
    return (
      <span className="text-[10px] text-amber-500 font-medium whitespace-nowrap">No alerts</span>
    )
  }

  // Show explicit channels if any, otherwise show default channels dimmed
  const displayChannels = hasExplicit ? channels : defaultChannels!
  const isDefault = !hasExplicit

  return (
    <div className="flex flex-wrap gap-1">
      {displayChannels.map((ch) => {
        const meta = CHANNEL_TYPES.find((ct) => ct.value === ch)
        const label = meta?.label ?? ch
        const color = meta?.color ?? 'bg-gray-500'
        return (
          <Tooltip key={ch}>
            <TooltipTrigger asChild>
              <span
                className={cn(
                  'inline-flex items-center px-1.5 py-0.5 rounded text-[10px] font-medium leading-none',
                  isDefault ? 'opacity-60 text-white' : 'text-white',
                  color
                )}
              >
                {label}{isDefault ? ' (default)' : ''}
              </span>
            </TooltipTrigger>
            <TooltipContent>{isDefault ? `Default channel: ${ch}` : ch}</TooltipContent>
          </Tooltip>
        )
      })}
    </div>
  )
}

function RegionBadges({ regions, totalRegions }: { regions?: string[]; totalRegions: number }) {
  if (totalRegions === 0) return null // no platform regions configured
  const hasExplicit = regions && regions.length > 0
  if (!hasExplicit) {
    return (
      <span className="text-[10px] text-muted-foreground whitespace-nowrap">(all regions)</span>
    )
  }
  return (
    <div className="flex flex-wrap gap-1">
      {regions.map((r) => (
        <span
          key={r}
          className="inline-flex items-center px-1.5 py-0.5 rounded text-[10px] font-medium leading-none bg-blue-500 text-white"
        >
          {r}
        </span>
      ))}
    </div>
  )
}

export interface ManagementProps {
  /** Extra columns rendered in the management table (not yet implemented -- placeholder for cloud) */
  extraColumns?: unknown[]
  /** Extra bulk actions available in the bulk action menu (not yet implemented -- placeholder for cloud) */
  bulkActions?: unknown[]
  /** Callback when a check is selected */
  onCheckSelect?: (uuid: string) => void
}

export function Management(_props: ManagementProps = {}) {
  const topBarConfig = useTopBarConfig()
  const { wsStatus, checksMap } = useChecks()
  const [definitions, setDefinitions] = useState<CheckDefinition[]>([])
  const [loading, setLoading] = useState(true)
  const [search, setSearch] = useState('')
  const [typeFilter, setTypeFilter] = useState('all')
  const [projectFilter, setProjectFilter] = useState('all')
  const [statusFilter, setStatusFilter] = useState('all')

  // Sort state — persisted in localStorage
  const [searchParams, setSearchParams] = useSearchParams()
  const [sortState, setSortState] = useState(loadSort)
  const sortColumn = sortState.column
  const sortDirection = sortState.direction

  // Collapse state
  const [collapsedGroups, setCollapsedGroups] = useState<Set<string>>(loadCollapsed)

  // Dialog state
  const [editDialogOpen, setEditDialogOpen] = useState(false)
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false)
  const [bulkDeleteDialogOpen, setBulkDeleteDialogOpen] = useState(false)
  const [editingCheck, setEditingCheck] = useState<Partial<CheckDefinition>>(EMPTY_FORM)
  const [deletingUUID, setDeletingUUID] = useState<string | null>(null)
  const [saving, setSaving] = useState(false)

  const [importDialogOpen, setImportDialogOpen] = useState(false)
  const [bulkMaintenanceDialogOpen, setBulkMaintenanceDialogOpen] = useState(false)
  const [bulkMaintenanceUntil, setBulkMaintenanceUntil] = useState('')
  const [bulkAlertChannelsDialogOpen, setBulkAlertChannelsDialogOpen] = useState(false)
  const [bulkAlertAction, setBulkAlertAction] = useState<'add' | 'remove' | 'replace'>('add')
  const [bulkAlertSelectedChannels, setBulkAlertSelectedChannels] = useState<Set<string>>(new Set())
  const [allAlertChannels, setAllAlertChannels] = useState<AlertChannel[]>([])
  const [checkDefaults, setCheckDefaults] = useState<CheckDefaults | null>(null)
  const [tenantRegionCount, setTenantRegionCount] = useState(0)

  // Bulk selection
  const [selectedUUIDs, setSelectedUUIDs] = useState<Set<string>>(new Set())
  const [bulkActing, setBulkActing] = useState(false)

  const searchRef = useRef<HTMLInputElement>(null)

  // Metadata
  const [projects, setProjects] = useState<string[]>([])
  const [checkTypes, setCheckTypes] = useState<string[]>([])

  const fetchData = useCallback(async () => {
    setLoading(true)
    try {
      const [defs, projs, types, channels, defaults, regions] = await Promise.all([
        api.getChecks(),
        api.getProjects().catch(() => [] as string[]),
        api.getCheckTypes().catch(() => [] as string[]),
        api.getAlertChannels().catch(() => [] as AlertChannel[]),
        api.getCheckDefaults().catch(() => null as CheckDefaults | null),
        api.getPlatformRegions().catch(() => null),
      ])
      setDefinitions(defs || [])
      setProjects(projs || [])
      setCheckTypes(types || [])
      setAllAlertChannels(channels || [])
      setCheckDefaults(defaults)
      setTenantRegionCount(regions?.regions?.length ?? 0)
      // Clear selection of items that no longer exist
      setSelectedUUIDs((prev) => {
        const validUUIDs = new Set((defs || []).map((d) => d.uuid))
        const next = new Set<string>()
        for (const uuid of prev) {
          if (validUUIDs.has(uuid)) next.add(uuid)
        }
        return next
      })
    } catch (err) {
      console.error('Failed to fetch data:', err)
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    fetchData()
  }, [fetchData])

  // Handle command-palette deep-link actions (?action=create / ?action=import)
  useEffect(() => {
    const action = searchParams.get('action')
    if (action === 'create') {
      setEditingCheck({ ...EMPTY_FORM })
      setEditDialogOpen(true)
      setSearchParams((prev) => {
        const next = new URLSearchParams(prev)
        next.delete('action')
        return next
      }, { replace: true })
    } else if (action === 'import') {
      setImportDialogOpen(true)
      setSearchParams((prev) => {
        const next = new URLSearchParams(prev)
        next.delete('action')
        return next
      }, { replace: true })
    }
  }, [searchParams, setSearchParams])

  const filtered = definitions.filter((d) => {
    if (search) {
      const q = search.toLowerCase()
      if (
        !d.name.toLowerCase().includes(q) &&
        !d.uuid.toLowerCase().includes(q) &&
        !d.project.toLowerCase().includes(q) &&
        !(d.group_name || '').toLowerCase().includes(q) &&
        !(d.url || '').toLowerCase().includes(q) &&
        !(d.host || '').toLowerCase().includes(q) &&
        !(d.domain || '').toLowerCase().includes(q) &&
        !(d.mongodb_uri || '').toLowerCase().includes(q)
      )
        return false
    }
    if (typeFilter !== 'all' && d.type !== typeFilter) return false
    if (projectFilter !== 'all' && d.project !== projectFilter) return false
    if (statusFilter !== 'all') {
      if (statusFilter === 'enabled' && !d.enabled) return false
      if (statusFilter === 'disabled' && d.enabled) return false
    }
    return true
  })

  const handleSort = (column: SortColumn) => {
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
  }

  const sortChecks = useCallback((checks: CheckDefinition[]) => {
    if (!sortColumn) return checks
    return [...checks].sort((a, b) => {
      if (sortColumn === 'status') {
        const aPriority = getStatusPriority(a, checksMap.get(a.uuid))
        const bPriority = getStatusPriority(b, checksMap.get(b.uuid))
        const cmp = aPriority - bPriority
        return sortDirection === 'asc' ? cmp : -cmp
      }
      let aVal: string | boolean
      let bVal: string | boolean
      switch (sortColumn) {
        case 'name':
          aVal = a.name.toLowerCase()
          bVal = b.name.toLowerCase()
          break
        case 'group':
          aVal = (a.group_name || '').toLowerCase()
          bVal = (b.group_name || '').toLowerCase()
          break
        case 'type':
          aVal = a.type.toLowerCase()
          bVal = b.type.toLowerCase()
          break
        case 'duration':
          aVal = a.duration.toLowerCase()
          bVal = b.duration.toLowerCase()
          break
        case 'enabled':
          aVal = a.enabled
          bVal = b.enabled
          break
        default:
          return 0
      }
      if (typeof aVal === 'boolean' && typeof bVal === 'boolean') {
        const cmp = aVal === bVal ? 0 : aVal ? -1 : 1
        return sortDirection === 'asc' ? cmp : -cmp
      }
      const cmp = (aVal as string).localeCompare(bVal as string)
      return sortDirection === 'asc' ? cmp : -cmp
    })
  }, [sortColumn, sortDirection, checksMap])

  // Two-level grouping: Project → Group (group_name)
  const groups: ProjectGroup[] = useMemo(() => {
    const projectMap = new Map<string, Map<string, CheckDefinition[]>>()
    for (const def of filtered) {
      const project = def.project || 'default'
      const group = def.group_name || ''
      if (!projectMap.has(project)) projectMap.set(project, new Map())
      const groupMap = projectMap.get(project)!
      if (!groupMap.has(group)) groupMap.set(group, [])
      groupMap.get(group)!.push(def)
    }
    return Array.from(projectMap.entries())
      .map(([project, groupMap]) => {
        const subGroups: SubGroup[] = Array.from(groupMap.entries())
          .map(([group, checks]) => ({
            group,
            checks: sortChecks(checks),
            enabledCount: checks.filter((c) => c.enabled).length,
            disabledCount: checks.filter((c) => !c.enabled).length,
          }))
          .sort((a, b) => a.group.localeCompare(b.group))
        const allChecks = subGroups.flatMap((sg) => sg.checks)
        return {
          project,
          subGroups,
          checks: allChecks,
          enabledCount: allChecks.filter((c) => c.enabled).length,
          disabledCount: allChecks.filter((c) => !c.enabled).length,
        }
      })
      .sort((a, b) => a.project.localeCompare(b.project))
  }, [filtered, sortChecks])

  // Flat sorted list for bulk selection helpers
  const allVisible = useMemo(() => groups.flatMap((g) => g.checks), [groups])

  // Unique project and group names for combobox suggestions
  const uniqueProjects = useMemo(() => {
    const set = new Set(definitions.map((d) => d.project).filter(Boolean))
    return [...set].sort()
  }, [definitions])
  const uniqueGroups = useMemo(() => {
    const set = new Set(definitions.map((d) => d.group_name).filter(Boolean))
    return [...set].sort()
  }, [definitions])


  const SortIcon = ({ column }: { column: SortColumn }) => {
    if (sortColumn !== column) return <ArrowUpDown className="h-3 w-3 ml-1 opacity-40" />
    if (sortDirection === 'asc') return <ArrowUp className="h-3 w-3 ml-1" />
    return <ArrowDown className="h-3 w-3 ml-1" />
  }

  const toggleGroup = useCallback((project: string) => {
    setCollapsedGroups((prev) => {
      const next = new Set(prev)
      if (next.has(project)) next.delete(project)
      else next.add(project)
      saveCollapsed(next)
      return next
    })
  }, [])

  const handleCreate = () => {
    setEditingCheck({ ...EMPTY_FORM })
    setEditDialogOpen(true)
  }

  const handleEdit = (def: CheckDefinition) => {
    setEditingCheck({ ...def })
    setEditDialogOpen(true)
  }

  const handleClone = (def: CheckDefinition) => {
    const cloned = { ...def }
    delete (cloned as Partial<CheckDefinition> & { uuid?: string }).uuid
    delete (cloned as Partial<CheckDefinition> & { id?: string }).id
    cloned.name = `${def.name} (copy)`
    setEditingCheck(cloned)
    setEditDialogOpen(true)
  }

  const handleSave = async () => {
    setSaving(true)
    try {
      if (editingCheck.uuid) {
        await api.updateCheck(editingCheck.uuid, editingCheck)
      } else {
        await api.createCheck(editingCheck)
      }
      setEditDialogOpen(false)
      fetchData()
    } catch (err) {
      console.error('Failed to save:', err)
    } finally {
      setSaving(false)
    }
  }

  const handleDelete = async () => {
    if (!deletingUUID) return
    try {
      await api.deleteCheck(deletingUUID)
      setDeleteDialogOpen(false)
      setDeletingUUID(null)
      fetchData()
    } catch (err) {
      console.error('Failed to delete:', err)
    }
  }

  const handleToggle = async (uuid: string) => {
    try {
      await api.toggleCheck(uuid)
      fetchData()
    } catch (err) {
      console.error('Failed to toggle:', err)
    }
  }

  const handleExport = async () => {
    try {
      const yamlContent = await apiClient.exportChecks(
        projectFilter !== 'all' ? projectFilter : undefined
      )
      const blob = new Blob([yamlContent], { type: 'application/x-yaml' })
      const url = URL.createObjectURL(blob)
      const a = document.createElement('a')
      a.href = url
      a.download = 'checks.yaml'
      document.body.appendChild(a)
      a.click()
      document.body.removeChild(a)
      URL.revokeObjectURL(url)
    } catch (err) {
      console.error('Failed to export:', err)
    }
  }

  // Bulk selection helpers
  const filteredUUIDs = useMemo(() => new Set(allVisible.map((d) => d.uuid)), [allVisible])
  const selectedInView = useMemo(
    () => new Set([...selectedUUIDs].filter((uuid) => filteredUUIDs.has(uuid))),
    [selectedUUIDs, filteredUUIDs]
  )

  const allSelected = allVisible.length > 0 && selectedInView.size === allVisible.length
  const someSelected = selectedInView.size > 0 && !allSelected

  const toggleSelectAll = () => {
    if (allSelected) {
      setSelectedUUIDs((prev) => {
        const next = new Set(prev)
        for (const uuid of filteredUUIDs) next.delete(uuid)
        return next
      })
    } else {
      setSelectedUUIDs((prev) => {
        const next = new Set(prev)
        for (const uuid of filteredUUIDs) next.add(uuid)
        return next
      })
    }
  }

  const toggleSelect = (uuid: string) => {
    setSelectedUUIDs((prev) => {
      const next = new Set(prev)
      if (next.has(uuid)) next.delete(uuid)
      else next.add(uuid)
      return next
    })
  }

  const toggleSelectGroup = (checks: CheckDefinition[]) => {
    const uuids = checks.map((c) => c.uuid)
    const allSelected = uuids.every((uuid) => selectedUUIDs.has(uuid))
    setSelectedUUIDs((prev) => {
      const next = new Set(prev)
      for (const uuid of uuids) {
        if (allSelected) next.delete(uuid)
        else next.add(uuid)
      }
      return next
    })
  }

  // Bulk actions
  const handleBulkEnable = async () => {
    setBulkActing(true)
    try {
      const result = await api.bulkEnable([...selectedInView])
      toast.success(`Enabled ${result.count} checks`)
      setSelectedUUIDs(new Set())
      fetchData()
    } catch (err) {
      console.error('Bulk enable failed:', err)
      toast.error('Failed to enable checks')
    } finally {
      setBulkActing(false)
    }
  }

  const handleBulkDisable = async () => {
    setBulkActing(true)
    try {
      const result = await api.bulkDisable([...selectedInView])
      toast.success(`Disabled ${result.count} checks`)
      setSelectedUUIDs(new Set())
      fetchData()
    } catch (err) {
      console.error('Bulk disable failed:', err)
      toast.error('Failed to disable checks')
    } finally {
      setBulkActing(false)
    }
  }

  const handleBulkDelete = async () => {
    setBulkActing(true)
    try {
      const result = await api.bulkDelete([...selectedInView])
      toast.success(`Deleted ${result.count} checks`)
      setSelectedUUIDs(new Set())
      setBulkDeleteDialogOpen(false)
      fetchData()
    } catch (err) {
      console.error('Bulk delete failed:', err)
      toast.error('Failed to delete checks')
    } finally {
      setBulkActing(false)
    }
  }

  const handleBulkMaintenance = async () => {
    if (!bulkMaintenanceUntil) return
    setBulkActing(true)
    try {
      const until = new Date(bulkMaintenanceUntil).toISOString()
      await Promise.all([...selectedInView].map((uuid) => api.setMaintenance(uuid, until)))
      toast.success(`Set maintenance on ${selectedInView.size} checks`)
      setSelectedUUIDs(new Set())
      setBulkMaintenanceDialogOpen(false)
      setBulkMaintenanceUntil('')
      fetchData()
    } catch (err) {
      console.error('Bulk maintenance failed:', err)
      toast.error('Failed to set maintenance on some checks')
    } finally {
      setBulkActing(false)
    }
  }

  // All alert channels available system-wide (fetched from API)
  const availableChannelNames = useMemo(
    () => allAlertChannels.map((ch) => ch.name).sort(),
    [allAlertChannels]
  )

  const handleBulkAlertChannels = async () => {
    if (bulkAlertSelectedChannels.size === 0) return
    setBulkActing(true)
    try {
      const result = await api.bulkAlertChannels(
        [...selectedInView],
        bulkAlertAction,
        [...bulkAlertSelectedChannels]
      )
      const actionLabel = bulkAlertAction === 'add' ? 'Added channels to' : bulkAlertAction === 'remove' ? 'Removed channels from' : 'Replaced channels on'
      toast.success(`${actionLabel} ${result.count} checks`)
      setSelectedUUIDs(new Set())
      setBulkAlertChannelsDialogOpen(false)
      setBulkAlertSelectedChannels(new Set())
      fetchData()
    } catch (err) {
      console.error('Bulk alert channels failed:', err)
      toast.error('Failed to update alert channels')
    } finally {
      setBulkActing(false)
    }
  }

  const renderCheckRow = (def: CheckDefinition) => {
    const isSelected = selectedUUIDs.has(def.uuid)
    const liveCheck = checksMap.get(def.uuid)
    const status = getCheckStatus(def, liveCheck)
    return (
      <tr
        key={def.uuid}
        className={`border-b border-border/50 transition-colors ${
          isSelected
            ? 'bg-primary/5 hover:bg-primary/10'
            : 'hover:bg-muted/30'
        }`}
      >
        <td className="px-3 py-2">
          <button
            onClick={() => toggleSelect(def.uuid)}
            className="flex items-center justify-center text-muted-foreground hover:text-foreground transition-colors"
          >
            {isSelected ? (
              <CheckSquare className="h-4 w-4 text-primary" />
            ) : (
              <Square className="h-4 w-4" />
            )}
          </button>
        </td>
        <td className="px-3 py-2">
          <div className="flex items-center gap-1.5">
            {status.pending ? (
              <span className="inline-block h-2.5 w-2.5 rounded-full bg-muted-foreground/40 shrink-0" />
            ) : (
              <StatusDot
                healthy={status.healthy}
                enabled={status.enabled}
                silenced={status.silenced}
              />
            )}
            <span
              className={cn(
                'font-mono text-xs whitespace-nowrap',
                !def.enabled
                  ? 'text-muted-foreground'
                  : status.silenced
                    ? 'text-warning'
                    : status.pending
                      ? 'text-muted-foreground/60'
                      : status.healthy
                        ? 'text-healthy'
                        : 'text-unhealthy font-semibold'
              )}
            >
              {status.label}
            </span>
          </div>
        </td>
        <td className="px-3 py-2 overflow-hidden">
          <div className="font-medium break-words">{def.name}</div>
          <div className="font-mono text-[10px] text-muted-foreground truncate">{def.uuid}</div>
        </td>
        <td className="px-3 py-2 overflow-hidden">
          <Tooltip>
            <TooltipTrigger asChild>
              <span className="font-mono text-xs text-muted-foreground truncate block">
                {def.url || def.host || def.domain || def.mongodb_uri || '—'}
              </span>
            </TooltipTrigger>
            <TooltipContent>{def.url || def.host || def.domain || def.mongodb_uri || 'No target'}</TooltipContent>
          </Tooltip>
        </td>
        <td className="px-3 py-2">
          <Badge variant="secondary" className="text-[10px]">
            {def.type}
          </Badge>
        </td>
        <td className="px-3 py-2">
          <AlertChannelBadges channels={def.alert_channels} defaultChannels={checkDefaults?.alert_channels} />
        </td>
        {tenantRegionCount > 0 && (
          <td className="px-3 py-2">
            <RegionBadges regions={def.target_regions} totalRegions={tenantRegionCount} />
          </td>
        )}
        <td className="px-3 py-2 font-mono text-xs text-muted-foreground">{def.duration}</td>
        <td className="px-3 py-2">
          <Switch
            checked={def.enabled}
            onCheckedChange={() => handleToggle(def.uuid)}
            className="scale-75"
          />
        </td>
        <td className="px-3 py-2 text-right">
          <div className="flex items-center justify-end gap-1">
            {(!def.alert_channels || def.alert_channels.length === 0) && (!checkDefaults?.alert_channels || checkDefaults.alert_channels.length === 0) && (
              <Tooltip>
                <TooltipTrigger asChild>
                  <Button
                    variant="ghost"
                    size="icon"
                    className="h-7 w-7 text-amber-500 hover:text-amber-600"
                    onClick={() => handleEdit(def)}
                  >
                    <BellOff className="h-3.5 w-3.5" />
                  </Button>
                </TooltipTrigger>
                <TooltipContent>No alert channels — notifications won't be sent</TooltipContent>
              </Tooltip>
            )}
            <Tooltip>
              <TooltipTrigger asChild>
                <Button variant="ghost" size="icon" className="h-7 w-7" onClick={() => handleClone(def)}>
                  <Copy className="h-3.5 w-3.5" />
                </Button>
              </TooltipTrigger>
              <TooltipContent>Clone check</TooltipContent>
            </Tooltip>
            <Tooltip>
              <TooltipTrigger asChild>
                <Button variant="ghost" size="icon" className="h-7 w-7" onClick={() => handleEdit(def)}>
                  <Pencil className="h-3.5 w-3.5" />
                </Button>
              </TooltipTrigger>
              <TooltipContent>Edit check</TooltipContent>
            </Tooltip>
            <Tooltip>
              <TooltipTrigger asChild>
                <Button
                  variant="ghost"
                  size="icon"
                  className="h-7 w-7 text-unhealthy hover:text-unhealthy"
                  onClick={() => {
                    setDeletingUUID(def.uuid)
                    setDeleteDialogOpen(true)
                  }}
                >
                  <Trash2 className="h-3.5 w-3.5" />
                </Button>
              </TooltipTrigger>
              <TooltipContent>Delete check</TooltipContent>
            </Tooltip>
          </div>
        </td>
      </tr>
    )
  }

  return (
    <TooltipProvider delayDuration={300}>
      <div className="min-h-screen bg-background pb-8">
        <TopBar
          {...topBarConfig}
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
          {/* Actions bar */}
          <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-2">
            <div className="flex items-center gap-2">
              <h2 className="text-lg font-semibold">Check Definitions</h2>
              <Badge variant="secondary" className="text-xs">
                {allVisible.length} check{allVisible.length !== 1 ? 's' : ''}
              </Badge>
              {selectedInView.size > 0 && (
                <Badge variant="info" className="text-xs">
                  {selectedInView.size} selected
                </Badge>
              )}
            </div>
            <div className="flex items-center gap-2 flex-wrap">
              {/* Bulk actions */}
              {selectedInView.size > 0 && (
                <div className="flex items-center gap-1 mr-2 sm:border-r sm:pr-3 border-border">
                  <Tooltip>
                    <TooltipTrigger asChild>
                      <Button
                        variant="outline"
                        size="sm"
                        onClick={handleBulkEnable}
                        disabled={bulkActing}
                        className="min-h-[44px]"
                      >
                        <Power className="h-4 w-4 mr-1 text-healthy" />
                        Enable
                      </Button>
                    </TooltipTrigger>
                    <TooltipContent>Enable {selectedInView.size} selected checks</TooltipContent>
                  </Tooltip>
                  <Tooltip>
                    <TooltipTrigger asChild>
                      <Button
                        variant="outline"
                        size="sm"
                        onClick={handleBulkDisable}
                        disabled={bulkActing}
                        className="min-h-[44px]"
                      >
                        <PowerOff className="h-4 w-4 mr-1 text-warning" />
                        Disable
                      </Button>
                    </TooltipTrigger>
                    <TooltipContent>Disable {selectedInView.size} selected checks</TooltipContent>
                  </Tooltip>
                  <Tooltip>
                    <TooltipTrigger asChild>
                      <Button
                        variant="outline"
                        size="sm"
                        onClick={() => setBulkDeleteDialogOpen(true)}
                        disabled={bulkActing}
                        className="text-unhealthy hover:text-unhealthy min-h-[44px]"
                      >
                        <Trash2 className="h-4 w-4 mr-1" />
                        Delete
                      </Button>
                    </TooltipTrigger>
                    <TooltipContent>Delete {selectedInView.size} selected checks</TooltipContent>
                  </Tooltip>
                </div>
              )}
              <Button variant="outline" size="sm" onClick={fetchData} disabled={loading} className="min-h-[44px]">
                <RefreshCw className={`h-4 w-4 mr-1 ${loading ? 'animate-spin' : ''}`} />
                Refresh
              </Button>
              <Button variant="outline" size="sm" onClick={handleExport} className="min-h-[44px]">
                <Download className="h-4 w-4 mr-1" />
                <span className="hidden sm:inline">Export</span>
              </Button>
              <Button variant="outline" size="sm" onClick={() => setImportDialogOpen(true)} className="min-h-[44px]">
                <Upload className="h-4 w-4 mr-1" />
                <span className="hidden sm:inline">Import YAML</span>
              </Button>
              <Button size="sm" onClick={handleCreate} className="min-h-[44px]">
                <Plus className="h-4 w-4 mr-1" />
                <span className="hidden sm:inline">New Check</span>
              </Button>
            </div>
          </div>

          {/* Desktop table — hidden on mobile */}
          <div className="hidden sm:block space-y-3">
            {loading ? (
              <div className="rounded-lg border bg-card p-8 text-center text-muted-foreground">
                Loading...
              </div>
            ) : groups.length === 0 ? (
              <div className="rounded-lg border bg-card p-8 text-center text-muted-foreground">
                No check definitions found
              </div>
            ) : (
              groups.map((group) => {
                const projectKey = `p:${group.project}`
                const isProjectCollapsed = collapsedGroups.has(projectKey)
                const projectUUIDs = group.checks.map((c) => c.uuid)
                const allProjectSelected = projectUUIDs.length > 0 && projectUUIDs.every((uuid) => selectedUUIDs.has(uuid))
                const someProjectSelected = !allProjectSelected && projectUUIDs.some((uuid) => selectedUUIDs.has(uuid))
                const showSubGroupHeaders = group.subGroups.length > 1 || (group.subGroups.length === 1 && group.subGroups[0].group !== '')

                return (
                  <div key={group.project} className="rounded-lg border bg-card overflow-hidden">
                    {/* Project header */}
                    <button
                      onClick={() => toggleGroup(projectKey)}
                      className="w-full flex items-center gap-2 px-3 py-2.5 bg-muted/50 hover:bg-muted/80 transition-colors text-left"
                    >
                      <div
                        className="flex items-center justify-center text-muted-foreground hover:text-foreground shrink-0"
                        onClick={(e) => {
                          e.stopPropagation()
                          toggleSelectGroup(group.checks)
                        }}
                      >
                        {allProjectSelected ? (
                          <CheckSquare className="h-4 w-4 text-primary" />
                        ) : someProjectSelected ? (
                          <MinusSquare className="h-4 w-4 text-primary" />
                        ) : (
                          <Square className="h-4 w-4" />
                        )}
                      </div>
                      {isProjectCollapsed ? (
                        <ChevronRight className="h-4 w-4 text-muted-foreground shrink-0" />
                      ) : (
                        <ChevronDown className="h-4 w-4 text-muted-foreground shrink-0" />
                      )}
                      <FolderOpen className="h-4 w-4 text-muted-foreground shrink-0" />
                      <span className="font-semibold text-sm">{group.project}</span>
                      <Badge variant="secondary" className="text-[10px] ml-1">
                        {group.checks.length}
                      </Badge>
                      <span className="text-xs text-muted-foreground ml-auto">
                        {group.enabledCount} enabled
                        {group.disabledCount > 0 && (
                          <span className="text-muted-foreground/60"> / {group.disabledCount} disabled</span>
                        )}
                      </span>
                    </button>

                    {/* Project content */}
                    {!isProjectCollapsed && (
                      <div>
                        {group.subGroups.map((sg) => {
                          const subGroupKey = `g:${group.project}/${sg.group}`
                          const isSubGroupCollapsed = collapsedGroups.has(subGroupKey)
                          const sgUUIDs = sg.checks.map((c) => c.uuid)
                          const allSgSelected = sgUUIDs.length > 0 && sgUUIDs.every((uuid) => selectedUUIDs.has(uuid))
                          const someSgSelected = !allSgSelected && sgUUIDs.some((uuid) => selectedUUIDs.has(uuid))

                          return (
                            <div key={sg.group} className={showSubGroupHeaders ? 'border-t border-border/50' : ''}>
                              {/* Sub-group header (only if project has multiple groups) */}
                              {showSubGroupHeaders && (
                                <button
                                  onClick={() => toggleGroup(subGroupKey)}
                                  className="w-full flex items-center gap-2 px-3 py-1.5 pl-8 bg-muted/25 hover:bg-muted/40 transition-colors text-left"
                                >
                                  <div
                                    className="flex items-center justify-center text-muted-foreground hover:text-foreground shrink-0"
                                    onClick={(e) => {
                                      e.stopPropagation()
                                      toggleSelectGroup(sg.checks)
                                    }}
                                  >
                                    {allSgSelected ? (
                                      <CheckSquare className="h-3.5 w-3.5 text-primary" />
                                    ) : someSgSelected ? (
                                      <MinusSquare className="h-3.5 w-3.5 text-primary" />
                                    ) : (
                                      <Square className="h-3.5 w-3.5" />
                                    )}
                                  </div>
                                  {isSubGroupCollapsed ? (
                                    <ChevronRight className="h-3.5 w-3.5 text-muted-foreground shrink-0" />
                                  ) : (
                                    <ChevronDown className="h-3.5 w-3.5 text-muted-foreground shrink-0" />
                                  )}
                                  <Layers className="h-3.5 w-3.5 text-muted-foreground shrink-0" />
                                  <span className="text-xs font-medium text-muted-foreground">{sg.group || '(no group)'}</span>
                                  <Badge variant="secondary" className="text-[10px] ml-1">
                                    {sg.checks.length}
                                  </Badge>
                                  <span className="text-[11px] text-muted-foreground/60 ml-auto">
                                    {sg.enabledCount} on{sg.disabledCount > 0 && ` / ${sg.disabledCount} off`}
                                  </span>
                                </button>
                              )}

                              {/* Checks table */}
                              {!(showSubGroupHeaders && isSubGroupCollapsed) && (
                                <div className={showSubGroupHeaders ? 'ml-4 border-l-2 border-border/40 overflow-x-auto' : 'overflow-x-auto'}>
                                  <table className="w-full text-sm table-fixed">
                                    <colgroup>
                                      <col className="w-10" />
                                      <col className="w-[100px]" />
                                      <col className="w-[200px]" />
                                      <col />
                                      <col className="w-[70px]" />
                                      <col className="w-[160px]" />
                                      <col className="w-[90px]" />
                                      <col className="w-[70px]" />
                                      <col className="w-[100px]" />
                                    </colgroup>
                                    <thead>
                                      <tr className="border-b border-t bg-muted/30 text-muted-foreground text-xs">
                                        <th className="px-3 py-1.5">
                                          <button
                                            onClick={() => toggleSelectGroup(sg.checks)}
                                            className="flex items-center justify-center text-muted-foreground hover:text-foreground transition-colors"
                                          >
                                            {allSgSelected ? (
                                              <CheckSquare className="h-3.5 w-3.5" />
                                            ) : someSgSelected ? (
                                              <MinusSquare className="h-3.5 w-3.5" />
                                            ) : (
                                              <Square className="h-3.5 w-3.5" />
                                            )}
                                          </button>
                                        </th>
                                        <th className="text-left px-3 py-1.5 font-medium cursor-pointer select-none hover:text-foreground transition-colors" onClick={() => handleSort('status')}>
                                          <span className="inline-flex items-center">Status<SortIcon column="status" /></span>
                                        </th>
                                        <th className="text-left px-3 py-1.5 font-medium cursor-pointer select-none hover:text-foreground transition-colors" onClick={() => handleSort('name')}>
                                          <span className="inline-flex items-center">Name<SortIcon column="name" /></span>
                                        </th>
                                        <th className="text-left px-3 py-1.5 font-medium text-muted-foreground">
                                          Host
                                        </th>
                                        <th className="text-left px-3 py-1.5 font-medium cursor-pointer select-none hover:text-foreground transition-colors" onClick={() => handleSort('type')}>
                                          <span className="inline-flex items-center">Type<SortIcon column="type" /></span>
                                        </th>
                                        <th className="text-left px-3 py-1.5 font-medium text-muted-foreground">
                                          Alert Channels
                                        </th>
                                        {tenantRegionCount > 0 && (
                                          <th className="text-left px-3 py-1.5 font-medium text-muted-foreground">
                                            Regions
                                          </th>
                                        )}
                                        <th className="text-left px-3 py-1.5 font-medium cursor-pointer select-none hover:text-foreground transition-colors" onClick={() => handleSort('duration')}>
                                          <span className="inline-flex items-center">Freq<SortIcon column="duration" /></span>
                                        </th>
                                        <th className="text-left px-3 py-1.5 font-medium cursor-pointer select-none hover:text-foreground transition-colors" onClick={() => handleSort('enabled')}>
                                          <span className="inline-flex items-center">On<SortIcon column="enabled" /></span>
                                        </th>
                                        <th className="text-right px-3 py-1.5 font-medium">Actions</th>
                                      </tr>
                                    </thead>
                                    <tbody>
                                      {sg.checks.map(renderCheckRow)}
                                    </tbody>
                                  </table>
                                </div>
                              )}
                            </div>
                          )
                        })}
                      </div>
                    )}
                  </div>
                )
              })
            )}
          </div>

          {/* Mobile card view — shown only on small screens */}
          <div className="sm:hidden space-y-4">
            {loading ? (
              <div className="text-center py-8 text-muted-foreground">Loading...</div>
            ) : groups.length === 0 ? (
              <div className="text-center py-8 text-muted-foreground">No check definitions found</div>
            ) : (
              groups.map((group) => {
                const projectKey = `p:${group.project}`
                const isProjectCollapsed = collapsedGroups.has(projectKey)
                const showSubGroupHeaders = group.subGroups.length > 1 || (group.subGroups.length === 1 && group.subGroups[0].group !== '')

                return (
                  <div key={group.project}>
                    {/* Mobile project header */}
                    <button
                      onClick={() => toggleGroup(projectKey)}
                      className="w-full flex items-center gap-2 px-3 py-2 rounded-lg bg-muted/50 mb-2"
                    >
                      {isProjectCollapsed ? (
                        <ChevronRight className="h-4 w-4 text-muted-foreground" />
                      ) : (
                        <ChevronDown className="h-4 w-4 text-muted-foreground" />
                      )}
                      <FolderOpen className="h-4 w-4 text-muted-foreground" />
                      <span className="font-semibold text-sm">{group.project}</span>
                      <Badge variant="secondary" className="text-[10px] ml-auto">
                        {group.checks.length}
                      </Badge>
                    </button>

                    {!isProjectCollapsed && group.subGroups.map((sg) => {
                      const subGroupKey = `g:${group.project}/${sg.group}`
                      const isSubGroupCollapsed = collapsedGroups.has(subGroupKey)

                      return (
                        <div key={sg.group} className="mb-2">
                          {showSubGroupHeaders && (
                            <button
                              onClick={() => toggleGroup(subGroupKey)}
                              className="w-full flex items-center gap-2 px-3 py-1.5 pl-6 rounded-md bg-muted/25 mb-1"
                            >
                              {isSubGroupCollapsed ? (
                                <ChevronRight className="h-3.5 w-3.5 text-muted-foreground" />
                              ) : (
                                <ChevronDown className="h-3.5 w-3.5 text-muted-foreground" />
                              )}
                              <Layers className="h-3.5 w-3.5 text-muted-foreground" />
                              <span className="text-xs font-medium text-muted-foreground">{sg.group || '(no group)'}</span>
                              <Badge variant="secondary" className="text-[10px] ml-auto">
                                {sg.checks.length}
                              </Badge>
                            </button>
                          )}

                          {!(showSubGroupHeaders && isSubGroupCollapsed) && (
                            <div className={showSubGroupHeaders ? 'ml-3 border-l-2 border-border/40 pl-2 space-y-2' : 'space-y-2'}>
                              {sg.checks.map((def) => {
                                const isSelected = selectedUUIDs.has(def.uuid)
                                return (
                                  <div
                                    key={def.uuid}
                                    className={`rounded-lg border bg-card p-4 active:bg-muted/50 transition-colors cursor-pointer ${
                                      isSelected ? 'bg-primary/5 border-primary/30' : ''
                                    }`}
                                    onClick={() => handleEdit(def)}
                                  >
                                    <div className="flex items-start justify-between gap-2">
                                      <div className="min-w-0 flex-1">
                                        <div className="font-medium text-sm truncate">{def.name}</div>
                                        {(def.url || def.host || def.domain || def.mongodb_uri) && (
                                          <div className="font-mono text-[11px] text-muted-foreground mt-0.5 truncate">{def.url || def.host || def.domain || def.mongodb_uri}</div>
                                        )}
                                        <div className="mt-1.5 flex flex-wrap items-center gap-2">
                                          <AlertChannelBadges channels={def.alert_channels} defaultChannels={checkDefaults?.alert_channels} />
                                          {tenantRegionCount > 0 && <RegionBadges regions={def.target_regions} totalRegions={tenantRegionCount} />}
                                        </div>
                                      </div>
                                      <div className="flex items-center gap-2 shrink-0">
                                        <Badge variant="secondary" className="text-[10px]">
                                          {def.type}
                                        </Badge>
                                        {(() => {
                                          const liveCheck = checksMap.get(def.uuid)
                                          const status = getCheckStatus(def, liveCheck)
                                          return (
                                            <div className="flex items-center gap-1" title={status.label}>
                                              {status.pending ? (
                                                <span className="inline-block h-2.5 w-2.5 rounded-full bg-muted-foreground/40 shrink-0" />
                                              ) : (
                                                <StatusDot
                                                  healthy={status.healthy}
                                                  enabled={status.enabled}
                                                  silenced={status.silenced}
                                                  size="sm"
                                                />
                                              )}
                                            </div>
                                          )
                                        })()}
                                      </div>
                                    </div>
                                    <div className="flex items-center justify-between mt-3">
                                      <span className="text-xs text-muted-foreground font-mono">{def.duration}</span>
                                      <div className="flex items-center gap-1" onClick={(e) => e.stopPropagation()}>
                                        <Button variant="ghost" size="icon" className="h-9 w-9 min-h-[44px] min-w-[44px]" onClick={() => handleClone(def)}>
                                          <Copy className="h-4 w-4" />
                                        </Button>
                                        <Button variant="ghost" size="icon" className="h-9 w-9 min-h-[44px] min-w-[44px]" onClick={() => handleEdit(def)}>
                                          <Pencil className="h-4 w-4" />
                                        </Button>
                                        <Button
                                          variant="ghost"
                                          size="icon"
                                          className="h-9 w-9 min-h-[44px] min-w-[44px] text-unhealthy hover:text-unhealthy"
                                          onClick={() => {
                                            setDeletingUUID(def.uuid)
                                            setDeleteDialogOpen(true)
                                          }}
                                        >
                                          <Trash2 className="h-4 w-4" />
                                        </Button>
                                      </div>
                                    </div>
                                  </div>
                                )
                              })}
                            </div>
                          )}
                        </div>
                      )
                    })}
                  </div>
                )
              })
            )}
          </div>
        </main>

        <StatusBar wsStatus={wsStatus} />

        {/* Sticky bulk action bar */}
        {selectedInView.size > 0 && (
          <div className="fixed bottom-8 left-1/2 -translate-x-1/2 z-50">
            <div className="flex items-center gap-3 bg-card border border-border rounded-lg shadow-lg px-4 py-2.5">
              <span className="text-sm font-medium whitespace-nowrap">
                {selectedInView.size} {selectedInView.size === 1 ? 'check' : 'checks'} selected
              </span>
              <div className="h-5 w-px bg-border" />
              <div className="flex items-center gap-1.5">
                <Tooltip>
                  <TooltipTrigger asChild>
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={handleBulkEnable}
                      disabled={bulkActing}
                    >
                      <Power className="h-4 w-4 mr-1 text-healthy" />
                      Enable
                    </Button>
                  </TooltipTrigger>
                  <TooltipContent>Enable {selectedInView.size} selected checks</TooltipContent>
                </Tooltip>
                <Tooltip>
                  <TooltipTrigger asChild>
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={handleBulkDisable}
                      disabled={bulkActing}
                    >
                      <PowerOff className="h-4 w-4 mr-1 text-warning" />
                      Disable
                    </Button>
                  </TooltipTrigger>
                  <TooltipContent>Disable {selectedInView.size} selected checks</TooltipContent>
                </Tooltip>
                <Tooltip>
                  <TooltipTrigger asChild>
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={() => setBulkMaintenanceDialogOpen(true)}
                      disabled={bulkActing}
                    >
                      <Clock className="h-4 w-4 mr-1" />
                      Set Maintenance
                    </Button>
                  </TooltipTrigger>
                  <TooltipContent>Set maintenance window on {selectedInView.size} selected checks</TooltipContent>
                </Tooltip>
                <Tooltip>
                  <TooltipTrigger asChild>
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={() => {
                        setBulkAlertAction('add')
                        setBulkAlertSelectedChannels(new Set())
                        setBulkAlertChannelsDialogOpen(true)
                      }}
                      disabled={bulkActing}
                    >
                      <Bell className="h-4 w-4 mr-1" />
                      Set Alert Channels
                    </Button>
                  </TooltipTrigger>
                  <TooltipContent>Change alert channels on {selectedInView.size} selected checks</TooltipContent>
                </Tooltip>
                <Tooltip>
                  <TooltipTrigger asChild>
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={() => setBulkDeleteDialogOpen(true)}
                      disabled={bulkActing}
                      className="text-unhealthy hover:text-unhealthy"
                    >
                      <Trash2 className="h-4 w-4 mr-1" />
                      Delete
                    </Button>
                  </TooltipTrigger>
                  <TooltipContent>Delete {selectedInView.size} selected checks</TooltipContent>
                </Tooltip>
              </div>
              <div className="h-5 w-px bg-border" />
              <Tooltip>
                <TooltipTrigger asChild>
                  <Button
                    variant="ghost"
                    size="icon"
                    className="h-7 w-7"
                    onClick={() => setSelectedUUIDs(new Set())}
                  >
                    <X className="h-4 w-4" />
                  </Button>
                </TooltipTrigger>
                <TooltipContent>Clear selection</TooltipContent>
              </Tooltip>
            </div>
          </div>
        )}

        {/* Edit/Create Drawer */}
        <CheckEditDrawer
          open={editDialogOpen}
          onOpenChange={setEditDialogOpen}
          editingCheck={editingCheck}
          onCheckChange={setEditingCheck}
          onSave={handleSave}
          saving={saving}
          existingProjects={uniqueProjects}
          existingGroups={uniqueGroups}
        />

        {/* Delete Confirmation */}
        <Dialog open={deleteDialogOpen} onOpenChange={setDeleteDialogOpen}>
          <DialogContent>
            <DialogHeader>
              <DialogTitle>Delete Check</DialogTitle>
              <DialogDescription>
                Are you sure you want to delete this check? This action cannot be undone.
              </DialogDescription>
            </DialogHeader>
            <DialogFooter>
              <Button variant="outline" onClick={() => setDeleteDialogOpen(false)}>
                Cancel
              </Button>
              <Button variant="destructive" onClick={handleDelete}>
                Delete
              </Button>
            </DialogFooter>
          </DialogContent>
        </Dialog>

        {/* Bulk Delete Confirmation */}
        <Dialog open={bulkDeleteDialogOpen} onOpenChange={setBulkDeleteDialogOpen}>
          <DialogContent>
            <DialogHeader>
              <DialogTitle>Delete {selectedInView.size} Checks</DialogTitle>
              <DialogDescription>
                Are you sure you want to delete {selectedInView.size} selected checks? This action cannot be undone.
              </DialogDescription>
            </DialogHeader>
            <DialogFooter>
              <Button variant="outline" onClick={() => setBulkDeleteDialogOpen(false)}>
                Cancel
              </Button>
              <Button variant="destructive" onClick={handleBulkDelete} disabled={bulkActing}>
                {bulkActing ? 'Deleting...' : `Delete ${selectedInView.size} Checks`}
              </Button>
            </DialogFooter>
          </DialogContent>
        </Dialog>

        {/* Bulk Maintenance Dialog */}
        <Dialog open={bulkMaintenanceDialogOpen} onOpenChange={(open) => {
          setBulkMaintenanceDialogOpen(open)
          if (!open) setBulkMaintenanceUntil('')
        }}>
          <DialogContent>
            <DialogHeader>
              <DialogTitle>Set Maintenance Window</DialogTitle>
              <DialogDescription>
                Set a maintenance window for {selectedInView.size} selected checks.
                Checks in maintenance will not trigger alerts.
              </DialogDescription>
            </DialogHeader>
            <div className="py-4">
              <label className="text-sm font-medium mb-2 block">Maintenance until</label>
              <Input
                type="datetime-local"
                value={bulkMaintenanceUntil}
                onChange={(e) => setBulkMaintenanceUntil(e.target.value)}
                min={new Date().toISOString().slice(0, 16)}
              />
            </div>
            <DialogFooter>
              <Button variant="outline" onClick={() => setBulkMaintenanceDialogOpen(false)}>
                Cancel
              </Button>
              <Button onClick={handleBulkMaintenance} disabled={bulkActing || !bulkMaintenanceUntil}>
                {bulkActing ? 'Setting...' : `Set Maintenance on ${selectedInView.size} Checks`}
              </Button>
            </DialogFooter>
          </DialogContent>
        </Dialog>

        {/* Bulk Alert Channels Dialog */}
        <Dialog open={bulkAlertChannelsDialogOpen} onOpenChange={(open) => {
          setBulkAlertChannelsDialogOpen(open)
          if (!open) {
            setBulkAlertSelectedChannels(new Set())
            setBulkAlertAction('add')
          }
        }}>
          <DialogContent className="sm:max-w-md">
            <DialogHeader>
              <DialogTitle>Set Alert Channels</DialogTitle>
              <DialogDescription>
                Update alert channels for {selectedInView.size} selected {selectedInView.size === 1 ? 'check' : 'checks'}.
              </DialogDescription>
            </DialogHeader>
            <div className="py-4 space-y-4">
              <div>
                <label className="text-sm font-medium mb-2 block">Action</label>
                <div className="flex gap-2">
                  {(['add', 'remove', 'replace'] as const).map((action) => (
                    <Button
                      key={action}
                      variant={bulkAlertAction === action ? 'default' : 'outline'}
                      size="sm"
                      onClick={() => setBulkAlertAction(action)}
                    >
                      {action === 'add' ? 'Add to existing' : action === 'remove' ? 'Remove from existing' : 'Replace all'}
                    </Button>
                  ))}
                </div>
              </div>
              <div>
                <label className="text-sm font-medium mb-2 block">Channels</label>
                <div className="flex flex-wrap gap-2">
                  {allAlertChannels.length === 0 ? (
                    <p className="text-sm text-muted-foreground">No alert channels configured in the system.</p>
                  ) : (
                    allAlertChannels.map((channel) => {
                      const meta = CHANNEL_TYPES.find((ct) => ct.value === channel.type)
                      const isSelected = bulkAlertSelectedChannels.has(channel.name)
                      return (
                        <button
                          key={channel.name}
                          type="button"
                          onClick={() => {
                            setBulkAlertSelectedChannels((prev) => {
                              const next = new Set(prev)
                              if (next.has(channel.name)) next.delete(channel.name)
                              else next.add(channel.name)
                              return next
                            })
                          }}
                          className={cn(
                            'inline-flex items-center gap-1.5 px-3 py-1.5 rounded-md text-sm font-medium border transition-colors',
                            isSelected
                              ? 'border-primary bg-primary text-primary-foreground'
                              : 'border-border bg-card hover:bg-muted'
                          )}
                        >
                          {isSelected && <CheckSquare className="h-3.5 w-3.5" />}
                          {!isSelected && <Square className="h-3.5 w-3.5" />}
                          {channel.name}
                          {meta && <span className="text-xs opacity-70">({meta.label})</span>}
                        </button>
                      )
                    })
                  )}
                </div>
              </div>
            </div>
            <DialogFooter>
              <Button variant="outline" onClick={() => setBulkAlertChannelsDialogOpen(false)}>
                Cancel
              </Button>
              <Button
                onClick={handleBulkAlertChannels}
                disabled={bulkActing || bulkAlertSelectedChannels.size === 0}
              >
                {bulkActing ? 'Updating...' : `Update ${selectedInView.size} ${selectedInView.size === 1 ? 'Check' : 'Checks'}`}
              </Button>
            </DialogFooter>
          </DialogContent>
        </Dialog>

        {/* Import Dialog */}
        <ImportDialog
          open={importDialogOpen}
          onOpenChange={setImportDialogOpen}
          onImportComplete={fetchData}
        />
      </div>
    </TooltipProvider>
  )
}
