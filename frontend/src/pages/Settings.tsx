import { useState, useEffect, useCallback, useRef } from 'react'
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
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { Switch } from '@/components/ui/switch'
import { TopBar } from '@/components/TopBar'
import { api } from '@/lib/api'
import type { AlertChannel, AlertChannelInput } from '@/lib/api'
import { toast } from 'sonner'
import {
  Plus,
  Pencil,
  Trash2,
  Play,
  MoreVertical,
  MessageSquare,
  Hash,
  Mail,
  Megaphone,
  Users,
  AlertTriangle,
  Eye,
  Loader2,
  RefreshCw,
  Search,
  Settings as SettingsIcon,
  Bell,
} from 'lucide-react'

// Channel type metadata
const CHANNEL_TYPES = [
  { value: 'telegram', label: 'Telegram', icon: MessageSquare, color: 'bg-blue-500' },
  { value: 'slack', label: 'Slack App', icon: Hash, color: 'bg-purple-500' },
  { value: 'slack_webhook', label: 'Slack Webhook', icon: Hash, color: 'bg-purple-400' },
  { value: 'email', label: 'Email', icon: Mail, color: 'bg-green-500' },
  { value: 'discord', label: 'Discord', icon: Megaphone, color: 'bg-indigo-500' },
  { value: 'teams', label: 'Teams', icon: Users, color: 'bg-blue-600' },
  { value: 'pagerduty', label: 'PagerDuty', icon: AlertTriangle, color: 'bg-emerald-500' },
  { value: 'opsgenie', label: 'Opsgenie', icon: Eye, color: 'bg-cyan-500' },
  { value: 'ntfy', label: 'ntfy', icon: Bell, color: 'bg-amber-500' },
] as const

type ChannelType = (typeof CHANNEL_TYPES)[number]['value']

// Config field definitions per channel type
interface ConfigField {
  key: string
  label: string
  type: 'text' | 'password' | 'number' | 'email' | 'tags' | 'toggle'
  placeholder?: string
  required?: boolean
}

const CONFIG_FIELDS: Record<ChannelType, ConfigField[]> = {
  telegram: [
    { key: 'bot_token', label: 'Bot Token', type: 'password', placeholder: 'e.g. 123456:ABC-DEF1234...', required: true },
    { key: 'chat_id', label: 'Chat ID', type: 'text', placeholder: 'e.g. -1001234567890', required: true },
  ],
  slack: [
    { key: 'bot_token', label: 'Bot Token', type: 'password', placeholder: 'xoxb-...', required: true },
    { key: 'signing_secret', label: 'Signing Secret', type: 'password', placeholder: 'Slack app signing secret', required: true },
    { key: 'default_channel', label: 'Default Channel', type: 'text', placeholder: 'Channel ID (C01ABCDEF) or name (general)', required: true },
  ],
  slack_webhook: [
    { key: 'webhook_url', label: 'Webhook URL', type: 'password', placeholder: 'https://hooks.slack.com/services/...', required: true },
  ],
  email: [
    { key: 'smtp_host', label: 'SMTP Host', type: 'text', placeholder: 'smtp.gmail.com', required: true },
    { key: 'smtp_port', label: 'SMTP Port', type: 'number', placeholder: '587', required: true },
    { key: 'smtp_user', label: 'SMTP User', type: 'text', placeholder: 'user@example.com' },
    { key: 'smtp_password', label: 'SMTP Password', type: 'password', placeholder: 'password' },
    { key: 'from', label: 'From Address', type: 'email', placeholder: 'alerts@example.com', required: true },
    { key: 'to', label: 'To Addresses', type: 'tags', placeholder: 'recipient@example.com', required: true },
    { key: 'use_tls', label: 'Use TLS', type: 'toggle' },
  ],
  discord: [
    { key: 'webhook_url', label: 'Webhook URL', type: 'password', placeholder: 'https://discord.com/api/webhooks/...', required: true },
  ],
  teams: [
    { key: 'webhook_url', label: 'Webhook URL', type: 'password', placeholder: 'https://outlook.office.com/webhook/...', required: true },
  ],
  pagerduty: [
    { key: 'routing_key', label: 'Routing Key', type: 'password', placeholder: 'Events API v2 routing key', required: true },
  ],
  opsgenie: [
    { key: 'api_key', label: 'API Key', type: 'password', placeholder: 'Opsgenie API key', required: true },
    { key: 'region', label: 'Region', type: 'text', placeholder: 'us or eu' },
  ],
  ntfy: [
    { key: 'server_url', label: 'Server URL', type: 'text', placeholder: 'https://ntfy.sh (default)' },
    { key: 'topic', label: 'Topic', type: 'text', placeholder: 'my-alerts', required: true },
    { key: 'token', label: 'Access Token', type: 'password', placeholder: 'Bearer token (optional)' },
    { key: 'username', label: 'Username', type: 'text', placeholder: 'Basic auth username (optional)' },
    { key: 'password', label: 'Password', type: 'password', placeholder: 'Basic auth password (optional)' },
    { key: 'icon', label: 'Icon URL', type: 'text', placeholder: 'https://example.com/icon.png (optional)' },
    { key: 'click_url', label: 'Checker URL', type: 'text', placeholder: 'https://checker.example.com (optional, for action buttons)' },
  ],
}

function getChannelMeta(type: string) {
  return CHANNEL_TYPES.find((ct) => ct.value === type) || CHANNEL_TYPES[0]
}

function isMasked(value: unknown): boolean {
  return typeof value === 'string' && value.includes('****')
}

export function Settings() {
  const [channels, setChannels] = useState<AlertChannel[]>([])
  const [loading, setLoading] = useState(true)
  const [dialogOpen, setDialogOpen] = useState(false)
  const [editingChannel, setEditingChannel] = useState<AlertChannel | null>(null)
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false)
  const [deletingChannel, setDeletingChannel] = useState<AlertChannel | null>(null)
  const [testingChannel, setTestingChannel] = useState<string | null>(null)
  const [saving, setSaving] = useState(false)

  // Form state
  const [formName, setFormName] = useState('')
  const [formType, setFormType] = useState<ChannelType>('telegram')
  const [formConfig, setFormConfig] = useState<Record<string, unknown>>({})

  // TopBar state (minimal - Settings page doesn't use most TopBar features)
  const [search, setSearch] = useState('')
  const searchRef = useRef<HTMLInputElement>(null)

  const fetchChannels = useCallback(async () => {
    try {
      const data = await api.getAlertChannels()
      setChannels(Array.isArray(data) ? data : [])
    } catch (err) {
      toast.error('Failed to load alert channels')
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    fetchChannels()
  }, [fetchChannels])

  const openCreateDialog = () => {
    setEditingChannel(null)
    setFormName('')
    setFormType('telegram')
    setFormConfig({})
    setDialogOpen(true)
  }

  const openEditDialog = (channel: AlertChannel) => {
    setEditingChannel(channel)
    setFormName(channel.name)
    setFormType(channel.type as ChannelType)
    setFormConfig({ ...channel.config })
    setDialogOpen(true)
  }

  const openDeleteDialog = (channel: AlertChannel) => {
    setDeletingChannel(channel)
    setDeleteDialogOpen(true)
  }

  const handleSave = async () => {
    if (!formName.trim()) {
      toast.error('Channel name is required')
      return
    }

    const fields = CONFIG_FIELDS[formType]
    for (const field of fields) {
      if (field.required) {
        const value = formConfig[field.key]
        if (field.type === 'tags') {
          if (!Array.isArray(value) || value.length === 0) {
            toast.error(`${field.label} is required`)
            return
          }
        } else if (!value && value !== 0 && value !== false) {
          // When editing, masked values are OK
          if (editingChannel && isMasked(value)) continue
          toast.error(`${field.label} is required`)
          return
        }
      }
    }

    setSaving(true)
    try {
      const input: AlertChannelInput = {
        name: formName.trim(),
        type: formType,
        config: formConfig,
      }

      if (editingChannel) {
        await api.updateAlertChannel(editingChannel.name, input)
        toast.success(`Channel "${formName}" updated`)
      } else {
        await api.createAlertChannel(input)
        toast.success(`Channel "${formName}" created`)
      }

      setDialogOpen(false)
      fetchChannels()
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Failed to save channel')
    } finally {
      setSaving(false)
    }
  }

  const handleDelete = async () => {
    if (!deletingChannel) return

    try {
      await api.deleteAlertChannel(deletingChannel.name)
      toast.success(`Channel "${deletingChannel.name}" deleted`)
      setDeleteDialogOpen(false)
      setDeletingChannel(null)
      fetchChannels()
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Failed to delete channel')
    }
  }

  const handleTest = async (channel: AlertChannel) => {
    setTestingChannel(channel.name)
    try {
      const result = await api.testAlertChannel(channel.name)
      if (result.success) {
        toast.success(`Test notification sent to "${channel.name}"`)
      } else {
        toast.error('Test notification failed')
      }
    } catch (err) {
      let message = 'Test notification failed'
      if (err instanceof Error) {
        try {
          const parsed = JSON.parse(err.message)
          message = parsed.error || err.message
        } catch {
          message = err.message
        }
      }
      toast.error(message)
    } finally {
      setTestingChannel(null)
    }
  }

  const setConfigField = (key: string, value: unknown) => {
    setFormConfig((prev) => ({ ...prev, [key]: value }))
  }

  const filteredChannels = channels.filter(
    (ch) =>
      !search ||
      ch.name.toLowerCase().includes(search.toLowerCase()) ||
      ch.type.toLowerCase().includes(search.toLowerCase())
  )

  return (
    <div className="min-h-screen bg-background">
      <TopBar
        search={search}
        onSearchChange={setSearch}
        statusFilter="all"
        onStatusFilterChange={() => {}}
        projectFilter="all"
        onProjectFilterChange={() => {}}
        typeFilter="all"
        onTypeFilterChange={() => {}}
        projects={[]}
        checkTypes={[]}
        searchRef={searchRef}
      />

      <main className="mx-auto max-w-[1600px] px-4 py-6">
        {/* Page header */}
        <div className="flex items-center justify-between mb-6">
          <div>
            <h1 className="text-2xl font-bold text-foreground flex items-center gap-2">
              <SettingsIcon className="h-6 w-6" />
              Settings
            </h1>
            <p className="text-sm text-muted-foreground mt-1">
              Manage notification channels and system configuration
            </p>
          </div>
        </div>

        {/* Notification Channels section */}
        <section className="mb-8">
          <div className="flex items-center justify-between mb-4">
            <h2 className="text-lg font-semibold text-foreground">Notification Channels</h2>
            <div className="flex items-center gap-2">
              <Button variant="outline" size="sm" onClick={fetchChannels} disabled={loading}>
                <RefreshCw className={`h-4 w-4 mr-1 ${loading ? 'animate-spin' : ''}`} />
                Refresh
              </Button>
              <Button size="sm" onClick={openCreateDialog}>
                <Plus className="h-4 w-4 mr-1" />
                Add Channel
              </Button>
            </div>
          </div>

          {loading ? (
            <div className="flex items-center justify-center py-12">
              <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
              <span className="ml-2 text-muted-foreground">Loading channels...</span>
            </div>
          ) : filteredChannels.length === 0 ? (
            <div className="border rounded-lg p-8 text-center">
              <MessageSquare className="h-10 w-10 mx-auto text-muted-foreground mb-3" />
              <p className="text-muted-foreground">
                {channels.length === 0
                  ? 'No notification channels configured yet.'
                  : 'No channels match your search.'}
              </p>
              {channels.length === 0 && (
                <Button className="mt-4" onClick={openCreateDialog}>
                  <Plus className="h-4 w-4 mr-1" />
                  Add your first channel
                </Button>
              )}
            </div>
          ) : (
            <div className="grid gap-3">
              {filteredChannels.map((channel) => {
                const meta = getChannelMeta(channel.type)
                const Icon = meta.icon
                const isTesting = testingChannel === channel.name

                return (
                  <div
                    key={channel.id}
                    className="border rounded-lg p-4 bg-card hover:border-foreground/20 transition-colors"
                  >
                    <div className="flex items-center gap-3">
                      {/* Channel type icon */}
                      <div
                        className={`h-10 w-10 rounded-lg ${meta.color} flex items-center justify-center shrink-0`}
                      >
                        <Icon className="h-5 w-5 text-white" />
                      </div>

                      {/* Channel info */}
                      <div className="flex-1 min-w-0">
                        <div className="flex items-center gap-2">
                          <span className="font-medium text-foreground truncate">{channel.name}</span>
                          <Badge variant="outline" className="text-xs shrink-0">
                            {meta.label}
                          </Badge>
                        </div>
                        <div className="text-xs text-muted-foreground mt-0.5">
                          <ConfigSummary channel={channel} />
                        </div>
                      </div>

                      {/* Actions */}
                      <div className="flex items-center gap-1 shrink-0">
                        <Button
                          variant="outline"
                          size="sm"
                          onClick={() => handleTest(channel)}
                          disabled={isTesting}
                          className="hidden sm:flex"
                        >
                          {isTesting ? (
                            <Loader2 className="h-3.5 w-3.5 mr-1 animate-spin" />
                          ) : (
                            <Play className="h-3.5 w-3.5 mr-1" />
                          )}
                          Test
                        </Button>

                        <DropdownMenu>
                          <DropdownMenuTrigger asChild>
                            <Button variant="ghost" size="icon" className="h-8 w-8">
                              <MoreVertical className="h-4 w-4" />
                            </Button>
                          </DropdownMenuTrigger>
                          <DropdownMenuContent align="end">
                            <DropdownMenuItem onClick={() => openEditDialog(channel)}>
                              <Pencil className="h-4 w-4 mr-2" />
                              Edit
                            </DropdownMenuItem>
                            <DropdownMenuItem
                              onClick={() => handleTest(channel)}
                              disabled={isTesting}
                              className="sm:hidden"
                            >
                              <Play className="h-4 w-4 mr-2" />
                              Test
                            </DropdownMenuItem>
                            <DropdownMenuItem
                              onClick={() => openDeleteDialog(channel)}
                              className="text-destructive focus:text-destructive"
                            >
                              <Trash2 className="h-4 w-4 mr-2" />
                              Delete
                            </DropdownMenuItem>
                          </DropdownMenuContent>
                        </DropdownMenu>
                      </div>
                    </div>
                  </div>
                )
              })}
            </div>
          )}
        </section>
      </main>

      {/* Create/Edit Dialog */}
      <Dialog open={dialogOpen} onOpenChange={setDialogOpen}>
        <DialogContent className="max-w-lg max-h-[90vh] overflow-y-auto">
          <DialogHeader>
            <DialogTitle>
              {editingChannel ? 'Edit Alert Channel' : 'Add Alert Channel'}
            </DialogTitle>
            <DialogDescription>
              {editingChannel
                ? 'Update the notification channel configuration.'
                : 'Configure a new notification channel for alerts.'}
            </DialogDescription>
          </DialogHeader>

          <div className="space-y-4 py-2">
            {/* Channel name */}
            <div>
              <label className="text-sm font-medium text-foreground">Name</label>
              <Input
                value={formName}
                onChange={(e) => setFormName(e.target.value)}
                placeholder="e.g. production-slack"
                className="mt-1"
                disabled={!!editingChannel}
              />
              {editingChannel && (
                <p className="text-xs text-muted-foreground mt-1">
                  Channel name cannot be changed after creation.
                </p>
              )}
            </div>

            {/* Channel type */}
            <div>
              <label className="text-sm font-medium text-foreground">Type</label>
              <Select
                value={formType}
                onValueChange={(v) => {
                  setFormType(v as ChannelType)
                  if (!editingChannel) setFormConfig({})
                }}
                disabled={!!editingChannel}
              >
                <SelectTrigger className="mt-1">
                  <SelectValue placeholder="Select channel type" />
                </SelectTrigger>
                <SelectContent>
                  {CHANNEL_TYPES.map((ct) => {
                    const Icon = ct.icon
                    return (
                      <SelectItem key={ct.value} value={ct.value}>
                        <span className="flex items-center gap-2">
                          <Icon className="h-4 w-4" />
                          {ct.label}
                        </span>
                      </SelectItem>
                    )
                  })}
                </SelectContent>
              </Select>
            </div>

            {/* Dynamic config fields */}
            {CONFIG_FIELDS[formType]?.map((field) => (
              <ConfigFieldInput
                key={field.key}
                field={field}
                value={formConfig[field.key]}
                onChange={(v) => setConfigField(field.key, v)}
              />
            ))}
          </div>

          <DialogFooter>
            <Button variant="outline" onClick={() => setDialogOpen(false)}>
              Cancel
            </Button>
            {editingChannel && (
              <Button
                variant="outline"
                onClick={() => handleTest(editingChannel)}
                disabled={testingChannel === editingChannel.name}
              >
                {testingChannel === editingChannel.name ? (
                  <Loader2 className="h-3.5 w-3.5 mr-1 animate-spin" />
                ) : (
                  <Play className="h-3.5 w-3.5 mr-1" />
                )}
                Test
              </Button>
            )}
            <Button onClick={handleSave} disabled={saving}>
              {saving && <Loader2 className="h-4 w-4 mr-1 animate-spin" />}
              {editingChannel ? 'Update' : 'Create'}
            </Button>
          </DialogFooter>
          {!editingChannel && (
            <p className="text-xs text-muted-foreground text-center mt-2">
              Save the channel first, then use the Test button to verify it works.
            </p>
          )}
        </DialogContent>
      </Dialog>

      {/* Delete Confirmation Dialog */}
      <Dialog open={deleteDialogOpen} onOpenChange={setDeleteDialogOpen}>
        <DialogContent className="max-w-sm">
          <DialogHeader>
            <DialogTitle>Delete Channel</DialogTitle>
            <DialogDescription>
              Are you sure you want to delete &quot;{deletingChannel?.name}&quot;? This action cannot
              be undone.
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button variant="outline" onClick={() => setDeleteDialogOpen(false)}>
              Cancel
            </Button>
            <Button variant="destructive" onClick={handleDelete}>
              <Trash2 className="h-4 w-4 mr-1" />
              Delete
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}

// ConfigFieldInput renders a form input for a config field
function ConfigFieldInput({
  field,
  value,
  onChange,
}: {
  field: ConfigField
  value: unknown
  onChange: (v: unknown) => void
}) {
  if (field.type === 'toggle') {
    return (
      <div className="flex items-center justify-between">
        <label className="text-sm font-medium text-foreground">{field.label}</label>
        <Switch
          checked={!!value}
          onCheckedChange={(checked) => onChange(checked)}
        />
      </div>
    )
  }

  if (field.type === 'tags') {
    return <TagsInput field={field} value={value} onChange={onChange} />
  }

  return (
    <div>
      <label className="text-sm font-medium text-foreground">
        {field.label}
        {field.required && <span className="text-destructive ml-0.5">*</span>}
      </label>
      <Input
        type={field.type === 'password' ? 'password' : field.type === 'number' ? 'number' : 'text'}
        value={typeof value === 'string' || typeof value === 'number' ? String(value) : ''}
        onChange={(e) => {
          const v = field.type === 'number' ? Number(e.target.value) || 0 : e.target.value
          onChange(v)
        }}
        placeholder={field.placeholder}
        className="mt-1"
      />
    </div>
  )
}

// TagsInput renders a multi-value input (for email "to" addresses)
function TagsInput({
  field,
  value,
  onChange,
}: {
  field: ConfigField
  value: unknown
  onChange: (v: unknown) => void
}) {
  const [inputValue, setInputValue] = useState('')
  const tags = Array.isArray(value) ? (value as string[]) : []

  const addTag = () => {
    const trimmed = inputValue.trim()
    if (trimmed && !tags.includes(trimmed)) {
      onChange([...tags, trimmed])
      setInputValue('')
    }
  }

  const removeTag = (index: number) => {
    onChange(tags.filter((_, i) => i !== index))
  }

  return (
    <div>
      <label className="text-sm font-medium text-foreground">
        {field.label}
        {field.required && <span className="text-destructive ml-0.5">*</span>}
      </label>
      <div className="mt-1 space-y-2">
        {tags.length > 0 && (
          <div className="flex flex-wrap gap-1">
            {tags.map((tag, i) => (
              <Badge key={i} variant="secondary" className="text-xs gap-1">
                {tag}
                <button
                  onClick={() => removeTag(i)}
                  className="ml-1 hover:text-destructive"
                  type="button"
                >
                  &times;
                </button>
              </Badge>
            ))}
          </div>
        )}
        <div className="flex gap-2">
          <Input
            value={inputValue}
            onChange={(e) => setInputValue(e.target.value)}
            placeholder={field.placeholder}
            onKeyDown={(e) => {
              if (e.key === 'Enter') {
                e.preventDefault()
                addTag()
              }
            }}
            className="flex-1"
          />
          <Button type="button" variant="outline" size="sm" onClick={addTag}>
            Add
          </Button>
        </div>
      </div>
    </div>
  )
}

// ConfigSummary shows a brief summary of the channel config
function ConfigSummary({ channel }: { channel: AlertChannel }) {
  const cfg = channel.config
  switch (channel.type) {
    case 'telegram':
      return <span>Chat ID: {(cfg.chat_id as string) || 'N/A'}</span>
    case 'slack': {
      const channel = (cfg.default_channel as string) || 'N/A'
      return <span>Channel: {channel}</span>
    }
    case 'slack_webhook':
      return <span>Webhook configured</span>
    case 'email': {
      const host = (cfg.smtp_host as string) || 'N/A'
      const port = String(cfg.smtp_port || '')
      const to = Array.isArray(cfg.to) ? cfg.to.join(', ') : 'N/A'
      return (
        <span>
          {host}:{port} &rarr; {to}
        </span>
      )
    }
    case 'discord':
      return <span>Webhook configured</span>
    case 'teams':
      return <span>Webhook configured</span>
    case 'pagerduty':
      return <span>Routing key configured</span>
    case 'opsgenie': {
      const region = (cfg.region as string) || 'us'
      return <span>Region: {region}</span>
    }
    case 'ntfy': {
      const url = (cfg.server_url as string) || 'ntfy.sh'
      return <span>{url} → {(cfg.topic as string) || 'N/A'}</span>
    }
    default:
      return <span>Configured</span>
  }
}
