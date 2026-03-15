import { useState, useEffect, useCallback, useMemo } from 'react'
import { useSearchParams } from 'react-router-dom'
import { api, type CheckDefinition } from '@/lib/api'
import { Button } from '@/components/ui/button'
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
} from 'lucide-react'
import { TooltipProvider } from '@/components/ui/tooltip'
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip'
import { TopBar } from '@/components/TopBar'
import { StatusBar } from '@/components/StatusBar'
import { useChecks } from '@/hooks/useChecks'
import { useRef } from 'react'
import { ImportDialog } from '@/components/ImportDialog'
import { CheckEditDrawer } from '@/components/CheckEditDrawer'
import { Input } from '@/components/ui/input'
import { api as apiClient } from '@/lib/api'
import { toast } from 'sonner'

type SortColumn = 'name' | 'project' | 'type' | 'duration' | 'enabled'
type SortDirection = 'asc' | 'desc'

const VALID_SORT_COLUMNS: readonly string[] = ['name', 'project', 'type', 'duration', 'enabled'] as const
const VALID_SORT_DIRECTIONS: readonly string[] = ['asc', 'desc'] as const

function parseSortColumn(value: string | null): SortColumn | null {
  if (value && VALID_SORT_COLUMNS.includes(value)) return value as SortColumn
  return null
}

function parseSortDirection(value: string | null): SortDirection {
  if (value && VALID_SORT_DIRECTIONS.includes(value)) return value as SortDirection
  return 'asc'
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

export function Management() {
  const { wsStatus } = useChecks()
  const [definitions, setDefinitions] = useState<CheckDefinition[]>([])
  const [loading, setLoading] = useState(true)
  const [search, setSearch] = useState('')
  const [typeFilter, setTypeFilter] = useState('all')
  const [projectFilter, setProjectFilter] = useState('all')
  const [statusFilter, setStatusFilter] = useState('all')

  // Sort state — persisted in URL search params
  const [searchParams, setSearchParams] = useSearchParams()
  const sortColumn = parseSortColumn(searchParams.get('sort'))
  const sortDirection = parseSortDirection(searchParams.get('dir'))

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
      const [defs, projs, types] = await Promise.all([
        api.getChecks(),
        api.getProjects().catch(() => [] as string[]),
        api.getCheckTypes().catch(() => [] as string[]),
      ])
      setDefinitions(defs)
      setProjects(projs)
      setCheckTypes(types)
      // Clear selection of items that no longer exist
      setSelectedUUIDs((prev) => {
        const validUUIDs = new Set(defs.map((d) => d.uuid))
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
      // Remove the action param so it doesn't re-trigger
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
        !d.project.toLowerCase().includes(q)
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
    setSearchParams((prev) => {
      const next = new URLSearchParams(prev)
      if (sortColumn === column) {
        if (sortDirection === 'asc') {
          next.set('sort', column)
          next.set('dir', 'desc')
        } else {
          next.delete('sort')
          next.delete('dir')
        }
      } else {
        next.set('sort', column)
        next.set('dir', 'asc')
      }
      return next
    }, { replace: true })
  }

  const sorted = useMemo(() => {
    if (!sortColumn) return filtered
    return [...filtered].sort((a, b) => {
      let aVal: string | boolean
      let bVal: string | boolean
      switch (sortColumn) {
        case 'name':
          aVal = a.name.toLowerCase()
          bVal = b.name.toLowerCase()
          break
        case 'project':
          aVal = a.project.toLowerCase()
          bVal = b.project.toLowerCase()
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
  }, [filtered, sortColumn, sortDirection])

  const SortIcon = ({ column }: { column: SortColumn }) => {
    if (sortColumn !== column) return <ArrowUpDown className="h-3 w-3 ml-1 opacity-40" />
    if (sortDirection === 'asc') return <ArrowUp className="h-3 w-3 ml-1" />
    return <ArrowDown className="h-3 w-3 ml-1" />
  }

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
  const filteredUUIDs = useMemo(() => new Set(sorted.map((d) => d.uuid)), [sorted])
  const selectedInView = useMemo(
    () => new Set([...selectedUUIDs].filter((uuid) => filteredUUIDs.has(uuid))),
    [selectedUUIDs, filteredUUIDs]
  )

  const allSelected = sorted.length > 0 && selectedInView.size === sorted.length
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
          {/* Actions bar */}
          <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-2">
            <div className="flex items-center gap-2">
              <h2 className="text-lg font-semibold">Check Definitions</h2>
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
          <div className="rounded-lg border bg-card overflow-hidden hidden sm:block">
            <div className="overflow-x-auto">
              <table className="w-full text-sm">
                <thead>
                  <tr className="border-b bg-muted/50 text-muted-foreground text-xs">
                    {/* Checkbox header */}
                    <th className="px-3 py-2 w-10">
                      <button
                        onClick={toggleSelectAll}
                        className="flex items-center justify-center text-muted-foreground hover:text-foreground transition-colors"
                      >
                        {allSelected ? (
                          <CheckSquare className="h-4 w-4" />
                        ) : someSelected ? (
                          <MinusSquare className="h-4 w-4" />
                        ) : (
                          <Square className="h-4 w-4" />
                        )}
                      </button>
                    </th>
                    <th className="text-left px-3 py-2 font-medium cursor-pointer select-none hover:text-foreground transition-colors" onClick={() => handleSort('name')}>
                      <span className="inline-flex items-center">Name<SortIcon column="name" /></span>
                    </th>
                    <th className="text-left px-3 py-2 font-medium cursor-pointer select-none hover:text-foreground transition-colors" onClick={() => handleSort('project')}>
                      <span className="inline-flex items-center">Project<SortIcon column="project" /></span>
                    </th>
                    <th className="text-left px-3 py-2 font-medium cursor-pointer select-none hover:text-foreground transition-colors" onClick={() => handleSort('type')}>
                      <span className="inline-flex items-center">Type<SortIcon column="type" /></span>
                    </th>
                    <th className="text-left px-3 py-2 font-medium cursor-pointer select-none hover:text-foreground transition-colors" onClick={() => handleSort('duration')}>
                      <span className="inline-flex items-center">Frequency<SortIcon column="duration" /></span>
                    </th>
                    <th className="text-left px-3 py-2 font-medium cursor-pointer select-none hover:text-foreground transition-colors" onClick={() => handleSort('enabled')}>
                      <span className="inline-flex items-center">Enabled<SortIcon column="enabled" /></span>
                    </th>
                    <th className="text-right px-3 py-2 font-medium">Actions</th>
                  </tr>
                </thead>
                <tbody>
                  {loading ? (
                    <tr>
                      <td colSpan={7} className="text-center py-8 text-muted-foreground">
                        Loading...
                      </td>
                    </tr>
                  ) : sorted.length === 0 ? (
                    <tr>
                      <td colSpan={7} className="text-center py-8 text-muted-foreground">
                        No check definitions found
                      </td>
                    </tr>
                  ) : (
                    sorted.map((def) => {
                      const isSelected = selectedUUIDs.has(def.uuid)
                      return (
                        <tr
                          key={def.uuid}
                          className={`border-b border-border/50 transition-colors ${
                            isSelected
                              ? 'bg-primary/5 hover:bg-primary/10'
                              : 'hover:bg-muted/30'
                          }`}
                        >
                          {/* Checkbox */}
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
                            <div className="font-medium">{def.name}</div>
                            <div className="font-mono text-[10px] text-muted-foreground">{def.uuid}</div>
                          </td>
                          <td className="px-3 py-2 text-muted-foreground">{def.project}</td>
                          <td className="px-3 py-2">
                            <Badge variant="secondary" className="text-[10px]">
                              {def.type}
                            </Badge>
                          </td>
                          <td className="px-3 py-2 font-mono text-muted-foreground">{def.duration}</td>
                          <td className="px-3 py-2">
                            <Switch
                              checked={def.enabled}
                              onCheckedChange={() => handleToggle(def.uuid)}
                              className="scale-75"
                            />
                          </td>
                          <td className="px-3 py-2 text-right">
                            <div className="flex items-center justify-end gap-1">
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
                    })
                  )}
                </tbody>
              </table>
            </div>
          </div>

          {/* Mobile card view — shown only on small screens */}
          <div className="sm:hidden space-y-2">
            {loading ? (
              <div className="text-center py-8 text-muted-foreground">Loading...</div>
            ) : sorted.length === 0 ? (
              <div className="text-center py-8 text-muted-foreground">No check definitions found</div>
            ) : (
              sorted.map((def) => {
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
                        <div className="text-xs text-muted-foreground mt-0.5">{def.project}</div>
                      </div>
                      <div className="flex items-center gap-2 shrink-0">
                        <Badge variant="secondary" className="text-[10px]">
                          {def.type}
                        </Badge>
                        <div
                          className={`h-2.5 w-2.5 rounded-full ${def.enabled ? 'bg-healthy' : 'bg-disabled'}`}
                          title={def.enabled ? 'Enabled' : 'Disabled'}
                        />
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
