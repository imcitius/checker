import { useState, useEffect } from 'react'
import { Settings, Wrench, X, Power, PowerOff, Timer, Bell } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Switch } from '@/components/ui/switch'
import { Badge } from '@/components/ui/badge'
import { api, type ProjectSettings, type GroupSettings } from '@/lib/api'
import { toast } from 'sonner'
import { cn } from '@/lib/utils'

const MAINTENANCE_DURATIONS = [
  { label: '1h', value: '1h' },
  { label: '4h', value: '4h' },
  { label: '8h', value: '8h' },
  { label: '24h', value: '24h' },
  { label: 'Custom', value: 'custom' },
]

interface HierarchySettingsPanelProps {
  level: 'project' | 'group'
  project: string
  group?: string
  settings: ProjectSettings | GroupSettings | null
  onSettingsChanged: () => void
}

function isInMaintenance(settings: ProjectSettings | GroupSettings | null): boolean {
  if (!settings?.maintenance_until) return false
  return new Date(settings.maintenance_until) > new Date()
}

function formatTimeRemaining(until: string): string {
  const diff = new Date(until).getTime() - Date.now()
  if (diff <= 0) return 'expired'
  const hours = Math.floor(diff / 3600000)
  const minutes = Math.floor((diff % 3600000) / 60000)
  if (hours > 24) return `${Math.floor(hours / 24)}d ${hours % 24}h`
  if (hours > 0) return `${hours}h ${minutes}m`
  return `${minutes}m`
}

export function HierarchySettingsPanel({
  level,
  project,
  group,
  settings,
  onSettingsChanged,
}: HierarchySettingsPanelProps) {
  const [open, setOpen] = useState(false)
  const [saving, setSaving] = useState(false)
  const [enabled, setEnabled] = useState<boolean | null>(settings?.enabled ?? null)
  const [duration, setDuration] = useState(settings?.duration ?? '')
  const [reAlertInterval, setReAlertInterval] = useState(settings?.re_alert_interval ?? '')
  const [maintenanceDuration, setMaintenanceDuration] = useState('1h')
  const [customDuration, setCustomDuration] = useState('2h')
  const [maintenanceReason, setMaintenanceReason] = useState('')

  const inMaintenance = isInMaintenance(settings)

  // Sync state when settings change
  useEffect(() => {
    setEnabled(settings?.enabled ?? null)
    setDuration(settings?.duration ?? '')
    setReAlertInterval(settings?.re_alert_interval ?? '')
  }, [settings])

  const handleSaveSettings = async () => {
    setSaving(true)
    try {
      const data = {
        enabled: enabled,
        duration: duration || null,
        re_alert_interval: reAlertInterval || null,
      }
      if (level === 'project') {
        await api.updateProjectSettings(project, data)
      } else {
        await api.updateGroupSettings(project, group!, data)
      }
      toast.success(`${level === 'project' ? 'Project' : 'Group'} settings updated`)
      onSettingsChanged()
    } catch (err) {
      toast.error(`Failed to save settings: ${err}`)
    } finally {
      setSaving(false)
    }
  }

  const handleSetMaintenance = async () => {
    setSaving(true)
    try {
      const dur = maintenanceDuration === 'custom' ? customDuration : maintenanceDuration
      if (level === 'project') {
        await api.setProjectMaintenance(project, dur, maintenanceReason)
      } else {
        await api.setGroupMaintenance(project, group!, dur, maintenanceReason)
      }
      toast.success(`Maintenance mode enabled`)
      setMaintenanceReason('')
      onSettingsChanged()
    } catch (err) {
      toast.error(`Failed to set maintenance: ${err}`)
    } finally {
      setSaving(false)
    }
  }

  const handleClearMaintenance = async () => {
    setSaving(true)
    try {
      if (level === 'project') {
        await api.clearProjectMaintenance(project)
      } else {
        await api.clearGroupMaintenance(project, group!)
      }
      toast.success('Maintenance mode cleared')
      onSettingsChanged()
    } catch (err) {
      toast.error(`Failed to clear maintenance: ${err}`)
    } finally {
      setSaving(false)
    }
  }

  const hasOverrides = settings && (
    settings.enabled !== null && settings.enabled !== undefined ||
    (settings.duration !== null && settings.duration !== undefined && settings.duration !== '') ||
    (settings.re_alert_interval !== null && settings.re_alert_interval !== undefined && settings.re_alert_interval !== '')
  )

  return (
    <>
      {/* Inline indicators */}
      {inMaintenance && (
        <Badge variant="outline" className="text-[10px] bg-amber-500/10 text-amber-600 border-amber-300 gap-1 shrink-0">
          <Wrench className="h-3 w-3" />
          {settings?.maintenance_reason || 'Maintenance'}
          <span className="text-amber-500/70">ends in {formatTimeRemaining(settings!.maintenance_until!)}</span>
        </Badge>
      )}
      {settings?.enabled === false && !inMaintenance && (
        <Badge variant="outline" className="text-[10px] bg-red-500/10 text-red-600 border-red-300 gap-1 shrink-0">
          <PowerOff className="h-3 w-3" />
          Disabled
        </Badge>
      )}
      {hasOverrides && !inMaintenance && settings?.enabled !== false && (
        <Badge variant="outline" className="text-[10px] bg-blue-500/10 text-blue-600 border-blue-300 gap-0.5 shrink-0">
          <Settings className="h-2.5 w-2.5" />
          Overrides
        </Badge>
      )}

      {/* Gear icon toggle */}
      <button
        onClick={(e) => {
          e.stopPropagation()
          setOpen(!open)
        }}
        className={cn(
          'p-0.5 rounded hover:bg-muted/60 transition-colors shrink-0',
          open && 'bg-muted text-foreground',
          !open && 'text-muted-foreground/50 hover:text-muted-foreground'
        )}
        title={`${level === 'project' ? 'Project' : 'Group'} settings`}
      >
        <Settings className="h-3.5 w-3.5" />
      </button>

      {/* Settings panel (inline, below the header) */}
      {open && (
        <div
          className="absolute left-0 right-0 top-full z-10 bg-card border-b border-border shadow-sm"
          onClick={(e) => e.stopPropagation()}
        >
          <div className="px-4 py-3 space-y-3 max-w-2xl">
            <div className="flex items-center justify-between">
              <span className="text-xs font-semibold text-foreground">
                {level === 'project' ? 'Project' : 'Group'} Settings
              </span>
              <button onClick={() => setOpen(false)} className="text-muted-foreground hover:text-foreground">
                <X className="h-3.5 w-3.5" />
              </button>
            </div>

            {/* Enable/Disable */}
            <div className="flex items-center gap-3">
              <div className="flex items-center gap-2 min-w-[140px]">
                {enabled === false ? (
                  <PowerOff className="h-3.5 w-3.5 text-red-500" />
                ) : (
                  <Power className="h-3.5 w-3.5 text-green-500" />
                )}
                <span className="text-xs">Enable checks</span>
              </div>
              <div className="flex items-center gap-2">
                <Switch
                  checked={enabled !== false}
                  onCheckedChange={(checked) => setEnabled(checked ? null : false)}
                />
                <span className="text-[10px] text-muted-foreground">
                  {enabled === null ? '(inherit)' : enabled === false ? 'Disabled' : 'Enabled'}
                </span>
              </div>
            </div>

            {/* Duration override */}
            <div className="flex items-center gap-3">
              <div className="flex items-center gap-2 min-w-[140px]">
                <Timer className="h-3.5 w-3.5 text-muted-foreground" />
                <span className="text-xs">Check interval</span>
              </div>
              <Input
                value={duration ?? ''}
                onChange={(e) => setDuration(e.target.value || '')}
                placeholder="inherit"
                className="h-7 w-24 text-xs"
              />
              <span className="text-[10px] text-muted-foreground">e.g. 1m, 5m, 1h</span>
            </div>

            {/* Re-alert interval override */}
            <div className="flex items-center gap-3">
              <div className="flex items-center gap-2 min-w-[140px]">
                <Bell className="h-3.5 w-3.5 text-muted-foreground" />
                <span className="text-xs">Re-alert interval</span>
              </div>
              <Input
                value={reAlertInterval ?? ''}
                onChange={(e) => setReAlertInterval(e.target.value || '')}
                placeholder="inherit"
                className="h-7 w-24 text-xs"
              />
              <span className="text-[10px] text-muted-foreground">e.g. 30m, 1h</span>
            </div>

            <div className="flex gap-2">
              <Button size="sm" variant="default" onClick={handleSaveSettings} disabled={saving} className="h-7 text-xs">
                Save Settings
              </Button>
              {(enabled !== null || duration || reAlertInterval) && (
                <Button
                  size="sm"
                  variant="ghost"
                  onClick={() => {
                    setEnabled(null)
                    setDuration('')
                    setReAlertInterval('')
                  }}
                  className="h-7 text-xs text-muted-foreground"
                >
                  Reset to inherit
                </Button>
              )}
            </div>

            {/* Maintenance mode */}
            <div className="border-t pt-3 mt-3">
              <div className="flex items-center gap-2 mb-2">
                <Wrench className="h-3.5 w-3.5 text-amber-500" />
                <span className="text-xs font-semibold">Maintenance Mode</span>
              </div>

              {inMaintenance ? (
                <div className="space-y-2">
                  <div className="flex items-center gap-2 text-xs">
                    <Badge variant="outline" className="bg-amber-500/10 text-amber-600 border-amber-300 text-[10px]">
                      Active
                    </Badge>
                    {settings?.maintenance_reason && (
                      <span className="text-muted-foreground">{settings.maintenance_reason}</span>
                    )}
                    <span className="text-muted-foreground">
                      — ends in {formatTimeRemaining(settings!.maintenance_until!)}
                    </span>
                  </div>
                  <Button
                    size="sm"
                    variant="outline"
                    onClick={handleClearMaintenance}
                    disabled={saving}
                    className="h-7 text-xs"
                  >
                    End Maintenance
                  </Button>
                </div>
              ) : (
                <div className="space-y-2">
                  <div className="flex items-center gap-2">
                    <div className="flex gap-1">
                      {MAINTENANCE_DURATIONS.map((d) => (
                        <button
                          key={d.value}
                          onClick={() => setMaintenanceDuration(d.value)}
                          className={cn(
                            'px-2 py-0.5 rounded text-[11px] border transition-colors',
                            maintenanceDuration === d.value
                              ? 'bg-amber-500/10 border-amber-300 text-amber-600'
                              : 'border-border text-muted-foreground hover:border-amber-300'
                          )}
                        >
                          {d.label}
                        </button>
                      ))}
                    </div>
                    {maintenanceDuration === 'custom' && (
                      <Input
                        value={customDuration}
                        onChange={(e) => setCustomDuration(e.target.value)}
                        placeholder="e.g. 2h30m"
                        className="h-7 w-24 text-xs"
                      />
                    )}
                  </div>
                  <Input
                    value={maintenanceReason}
                    onChange={(e) => setMaintenanceReason(e.target.value)}
                    placeholder="Reason (optional): e.g. Scheduled database migration"
                    className="h-7 text-xs"
                  />
                  <Button
                    size="sm"
                    variant="outline"
                    onClick={handleSetMaintenance}
                    disabled={saving}
                    className="h-7 text-xs border-amber-300 text-amber-600 hover:bg-amber-500/10"
                  >
                    <Wrench className="h-3 w-3 mr-1" />
                    Start Maintenance
                  </Button>
                </div>
              )}
            </div>
          </div>
        </div>
      )}
    </>
  )
}

/** Compact inline badges for project/group headers (no settings panel) */
export function HierarchyStatusBadges({
  settings,
}: {
  settings: ProjectSettings | GroupSettings | null
}) {
  if (!settings) return null
  const inMaintenance = isInMaintenance(settings)

  return (
    <>
      {inMaintenance && (
        <Badge variant="outline" className="text-[10px] bg-amber-500/10 text-amber-600 border-amber-300 gap-1 shrink-0">
          <Wrench className="h-3 w-3" />
          {settings.maintenance_reason || 'Maintenance'}
          <span className="text-amber-500/70">ends in {formatTimeRemaining(settings.maintenance_until!)}</span>
        </Badge>
      )}
      {settings.enabled === false && !inMaintenance && (
        <Badge variant="outline" className="text-[10px] bg-red-500/10 text-red-600 border-red-300 gap-1 shrink-0">
          <PowerOff className="h-3 w-3" />
          Disabled
        </Badge>
      )}
    </>
  )
}
