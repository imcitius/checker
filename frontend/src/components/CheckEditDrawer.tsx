import { useState, useEffect } from 'react'
import { type CheckDefinition } from '@/lib/api'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Badge } from '@/components/ui/badge'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
} from '@/components/ui/dialog'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Switch } from '@/components/ui/switch'
import {
  Collapsible,
  CollapsibleTrigger,
  CollapsibleContent,
} from '@/components/ui/collapsible'
import { Plus, X, ChevronDown, ChevronRight, Wrench } from 'lucide-react'
import { api } from '@/lib/api'

/** Inline string list editor — add/remove entries (used for server_list, analytic_replicas) */
function StringListEditor({
  label,
  values,
  onChange,
  placeholder = 'Add entry...',
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

/** Key-value pair editor for headers/cookies */
function KeyValueEditor({
  label,
  values,
  onChange,
  keyPlaceholder = 'Key',
  valuePlaceholder = 'Value',
}: {
  label: string
  values: Record<string, string>[]
  onChange: (v: Record<string, string>[]) => void
  keyPlaceholder?: string
  valuePlaceholder?: string
}) {
  const [draftKey, setDraftKey] = useState('')
  const [draftValue, setDraftValue] = useState('')
  const add = () => {
    const k = draftKey.trim()
    const v = draftValue.trim()
    if (k) {
      onChange([...values, { [k]: v }])
      setDraftKey('')
      setDraftValue('')
    }
  }
  return (
    <div>
      <label className="text-xs text-muted-foreground">{label}</label>
      <div className="flex gap-2 mt-1">
        <Input
          value={draftKey}
          onChange={(e) => setDraftKey(e.target.value)}
          placeholder={keyPlaceholder}
          className="flex-1"
        />
        <Input
          value={draftValue}
          onChange={(e) => setDraftValue(e.target.value)}
          placeholder={valuePlaceholder}
          className="flex-1"
          onKeyDown={(e) => {
            if (e.key === 'Enter') {
              e.preventDefault()
              add()
            }
          }}
        />
        <Button type="button" variant="outline" size="sm" onClick={add} disabled={!draftKey.trim()}>
          <Plus className="h-3 w-3" />
        </Button>
      </div>
      {values.length > 0 && (
        <div className="flex flex-wrap gap-1.5 mt-2">
          {values.map((entry, i) => {
            const [k, v] = Object.entries(entry)[0] || ['', '']
            return (
              <Badge key={i} variant="secondary" className="gap-1 pr-1">
                <span className="font-mono text-[11px]">
                  {k}: {v}
                </span>
                <button
                  type="button"
                  className="ml-0.5 hover:text-destructive"
                  onClick={() => onChange(values.filter((_, idx) => idx !== i))}
                >
                  <X className="h-3 w-3" />
                </button>
              </Badge>
            )
          })}
        </div>
      )}
    </div>
  )
}

/** Section header with a bottom border */
function SectionHeader({ children }: { children: React.ReactNode }) {
  return <h3 className="text-sm font-semibold border-b border-border pb-2 pt-1">{children}</h3>
}

/** Maintenance window section — quick-set buttons + custom datetime + clear */
function MaintenanceSection({
  uuid,
  maintenanceUntil,
  onMaintenanceChange,
}: {
  uuid: string
  maintenanceUntil?: string | null
  onMaintenanceChange: (v: string | null) => void
}) {
  const [loading, setLoading] = useState(false)
  const [customDatetime, setCustomDatetime] = useState('')

  const isActive = maintenanceUntil && new Date(maintenanceUntil) > new Date()

  const setWindow = async (minutes: number) => {
    setLoading(true)
    try {
      const until = new Date(Date.now() + minutes * 60 * 1000).toISOString()
      await api.setMaintenance(uuid, until)
      onMaintenanceChange(until)
    } catch (err) {
      console.error('Failed to set maintenance window:', err)
    } finally {
      setLoading(false)
    }
  }

  const setCustomWindow = async () => {
    if (!customDatetime) return
    setLoading(true)
    try {
      const until = new Date(customDatetime).toISOString()
      await api.setMaintenance(uuid, until)
      onMaintenanceChange(until)
    } catch (err) {
      console.error('Failed to set maintenance window:', err)
    } finally {
      setLoading(false)
    }
  }

  const clearWindow = async () => {
    setLoading(true)
    try {
      await api.clearMaintenance(uuid)
      onMaintenanceChange(null)
    } catch (err) {
      console.error('Failed to clear maintenance window:', err)
    } finally {
      setLoading(false)
    }
  }

  return (
    <section className="space-y-4">
      <SectionHeader>
        <span className="flex items-center gap-2">
          <Wrench className="h-4 w-4" />
          Maintenance Window
        </span>
      </SectionHeader>

      {isActive && (
        <div className="flex items-center gap-3 p-3 rounded-md bg-amber-500/10 border border-amber-500/30">
          <Badge variant="outline" className="text-amber-600 border-amber-500/50 font-medium">
            In Maintenance
          </Badge>
          <span className="text-sm text-muted-foreground">
            Until{' '}
            <span className="font-medium text-foreground">
              {new Date(maintenanceUntil!).toLocaleString()}
            </span>
          </span>
          <Button
            variant="ghost"
            size="sm"
            className="ml-auto text-destructive hover:text-destructive"
            onClick={clearWindow}
            disabled={loading}
          >
            <X className="h-3 w-3 mr-1" />
            Clear
          </Button>
        </div>
      )}

      <div>
        <label className="text-xs text-muted-foreground">Quick set</label>
        <div className="flex gap-2 mt-1">
          {[
            { label: '15m', minutes: 15 },
            { label: '1h', minutes: 60 },
            { label: '4h', minutes: 240 },
            { label: '24h', minutes: 1440 },
          ].map(({ label, minutes }) => (
            <Button
              key={label}
              variant="outline"
              size="sm"
              onClick={() => setWindow(minutes)}
              disabled={loading}
            >
              {label}
            </Button>
          ))}
        </div>
      </div>

      <div>
        <label className="text-xs text-muted-foreground">Custom datetime</label>
        <div className="flex gap-2 mt-1">
          <Input
            type="datetime-local"
            value={customDatetime}
            onChange={(e) => setCustomDatetime(e.target.value)}
            className="flex-1"
            min={new Date().toISOString().slice(0, 16)}
          />
          <Button
            variant="outline"
            size="sm"
            onClick={setCustomWindow}
            disabled={loading || !customDatetime}
          >
            Set
          </Button>
        </div>
      </div>
    </section>
  )
}

interface CheckEditDrawerProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  editingCheck: Partial<CheckDefinition>
  onCheckChange: (check: Partial<CheckDefinition>) => void
  onSave: () => void
  saving: boolean
  existingProjects?: string[]
  existingGroups?: string[]
}

export function CheckEditDrawer({
  open,
  onOpenChange,
  editingCheck,
  onCheckChange,
  onSave,
  saving,
  existingProjects = [],
  existingGroups = [],
}: CheckEditDrawerProps) {
  const isMySQL = editingCheck.type?.includes('mysql')
  const isPgSQL = editingCheck.type?.includes('pgsql')
  const isDB = isMySQL || isPgSQL
  const isHTTP = editingCheck.type === 'http'
  const isTCP = editingCheck.type === 'tcp'
  const isICMP = editingCheck.type === 'icmp'
  const isDNS = editingCheck.type === 'dns'
  const isSSH = editingCheck.type === 'ssh'
  const isRedis = editingCheck.type === 'redis'
  const isMongoDB = editingCheck.type === 'mongodb'
  const isDomainExpiry = editingCheck.type === 'domain_expiry'

  const dbKey = isMySQL ? 'mysql' : 'pgsql'
  const dbConfig = isMySQL ? editingCheck.mysql : editingCheck.pgsql

  // Determine if advanced section should default to open
  const hasExistingAdvancedHTTP =
    isHTTP &&
    !!(
      editingCheck.answer ||
      editingCheck.answer_present ||
      (editingCheck.code && editingCheck.code.length > 0) ||
      (editingCheck.headers && editingCheck.headers.length > 0) ||
      (editingCheck.cookies && editingCheck.cookies.length > 0) ||
      editingCheck.skip_check_ssl ||
      editingCheck.ssl_expiration_period ||
      editingCheck.stop_follow_redirects ||
      editingCheck.auth?.user ||
      editingCheck.auth?.password
    )

  const hasExistingAdvancedDB =
    isDB &&
    !!(
      dbConfig?.query ||
      dbConfig?.response ||
      dbConfig?.difference ||
      dbConfig?.table_name ||
      dbConfig?.lag ||
      (dbConfig?.server_list && dbConfig.server_list.length > 0)
    )

  const hasExistingAdvancedICMP = isICMP && !!(editingCheck.count && editingCheck.count > 0)
  const hasExistingAdvancedSSH = isSSH && !!editingCheck.expect_banner
  const hasExistingAdvancedRedis = isRedis && !!(editingCheck.redis_password || editingCheck.redis_db)

  const [advancedOpen, setAdvancedOpen] = useState(false)
  const [errors, setErrors] = useState<Set<string>>(new Set())

  // Sync advanced section open state when check changes
  useEffect(() => {
    setAdvancedOpen(hasExistingAdvancedHTTP || hasExistingAdvancedDB || hasExistingAdvancedICMP || hasExistingAdvancedSSH || hasExistingAdvancedRedis)
  }, [editingCheck.uuid, editingCheck.type]) // eslint-disable-line react-hooks/exhaustive-deps

  const updateForm = (field: string, value: string | number | boolean | number[] | Record<string, string>[]) => {
    onCheckChange({ ...editingCheck, [field]: value })
  }

  const updateDBField = (field: string, value: string | string[]) => {
    onCheckChange({
      ...editingCheck,
      [dbKey]: { ...(isMySQL ? editingCheck.mysql : editingCheck.pgsql), [field]: value },
    })
  }

  const updateAuth = (field: string, value: string) => {
    onCheckChange({
      ...editingCheck,
      auth: { ...editingCheck.auth, [field]: value },
    })
  }

  const validate = (): boolean => {
    const errs = new Set<string>()
    if (!editingCheck.name?.trim()) errs.add('name')
    if (isHTTP && !editingCheck.url?.trim()) errs.add('url')
    if ((isTCP || isDB || isSSH || isRedis) && !editingCheck.host?.trim()) errs.add('host')
    if (isDNS && !editingCheck.domain?.trim()) errs.add('domain')
    if (isDomainExpiry && !editingCheck.domain?.trim()) errs.add('domain')
    if (isMongoDB && !editingCheck.mongodb_uri?.trim()) errs.add('mongodb_uri')
    setErrors(errs)
    return errs.size === 0
  }

  const handleSave = () => {
    if (validate()) {
      onSave()
    }
  }

  const hasConnection = isHTTP || isTCP || isICMP || isDB || isDNS || isSSH || isRedis || isMongoDB || isDomainExpiry
  const hasAdvanced = isHTTP || isDB || isICMP || isDNS || isSSH || isRedis

  // Parse expected status codes from comma-separated string
  const codeString = (editingCheck.code || []).join(', ')

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent
        className="w-full h-full sm:w-auto sm:h-auto max-w-[100vw] sm:max-w-[80vw] min-w-0 sm:min-w-[600px] max-h-[100vh] sm:max-h-[85vh] p-0 flex flex-col gap-0 rounded-none sm:rounded-lg"
        style={{ maxWidth: 'min(100vw, 1200px)' }}
      >
        {/* Header */}
        <DialogHeader className="flex-shrink-0 px-6 pt-6 pb-4 border-b">
          <DialogTitle>{editingCheck.uuid ? 'Edit Check' : 'Create New Check'}</DialogTitle>
          <DialogDescription>
            {editingCheck.uuid ? `Editing ${editingCheck.name}` : 'Configure a new check definition'}
          </DialogDescription>
        </DialogHeader>

        {/* Scrollable content */}
        <div className="flex-1 overflow-y-auto px-6 py-5 space-y-6">
          {/* ─── General ─── */}
          <section className="space-y-4">
            <SectionHeader>General</SectionHeader>
            <div className="grid grid-cols-1 sm:grid-cols-3 gap-3">
              <div>
                <label className="text-xs text-muted-foreground">Name *</label>
                <Input
                  value={editingCheck.name || ''}
                  onChange={(e) => updateForm('name', e.target.value)}
                  className={errors.has('name') ? 'border-destructive' : ''}
                />
              </div>
              <div>
                <label className="text-xs text-muted-foreground">Project</label>
                <Input
                  value={editingCheck.project || ''}
                  onChange={(e) => updateForm('project', e.target.value)}
                  list="project-suggestions"
                  autoComplete="off"
                />
                <datalist id="project-suggestions">
                  {existingProjects.map((p) => (
                    <option key={p} value={p} />
                  ))}
                </datalist>
              </div>
              <div>
                <label className="text-xs text-muted-foreground">Group</label>
                <Input
                  value={editingCheck.group_name || ''}
                  onChange={(e) => updateForm('group_name', e.target.value)}
                  list="group-suggestions"
                  autoComplete="off"
                />
                <datalist id="group-suggestions">
                  {existingGroups.map((g) => (
                    <option key={g} value={g} />
                  ))}
                </datalist>
              </div>
            </div>
            <div className="grid grid-cols-1 sm:grid-cols-4 gap-3">
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
                    <SelectItem value="dns">DNS</SelectItem>
                    <SelectItem value="ssh">SSH</SelectItem>
                    <SelectItem value="passive">Passive</SelectItem>
                    <SelectItem value="redis">Redis</SelectItem>
                    <SelectItem value="mongodb">MongoDB</SelectItem>
                    <SelectItem value="domain_expiry">Domain Expiry</SelectItem>
                    <SelectItem value="mysql_query">MySQL Query</SelectItem>
                    <SelectItem value="mysql_query_unixtime">MySQL Query (Unixtime)</SelectItem>
                    <SelectItem value="mysql_replication">MySQL Replication</SelectItem>
                    <SelectItem value="pgsql_query">PostgreSQL Query</SelectItem>
                    <SelectItem value="pgsql_query_unixtime">PostgreSQL Query (Unixtime)</SelectItem>
                    <SelectItem value="pgsql_query_timestamp">PostgreSQL Query (Timestamp)</SelectItem>
                    <SelectItem value="pgsql_replication">PostgreSQL Replication</SelectItem>
                    <SelectItem value="pgsql_replication_status">PostgreSQL Replication Status</SelectItem>
                  </SelectContent>
                </Select>
              </div>
              <div>
                <label className="text-xs text-muted-foreground">Duration (frequency)</label>
                <Input
                  value={editingCheck.duration || ''}
                  onChange={(e) => updateForm('duration', e.target.value)}
                />
              </div>
              <div>
                <label className="text-xs text-muted-foreground">Timeout</label>
                <Input
                  value={editingCheck.timeout || ''}
                  onChange={(e) => updateForm('timeout', e.target.value)}
                />
              </div>
              <div className="flex items-end pb-1.5 gap-2">
                <Switch
                  checked={editingCheck.enabled ?? true}
                  onCheckedChange={(v) => updateForm('enabled', v)}
                />
                <label className="text-sm">Enabled</label>
              </div>
            </div>
            <div>
              <label className="text-xs text-muted-foreground">Description</label>
              <Input
                value={editingCheck.description || ''}
                onChange={(e) => updateForm('description', e.target.value)}
              />
            </div>
          </section>

          {/* ─── Connection ─── */}
          {hasConnection && (
            <section className="space-y-4">
              <SectionHeader>Connection</SectionHeader>

              {isHTTP && (
                <div>
                  <label className="text-xs text-muted-foreground">URL *</label>
                  <Input
                    value={editingCheck.url || ''}
                    onChange={(e) => updateForm('url', e.target.value)}
                    placeholder="https://example.com/health"
                    className={errors.has('url') ? 'border-destructive' : ''}
                  />
                </div>
              )}

              {isTCP && (
                <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
                  <div>
                    <label className="text-xs text-muted-foreground">Host *</label>
                    <Input
                      value={editingCheck.host || ''}
                      onChange={(e) => updateForm('host', e.target.value)}
                      className={errors.has('host') ? 'border-destructive' : ''}
                    />
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
              )}

              {isICMP && (
                <div>
                  <label className="text-xs text-muted-foreground">Host</label>
                  <Input
                    value={editingCheck.host || ''}
                    onChange={(e) => updateForm('host', e.target.value)}
                  />
                </div>
              )}

              {isDB && (
                <div className="space-y-3">
                  <div className="grid grid-cols-1 sm:grid-cols-3 gap-3">
                    <div>
                      <label className="text-xs text-muted-foreground">Host *</label>
                      <Input
                        value={editingCheck.host || ''}
                        onChange={(e) => updateForm('host', e.target.value)}
                        className={errors.has('host') ? 'border-destructive' : ''}
                      />
                    </div>
                    <div>
                      <label className="text-xs text-muted-foreground">Port</label>
                      <Input
                        type="number"
                        value={editingCheck.port || ''}
                        onChange={(e) => updateForm('port', parseInt(e.target.value) || 0)}
                      />
                    </div>
                    <div>
                      <label className="text-xs text-muted-foreground">Database Name</label>
                      <Input
                        value={dbConfig?.dbname || ''}
                        onChange={(e) => updateDBField('dbname', e.target.value)}
                      />
                    </div>
                  </div>
                  <div className="grid grid-cols-1 sm:grid-cols-3 gap-3">
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
                    {isPgSQL && (
                      <div>
                        <label className="text-xs text-muted-foreground">SSL Mode</label>
                        <Select
                          value={editingCheck.pgsql?.sslmode || 'disable'}
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
                </div>
              )}

              {isDNS && (
                <div className="space-y-3">
                  <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
                    <div>
                      <label className="text-xs text-muted-foreground">Domain *</label>
                      <Input
                        value={editingCheck.domain || ''}
                        onChange={(e) => updateForm('domain', e.target.value)}
                        placeholder="example.com"
                        className={errors.has('domain') ? 'border-destructive' : ''}
                      />
                    </div>
                    <div>
                      <label className="text-xs text-muted-foreground">Record Type</label>
                      <Select
                        value={editingCheck.record_type || 'A'}
                        onValueChange={(v) => updateForm('record_type', v)}
                      >
                        <SelectTrigger>
                          <SelectValue />
                        </SelectTrigger>
                        <SelectContent>
                          <SelectItem value="A">A</SelectItem>
                          <SelectItem value="AAAA">AAAA</SelectItem>
                          <SelectItem value="CNAME">CNAME</SelectItem>
                          <SelectItem value="MX">MX</SelectItem>
                          <SelectItem value="TXT">TXT</SelectItem>
                          <SelectItem value="NS">NS</SelectItem>
                        </SelectContent>
                      </Select>
                    </div>
                  </div>
                  <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
                    <div>
                      <label className="text-xs text-muted-foreground">DNS Resolver (optional)</label>
                      <Input
                        value={editingCheck.host || ''}
                        onChange={(e) => updateForm('host', e.target.value)}
                        placeholder="8.8.8.8"
                      />
                    </div>
                    <div>
                      <label className="text-xs text-muted-foreground">Expected Value (optional)</label>
                      <Input
                        value={editingCheck.expected || ''}
                        onChange={(e) => updateForm('expected', e.target.value)}
                        placeholder="Expected value in DNS response"
                      />
                    </div>
                  </div>
                </div>
              )}

              {isSSH && (
                <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
                  <div>
                    <label className="text-xs text-muted-foreground">Host *</label>
                    <Input
                      value={editingCheck.host || ''}
                      onChange={(e) => updateForm('host', e.target.value)}
                      className={errors.has('host') ? 'border-destructive' : ''}
                    />
                  </div>
                  <div>
                    <label className="text-xs text-muted-foreground">Port</label>
                    <Input
                      type="number"
                      value={editingCheck.port || 22}
                      onChange={(e) => updateForm('port', parseInt(e.target.value) || 22)}
                    />
                  </div>
                </div>
              )}

              {isRedis && (
                <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
                  <div>
                    <label className="text-xs text-muted-foreground">Host *</label>
                    <Input
                      value={editingCheck.host || ''}
                      onChange={(e) => updateForm('host', e.target.value)}
                      placeholder="localhost"
                      className={errors.has('host') ? 'border-destructive' : ''}
                    />
                  </div>
                  <div>
                    <label className="text-xs text-muted-foreground">Port</label>
                    <Input
                      type="number"
                      value={editingCheck.port || 6379}
                      onChange={(e) => updateForm('port', parseInt(e.target.value) || 6379)}
                    />
                  </div>
                </div>
              )}

              {isMongoDB && (
                <div>
                  <label className="text-xs text-muted-foreground">Connection URI *</label>
                  <Input
                    value={editingCheck.mongodb_uri || ''}
                    onChange={(e) => updateForm('mongodb_uri', e.target.value)}
                    placeholder="mongodb://user:pass@host:27017/db"
                    className={errors.has('mongodb_uri') ? 'border-destructive' : ''}
                  />
                </div>
              )}

              {isDomainExpiry && (
                <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
                  <div>
                    <label className="text-xs text-muted-foreground">Domain *</label>
                    <Input
                      value={editingCheck.domain || ''}
                      onChange={(e) => updateForm('domain', e.target.value)}
                      placeholder="example.com"
                      className={errors.has('domain') ? 'border-destructive' : ''}
                    />
                  </div>
                  <div>
                    <label className="text-xs text-muted-foreground">Warning Days Before Expiry</label>
                    <Input
                      type="number"
                      value={editingCheck.expiry_warning_days || 30}
                      onChange={(e) => updateForm('expiry_warning_days', parseInt(e.target.value) || 30)}
                    />
                  </div>
                </div>
              )}
            </section>
          )}

          {/* ─── Advanced Settings ─── */}
          {hasAdvanced && (
            <section>
              <Collapsible open={advancedOpen} onOpenChange={setAdvancedOpen}>
                <CollapsibleTrigger asChild>
                  <button
                    type="button"
                    className="flex items-center gap-2 w-full text-sm font-semibold border-b border-border pb-2 pt-1 hover:text-foreground transition-colors text-left"
                  >
                    {advancedOpen ? (
                      <ChevronDown className="h-4 w-4" />
                    ) : (
                      <ChevronRight className="h-4 w-4" />
                    )}
                    Advanced Settings
                  </button>
                </CollapsibleTrigger>
                <CollapsibleContent className="space-y-4 pt-4">
                  {/* HTTP Advanced */}
                  {isHTTP && (
                    <div className="space-y-4">
                      <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
                        <div>
                          <label className="text-xs text-muted-foreground">
                            Expected Status Codes (comma-separated)
                          </label>
                          <Input
                            value={codeString}
                            onChange={(e) => {
                              const codes = e.target.value
                                .split(',')
                                .map((s) => parseInt(s.trim()))
                                .filter((n) => !isNaN(n))
                              updateForm('code', codes)
                            }}
                            placeholder="200, 201, 301"
                          />
                        </div>
                        <div>
                          <label className="text-xs text-muted-foreground">SSL Expiration Period</label>
                          <Input
                            value={editingCheck.ssl_expiration_period || ''}
                            onChange={(e) => updateForm('ssl_expiration_period', e.target.value)}
                            placeholder="720h"
                          />
                        </div>
                      </div>

                      <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
                        <div>
                          <label className="text-xs text-muted-foreground">Response Body Match</label>
                          <Input
                            value={editingCheck.answer || ''}
                            onChange={(e) => updateForm('answer', e.target.value)}
                            placeholder="Expected string in response body"
                          />
                        </div>
                        <div className="flex items-end pb-1.5 gap-2">
                          <Switch
                            checked={editingCheck.answer_present ?? false}
                            onCheckedChange={(v) => updateForm('answer_present', v)}
                          />
                          <label className="text-sm">Answer must be present</label>
                        </div>
                      </div>

                      <div className="grid grid-cols-1 sm:grid-cols-3 gap-3">
                        <div className="flex items-center gap-2">
                          <Switch
                            checked={editingCheck.skip_check_ssl ?? false}
                            onCheckedChange={(v) => updateForm('skip_check_ssl', v)}
                          />
                          <label className="text-sm">Skip SSL Verification</label>
                        </div>
                        <div className="flex items-center gap-2">
                          <Switch
                            checked={editingCheck.stop_follow_redirects ?? false}
                            onCheckedChange={(v) => updateForm('stop_follow_redirects', v)}
                          />
                          <label className="text-sm">Stop Following Redirects</label>
                        </div>
                      </div>

                      <KeyValueEditor
                        label="Headers"
                        values={editingCheck.headers || []}
                        onChange={(v) => updateForm('headers', v)}
                        keyPlaceholder="Header name"
                        valuePlaceholder="Header value"
                      />

                      <KeyValueEditor
                        label="Cookies"
                        values={editingCheck.cookies || []}
                        onChange={(v) => updateForm('cookies', v)}
                        keyPlaceholder="Cookie name"
                        valuePlaceholder="Cookie value"
                      />

                      <div className="space-y-2">
                        <label className="text-xs text-muted-foreground font-medium">Basic Auth</label>
                        <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
                          <div>
                            <label className="text-xs text-muted-foreground">Username</label>
                            <Input
                              value={editingCheck.auth?.user || ''}
                              onChange={(e) => updateAuth('user', e.target.value)}
                            />
                          </div>
                          <div>
                            <label className="text-xs text-muted-foreground">Password</label>
                            <Input
                              type="password"
                              value={editingCheck.auth?.password || ''}
                              onChange={(e) => updateAuth('password', e.target.value)}
                            />
                          </div>
                        </div>
                      </div>
                    </div>
                  )}

                  {/* ICMP Advanced */}
                  {isICMP && (
                    <div>
                      <label className="text-xs text-muted-foreground">Ping Count</label>
                      <Input
                        type="number"
                        value={editingCheck.count || ''}
                        onChange={(e) => updateForm('count', parseInt(e.target.value) || 0)}
                        placeholder="Number of pings"
                      />
                    </div>
                  )}

                  {/* SSH Advanced */}
                  {isSSH && (
                    <div>
                      <label className="text-xs text-muted-foreground">Expected Banner (optional)</label>
                      <Input
                        value={editingCheck.expect_banner || ''}
                        onChange={(e) => updateForm('expect_banner', e.target.value)}
                        placeholder="SSH-2.0-OpenSSH"
                      />
                    </div>
                  )}

                  {/* Redis Advanced */}
                  {isRedis && (
                    <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
                      <div>
                        <label className="text-xs text-muted-foreground">Password</label>
                        <Input
                          type="password"
                          value={editingCheck.redis_password || ''}
                          onChange={(e) => updateForm('redis_password', e.target.value)}
                        />
                      </div>
                      <div>
                        <label className="text-xs text-muted-foreground">Database Number</label>
                        <Input
                          type="number"
                          value={editingCheck.redis_db ?? 0}
                          onChange={(e) => updateForm('redis_db', parseInt(e.target.value) || 0)}
                        />
                      </div>
                    </div>
                  )}

                  {/* DNS Advanced — expected value already in connection section */}
                  {isDNS && (
                    <div className="text-xs text-muted-foreground">
                      DNS advanced options are configured in the Connection section above.
                    </div>
                  )}

                  {/* Database Advanced */}
                  {isDB && (
                    <div className="space-y-4">
                      <h4 className="text-xs font-medium text-muted-foreground uppercase tracking-wide">
                        Query Settings
                      </h4>
                      <div>
                        <label className="text-xs text-muted-foreground">Query</label>
                        <textarea
                          className="flex w-full rounded-md border border-input bg-background px-3 py-2 text-sm font-mono ring-offset-background placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 min-h-[80px] resize-y"
                          value={dbConfig?.query || ''}
                          onChange={(e) => updateDBField('query', e.target.value)}
                          placeholder="SELECT 1"
                        />
                      </div>
                      <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
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

                      <h4 className="text-xs font-medium text-muted-foreground uppercase tracking-wide pt-2">
                        Replication Settings
                      </h4>
                      <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
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
                    </div>
                  )}
                </CollapsibleContent>
              </Collapsible>
            </section>
          )}

          {/* ─── Alert Configuration ─── */}
          <section className="space-y-4">
            <SectionHeader>Alert Configuration</SectionHeader>
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
                  placeholder={
                    editingCheck.alert_type === 'slack'
                      ? '#channel or @user'
                      : editingCheck.alert_type === 'webhook'
                        ? 'https://webhook.example.com/...'
                        : 'Select an alert type first'
                  }
                />
              </div>
            </div>
          </section>

          {/* ─── Maintenance Window ─── */}
          {editingCheck.uuid && (
            <MaintenanceSection
              uuid={editingCheck.uuid}
              maintenanceUntil={editingCheck.maintenance_until}
              onMaintenanceChange={(v) => updateForm('maintenance_until', v as string)}
            />
          )}
        </div>

        {/* Sticky footer */}
        <div className="flex-shrink-0 flex items-center justify-end gap-2 px-6 py-4 border-t bg-background">
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            Cancel
          </Button>
          <Button onClick={handleSave} disabled={saving}>
            {saving ? 'Saving...' : editingCheck.uuid ? 'Update' : 'Create'}
          </Button>
        </div>
      </DialogContent>
    </Dialog>
  )
}
