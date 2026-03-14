import { useState, useEffect, useCallback, useMemo } from 'react'
import { useSearchParams } from 'react-router-dom'
import { api, type CheckDefinition } from '@/lib/api'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Badge } from '@/components/ui/badge'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from '@/components/ui/dialog'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Switch } from '@/components/ui/switch'
import { Separator } from '@/components/ui/separator'
import { Plus, Pencil, Trash2, Search, RefreshCw, Upload, Download, ArrowUp, ArrowDown, ArrowUpDown, ChevronDown, X } from 'lucide-react'
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from '@/components/ui/collapsible'
import { TooltipProvider } from '@/components/ui/tooltip'
import { TopBar } from '@/components/TopBar'
import { StatusBar } from '@/components/StatusBar'
import { useChecks } from '@/hooks/useChecks'
import { useRef } from 'react'
import { ImportDialog } from '@/components/ImportDialog'
import { api as apiClient } from '@/lib/api'

/** Inline string list editor — add/remove entries (used for server_list, analytic_replicas) */
function StringListEditor({
  label,
  values,
  onChange,
  placeholder = 'Add entry…',
}: {
  label: string
  values: string[]
  onChange: (v: string[]) => void
  placeholder?: string
}) {
  const [draft, setDraft] = useState('')
  const add = () => {
    const trimmed = draft.trim()
    if (trimmed && !values.includes(trimmed)) {
      onChange([...values, trimmed])
      setDraft('')
    }
  }
  return (
    <div>
      <label className="text-xs text-muted-foreground">{label}</label>
      <div className="flex gap-2 mt-1">
        <Input
          value={draft}
          onChange={(e) => setDraft(e.target.value)}
          onKeyDown={(e) => {
            if (e.key === 'Enter') {
              e.preventDefault()
              add()
            }
          }}
          placeholder={placeholder}
          className="flex-1"
        />
        <Button type="button" variant="outline" size="sm" onClick={add} disabled={!draft.trim()}>
          <Plus className="h-3 w-3" />
        </Button>
      </div>
      {values.length > 0 && (
        <div className="flex flex-wrap gap-1.5 mt-2">
          {values.map((v, i) => (
            <Badge key={i} variant="secondary" className="gap-1 pr-1">
              <span className="font-mono text-[11px]">{v}</span>
              <button
                type="button"
                className="ml-0.5 hover:text-destructive"
                onClick={() => onChange(values.filter((_, idx) => idx !== i))}
              >
                <X className="h-3 w-3" />
              </button>
            </Badge>
          ))}
        </div>
      )}
    </div>
  )
}

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
  const [editingCheck, setEditingCheck] = useState<Partial<CheckDefinition>>(EMPTY_FORM)
  const [deletingUUID, setDeletingUUID] = useState<string | null>(null)
  const [saving, setSaving] = useState(false)

  const [importDialogOpen, setImportDialogOpen] = useState(false)

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
    } catch (err) {
      console.error('Failed to fetch data:', err)
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    fetchData()
  }, [fetchData])

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
          // Third click resets sorting — clear params
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
    setAdvancedOpen(false)
    setEditDialogOpen(true)
  }

  const handleEdit = (def: CheckDefinition) => {
    setEditingCheck({ ...def })
    // Auto-expand advanced if editing a DB check with advanced fields populated
    const db = def.mysql || def.pgsql
    setAdvancedOpen(!!(db && (db.username || db.password || db.dbname || db.query)))
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
      // Create a blob and download
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

  const [advancedOpen, setAdvancedOpen] = useState(false)

  const updateForm = (field: string, value: string | number | boolean) => {
    setEditingCheck((prev) => ({ ...prev, [field]: value }))
  }

  const isMySQL = editingCheck.type?.includes('mysql')
  const isPgSQL = editingCheck.type?.includes('pgsql')
  const isDB = isMySQL || isPgSQL

  /** Helper: get the nested db config key name */
  const dbKey = isMySQL ? 'mysql' : 'pgsql'
  const dbConfig = isMySQL ? editingCheck.mysql : editingCheck.pgsql

  const updateDBField = (field: string, value: string | string[]) => {
    setEditingCheck((prev) => ({
      ...prev,
      [dbKey]: { ...(isMySQL ? prev.mysql : prev.pgsql), [field]: value },
    }))
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
          onOpenCommandPalette={() => {}}
        />

        <main className="mx-auto max-w-[1600px] px-4 py-4 space-y-4">
          {/* Actions bar */}
          <div className="flex items-center justify-between">
            <h2 className="text-lg font-semibold">Check Definitions</h2>
            <div className="flex items-center gap-2">
              <Button variant="outline" size="sm" onClick={fetchData} disabled={loading}>
                <RefreshCw className={`h-4 w-4 mr-1 ${loading ? 'animate-spin' : ''}`} />
                Refresh
              </Button>
              <Button variant="outline" size="sm" onClick={handleExport}>
                <Download className="h-4 w-4 mr-1" />
                Export
              </Button>
              <Button variant="outline" size="sm" onClick={() => setImportDialogOpen(true)}>
                <Upload className="h-4 w-4 mr-1" />
                Import YAML
              </Button>
              <Button size="sm" onClick={handleCreate}>
                <Plus className="h-4 w-4 mr-1" />
                New Check
              </Button>
            </div>
          </div>

          {/* Table */}
          <div className="rounded-lg border bg-card overflow-hidden">
            <div className="overflow-x-auto">
              <table className="w-full text-sm">
                <thead>
                  <tr className="border-b bg-[hsl(215_14%_10%)] text-muted-foreground text-xs">
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
                      <td colSpan={6} className="text-center py-8 text-muted-foreground">
                        Loading...
                      </td>
                    </tr>
                  ) : sorted.length === 0 ? (
                    <tr>
                      <td colSpan={6} className="text-center py-8 text-muted-foreground">
                        No check definitions found
                      </td>
                    </tr>
                  ) : (
                    sorted.map((def) => (
                      <tr
                        key={def.uuid}
                        className="border-b border-border/50 hover:bg-[hsl(215_14%_14%)] transition-colors"
                      >
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
                            <Button variant="ghost" size="icon" className="h-7 w-7" onClick={() => handleEdit(def)}>
                              <Pencil className="h-3.5 w-3.5" />
                            </Button>
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
                          </div>
                        </td>
                      </tr>
                    ))
                  )}
                </tbody>
              </table>
            </div>
          </div>
        </main>

        <StatusBar wsStatus={wsStatus} />

        {/* Edit/Create Dialog */}
        <Dialog open={editDialogOpen} onOpenChange={setEditDialogOpen}>
          <DialogContent className="max-w-xl max-h-[85vh] overflow-y-auto">
            <DialogHeader>
              <DialogTitle>{editingCheck.uuid ? 'Edit Check' : 'Create New Check'}</DialogTitle>
              <DialogDescription>
                {editingCheck.uuid ? `Editing ${editingCheck.name}` : 'Configure a new check definition'}
              </DialogDescription>
            </DialogHeader>

            <div className="space-y-4">
              {/* Basic fields */}
              <div className="grid grid-cols-2 gap-3">
                <div>
                  <label className="text-xs text-muted-foreground">Name</label>
                  <Input value={editingCheck.name || ''} onChange={(e) => updateForm('name', e.target.value)} />
                </div>
                <div>
                  <label className="text-xs text-muted-foreground">Project</label>
                  <Input value={editingCheck.project || ''} onChange={(e) => updateForm('project', e.target.value)} />
                </div>
              </div>

              <div className="grid grid-cols-2 gap-3">
                <div>
                  <label className="text-xs text-muted-foreground">Group</label>
                  <Input
                    value={editingCheck.group_name || ''}
                    onChange={(e) => updateForm('group_name', e.target.value)}
                  />
                </div>
                <div>
                  <label className="text-xs text-muted-foreground">Type</label>
                  <Select
                    value={editingCheck.type || 'http'}
                    onValueChange={(v) => updateForm('type', v)}
                  >
                    <SelectTrigger>
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="http">HTTP</SelectItem>
                      <SelectItem value="tcp">TCP</SelectItem>
                      <SelectItem value="icmp">ICMP</SelectItem>
                      <SelectItem value="passive">Passive</SelectItem>
                      <SelectItem value="mysql_query">MySQL Query</SelectItem>
                      <SelectItem value="pgsql_query">PostgreSQL Query</SelectItem>
                      <SelectItem value="pgsql_replication">PostgreSQL Replication</SelectItem>
                    </SelectContent>
                  </Select>
                </div>
              </div>

              <div className="grid grid-cols-2 gap-3">
                <div>
                  <label className="text-xs text-muted-foreground">Duration (frequency)</label>
                  <Input value={editingCheck.duration || ''} onChange={(e) => updateForm('duration', e.target.value)} />
                </div>
                <div>
                  <label className="text-xs text-muted-foreground">Timeout</label>
                  <Input value={editingCheck.timeout || ''} onChange={(e) => updateForm('timeout', e.target.value)} />
                </div>
              </div>

              <div>
                <label className="text-xs text-muted-foreground">Description</label>
                <Input
                  value={editingCheck.description || ''}
                  onChange={(e) => updateForm('description', e.target.value)}
                />
              </div>

              <div className="flex items-center gap-2">
                <Switch
                  checked={editingCheck.enabled ?? true}
                  onCheckedChange={(v) => updateForm('enabled', v)}
                />
                <label className="text-sm">Enabled</label>
              </div>

              <Separator />

              {/* Type-specific fields */}
              {(editingCheck.type === 'http') && (
                <div>
                  <h4 className="text-sm font-medium mb-2">HTTP Configuration</h4>
                  <div>
                    <label className="text-xs text-muted-foreground">URL</label>
                    <Input value={editingCheck.url || ''} onChange={(e) => updateForm('url', e.target.value)} />
                  </div>
                </div>
              )}

              {(editingCheck.type === 'tcp') && (
                <div>
                  <h4 className="text-sm font-medium mb-2">TCP Configuration</h4>
                  <div className="grid grid-cols-2 gap-3">
                    <div>
                      <label className="text-xs text-muted-foreground">Host</label>
                      <Input value={editingCheck.host || ''} onChange={(e) => updateForm('host', e.target.value)} />
                    </div>
                    <div>
                      <label className="text-xs text-muted-foreground">Port</label>
                      <Input
                        type="number"
                        value={editingCheck.port || ''}
                        onChange={(e) => updateForm('port', parseInt(e.target.value) || 0)}
                      />
                    </div>
                  </div>
                </div>
              )}

              {(editingCheck.type === 'icmp') && (
                <div>
                  <h4 className="text-sm font-medium mb-2">ICMP Configuration</h4>
                  <div>
                    <label className="text-xs text-muted-foreground">Host</label>
                    <Input value={editingCheck.host || ''} onChange={(e) => updateForm('host', e.target.value)} />
                  </div>
                </div>
              )}

              {isDB && (
                <div className="space-y-3">
                  <h4 className="text-sm font-medium mb-2">Database Configuration</h4>
                  <div className="grid grid-cols-2 gap-3">
                    <div>
                      <label className="text-xs text-muted-foreground">Host</label>
                      <Input value={editingCheck.host || ''} onChange={(e) => updateForm('host', e.target.value)} />
                    </div>
                    <div>
                      <label className="text-xs text-muted-foreground">Port</label>
                      <Input
                        type="number"
                        value={editingCheck.port || ''}
                        onChange={(e) => updateForm('port', parseInt(e.target.value) || 0)}
                      />
                    </div>
                  </div>

                  {/* Advanced Settings */}
                  <Collapsible open={advancedOpen} onOpenChange={setAdvancedOpen}>
                    <CollapsibleTrigger asChild>
                      <Button variant="ghost" size="sm" className="w-full justify-between px-2 text-xs text-muted-foreground hover:text-foreground">
                        Advanced Settings
                        <ChevronDown className={`h-3.5 w-3.5 transition-transform ${advancedOpen ? 'rotate-180' : ''}`} />
                      </Button>
                    </CollapsibleTrigger>
                    <CollapsibleContent className="space-y-3 pt-2">
                      {/* Connection */}
                      <div className="grid grid-cols-2 gap-3">
                        <div>
                          <label className="text-xs text-muted-foreground">Username</label>
                          <Input
                            value={dbConfig?.username || ''}
                            onChange={(e) => updateDBField('username', e.target.value)}
                          />
                        </div>
                        <div>
                          <label className="text-xs text-muted-foreground">Password</label>
                          <Input
                            type="password"
                            value={dbConfig?.password || ''}
                            onChange={(e) => updateDBField('password', e.target.value)}
                          />
                        </div>
                      </div>
                      <div className="grid grid-cols-2 gap-3">
                        <div>
                          <label className="text-xs text-muted-foreground">Database Name</label>
                          <Input
                            value={dbConfig?.dbname || ''}
                            onChange={(e) => updateDBField('dbname', e.target.value)}
                          />
                        </div>
                        {isPgSQL && (
                          <div>
                            <label className="text-xs text-muted-foreground">SSL Mode</label>
                            <Select
                              value={(editingCheck.pgsql?.sslmode) || 'disable'}
                              onValueChange={(v) => updateDBField('sslmode', v)}
                            >
                              <SelectTrigger>
                                <SelectValue />
                              </SelectTrigger>
                              <SelectContent>
                                <SelectItem value="disable">disable</SelectItem>
                                <SelectItem value="allow">allow</SelectItem>
                                <SelectItem value="prefer">prefer</SelectItem>
                                <SelectItem value="require">require</SelectItem>
                                <SelectItem value="verify-ca">verify-ca</SelectItem>
                                <SelectItem value="verify-full">verify-full</SelectItem>
                              </SelectContent>
                            </Select>
                          </div>
                        )}
                      </div>

                      {/* Query */}
                      <Separator />
                      <div>
                        <label className="text-xs text-muted-foreground">Query</label>
                        <textarea
                          className="flex w-full rounded-md border border-input bg-background px-3 py-2 text-sm font-mono ring-offset-background placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 min-h-[80px] resize-y"
                          value={dbConfig?.query || ''}
                          onChange={(e) => updateDBField('query', e.target.value)}
                          placeholder="SELECT 1"
                        />
                      </div>
                      <div className="grid grid-cols-2 gap-3">
                        <div>
                          <label className="text-xs text-muted-foreground">Expected Response</label>
                          <Input
                            value={dbConfig?.response || ''}
                            onChange={(e) => updateDBField('response', e.target.value)}
                          />
                        </div>
                        <div>
                          <label className="text-xs text-muted-foreground">Difference</label>
                          <Input
                            value={dbConfig?.difference || ''}
                            onChange={(e) => updateDBField('difference', e.target.value)}
                            placeholder="Acceptable difference threshold"
                          />
                        </div>
                      </div>

                      {/* Replication */}
                      <Separator />
                      <div className="grid grid-cols-2 gap-3">
                        <div>
                          <label className="text-xs text-muted-foreground">Table Name</label>
                          <Input
                            value={dbConfig?.table_name || ''}
                            onChange={(e) => updateDBField('table_name', e.target.value)}
                          />
                        </div>
                        <div>
                          <label className="text-xs text-muted-foreground">Lag</label>
                          <Input
                            value={dbConfig?.lag || ''}
                            onChange={(e) => updateDBField('lag', e.target.value)}
                            placeholder="Acceptable replication lag"
                          />
                        </div>
                      </div>

                      <StringListEditor
                        label="Server List"
                        values={dbConfig?.server_list || []}
                        onChange={(v) => updateDBField('server_list', v)}
                        placeholder="Add server (e.g. host:port)"
                      />

                      {isPgSQL && (
                        <StringListEditor
                          label="Analytic Replicas"
                          values={editingCheck.pgsql?.analytic_replicas || []}
                          onChange={(v) => updateDBField('analytic_replicas', v)}
                          placeholder="Add replica (e.g. host:port)"
                        />
                      )}
                    </CollapsibleContent>
                  </Collapsible>
                </div>
              )}

              {/* Alert config */}
              <Separator />
              <div>
                <h4 className="text-sm font-medium mb-2">Alert Configuration</h4>
                <div className="grid grid-cols-2 gap-3">
                  <div>
                    <label className="text-xs text-muted-foreground">Alert Type</label>
                    <Select
                      value={editingCheck.alert_type || 'none'}
                      onValueChange={(v) => updateForm('alert_type', v === 'none' ? '' : v)}
                    >
                      <SelectTrigger>
                        <SelectValue />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectItem value="none">None</SelectItem>
                        <SelectItem value="slack">Slack</SelectItem>
                        <SelectItem value="webhook">Webhook</SelectItem>
                      </SelectContent>
                    </Select>
                  </div>
                  <div>
                    <label className="text-xs text-muted-foreground">Alert Destination</label>
                    <Input
                      value={editingCheck.alert_destination || ''}
                      onChange={(e) => updateForm('alert_destination', e.target.value)}
                    />
                  </div>
                </div>
              </div>
            </div>

            <DialogFooter>
              <Button variant="outline" onClick={() => setEditDialogOpen(false)}>
                Cancel
              </Button>
              <Button onClick={handleSave} disabled={saving}>
                {saving ? 'Saving...' : editingCheck.uuid ? 'Update' : 'Create'}
              </Button>
            </DialogFooter>
          </DialogContent>
        </Dialog>

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
