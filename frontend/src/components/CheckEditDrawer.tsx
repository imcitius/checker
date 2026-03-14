import { useState } from 'react'
import { type CheckDefinition } from '@/lib/api'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Badge } from '@/components/ui/badge'
import {
  Sheet,
  SheetContent,
  SheetHeader,
  SheetTitle,
  SheetDescription,
  SheetFooter,
} from '@/components/ui/sheet'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Switch } from '@/components/ui/switch'
import { Tabs, TabsList, TabsTrigger, TabsContent } from '@/components/ui/tabs'
import { Plus, X, AlertCircle } from 'lucide-react'

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

type TabId = 'general' | 'connection' | 'advanced' | 'alerting'

interface CheckEditDrawerProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  editingCheck: Partial<CheckDefinition>
  onCheckChange: (check: Partial<CheckDefinition>) => void
  onSave: () => void
  saving: boolean
}

export function CheckEditDrawer({
  open,
  onOpenChange,
  editingCheck,
  onCheckChange,
  onSave,
  saving,
}: CheckEditDrawerProps) {
  const [activeTab, setActiveTab] = useState<TabId>('general')
  const [tabErrors, setTabErrors] = useState<Set<TabId>>(new Set())

  const isMySQL = editingCheck.type?.includes('mysql')
  const isPgSQL = editingCheck.type?.includes('pgsql')
  const isDB = isMySQL || isPgSQL
  const isHTTP = editingCheck.type === 'http'
  const isTCP = editingCheck.type === 'tcp'
  const isICMP = editingCheck.type === 'icmp'

  const dbKey = isMySQL ? 'mysql' : 'pgsql'
  const dbConfig = isMySQL ? editingCheck.mysql : editingCheck.pgsql

  const updateForm = (field: string, value: string | number | boolean) => {
    onCheckChange({ ...editingCheck, [field]: value })
  }

  const updateDBField = (field: string, value: string | string[]) => {
    onCheckChange({
      ...editingCheck,
      [dbKey]: { ...(isMySQL ? editingCheck.mysql : editingCheck.pgsql), [field]: value },
    })
  }

  const validate = (): boolean => {
    const errors = new Set<TabId>()

    // General tab: name is required
    if (!editingCheck.name?.trim()) {
      errors.add('general')
    }

    // Connection tab: type-specific required fields
    if (isHTTP && !editingCheck.url?.trim()) {
      errors.add('connection')
    }
    if ((isTCP || isDB) && !editingCheck.host?.trim()) {
      errors.add('connection')
    }

    setTabErrors(errors)
    if (errors.size > 0) {
      // Switch to first tab with errors
      const firstError = ['general', 'connection', 'advanced', 'alerting'].find((t) =>
        errors.has(t as TabId)
      ) as TabId
      if (firstError) setActiveTab(firstError)
      return false
    }
    return true
  }

  const handleSave = () => {
    if (validate()) {
      onSave()
    }
  }

  // Determine if connection/advanced tabs have content for the current type
  const hasConnectionTab = isHTTP || isTCP || isICMP || isDB
  const hasAdvancedTab = isHTTP || isDB

  const TabErrorDot = ({ tab }: { tab: TabId }) =>
    tabErrors.has(tab) ? (
      <AlertCircle className="h-3 w-3 ml-1 text-red-500" />
    ) : null

  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent side="right" className="flex flex-col p-0">
        <SheetHeader className="flex-shrink-0">
          <SheetTitle>{editingCheck.uuid ? 'Edit Check' : 'Create New Check'}</SheetTitle>
          <SheetDescription>
            {editingCheck.uuid ? `Editing ${editingCheck.name}` : 'Configure a new check definition'}
          </SheetDescription>
        </SheetHeader>

        <Tabs
          value={activeTab}
          onValueChange={(v) => setActiveTab(v as TabId)}
          className="flex flex-col flex-1 min-h-0"
        >
          <div className="flex-shrink-0 px-6">
            <TabsList className="w-full grid grid-cols-4">
              <TabsTrigger value="general" className="text-xs">
                General <TabErrorDot tab="general" />
              </TabsTrigger>
              <TabsTrigger value="connection" className="text-xs">
                Connection <TabErrorDot tab="connection" />
              </TabsTrigger>
              <TabsTrigger value="advanced" className="text-xs">
                Advanced <TabErrorDot tab="advanced" />
              </TabsTrigger>
              <TabsTrigger value="alerting" className="text-xs">
                Alerting <TabErrorDot tab="alerting" />
              </TabsTrigger>
            </TabsList>
          </div>

          <div className="flex-1 overflow-y-auto px-6 py-4">
            {/* ──────────── General Tab ──────────── */}
            <TabsContent value="general" className="mt-0 space-y-4">
              <div className="grid grid-cols-2 gap-3">
                <div>
                  <label className="text-xs text-muted-foreground">Name *</label>
                  <Input
                    value={editingCheck.name || ''}
                    onChange={(e) => updateForm('name', e.target.value)}
                    className={tabErrors.has('general') && !editingCheck.name?.trim() ? 'border-red-500' : ''}
                  />
                </div>
                <div>
                  <label className="text-xs text-muted-foreground">Project</label>
                  <Input
                    value={editingCheck.project || ''}
                    onChange={(e) => updateForm('project', e.target.value)}
                  />
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
            </TabsContent>

            {/* ──────────── Connection Tab ──────────── */}
            <TabsContent value="connection" className="mt-0 space-y-4">
              {isHTTP && (
                <div>
                  <h4 className="text-sm font-medium mb-3">HTTP Connection</h4>
                  <div>
                    <label className="text-xs text-muted-foreground">URL *</label>
                    <Input
                      value={editingCheck.url || ''}
                      onChange={(e) => updateForm('url', e.target.value)}
                      placeholder="https://example.com/health"
                      className={tabErrors.has('connection') && !editingCheck.url?.trim() ? 'border-red-500' : ''}
                    />
                  </div>
                </div>
              )}

              {isTCP && (
                <div>
                  <h4 className="text-sm font-medium mb-3">TCP Connection</h4>
                  <div className="grid grid-cols-2 gap-3">
                    <div>
                      <label className="text-xs text-muted-foreground">Host *</label>
                      <Input
                        value={editingCheck.host || ''}
                        onChange={(e) => updateForm('host', e.target.value)}
                        className={tabErrors.has('connection') && !editingCheck.host?.trim() ? 'border-red-500' : ''}
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
                </div>
              )}

              {isICMP && (
                <div>
                  <h4 className="text-sm font-medium mb-3">ICMP Connection</h4>
                  <div>
                    <label className="text-xs text-muted-foreground">Host</label>
                    <Input
                      value={editingCheck.host || ''}
                      onChange={(e) => updateForm('host', e.target.value)}
                    />
                  </div>
                </div>
              )}

              {isDB && (
                <div className="space-y-3">
                  <h4 className="text-sm font-medium mb-3">Database Connection</h4>
                  <div className="grid grid-cols-2 gap-3">
                    <div>
                      <label className="text-xs text-muted-foreground">Host *</label>
                      <Input
                        value={editingCheck.host || ''}
                        onChange={(e) => updateForm('host', e.target.value)}
                        className={tabErrors.has('connection') && !editingCheck.host?.trim() ? 'border-red-500' : ''}
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

              {!hasConnectionTab && (
                <div className="text-center py-8 text-muted-foreground text-sm">
                  No connection settings for this check type.
                </div>
              )}
            </TabsContent>

            {/* ──────────── Advanced Tab ──────────── */}
            <TabsContent value="advanced" className="mt-0 space-y-4">
              {isDB && (
                <div className="space-y-4">
                  <h4 className="text-sm font-medium">Query Settings</h4>
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

                  <h4 className="text-sm font-medium pt-2">Replication Settings</h4>
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
                </div>
              )}

              {!hasAdvancedTab && (
                <div className="text-center py-8 text-muted-foreground text-sm">
                  No advanced settings for this check type.
                </div>
              )}
            </TabsContent>

            {/* ──────────── Alerting Tab ──────────── */}
            <TabsContent value="alerting" className="mt-0 space-y-4">
              <h4 className="text-sm font-medium mb-3">Alert Configuration</h4>
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
            </TabsContent>
          </div>
        </Tabs>

        <SheetFooter className="flex-shrink-0">
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            Cancel
          </Button>
          <Button onClick={handleSave} disabled={saving}>
            {saving ? 'Saving...' : editingCheck.uuid ? 'Update' : 'Create'}
          </Button>
        </SheetFooter>
      </SheetContent>
    </Sheet>
  )
}
