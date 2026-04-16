/**
 * API client factory for the Checker UI.
 *
 * Usage:
 *   // Standalone (cookie auth, same-origin):
 *   const api = createApiClient()
 *
 *   // Cloud (JWT auth, custom base URL):
 *   const api = createApiClient({
 *     baseUrl: '/api/v1',
 *     authMode: 'bearer',
 *     getAuthToken: () => localStorage.getItem('token'),
 *     onUnauthorized: () => router.push('/login'),
 *   })
 */

export interface ApiClientConfig {
  /** Base URL prefix for all API calls. Default: '' (same origin) */
  baseUrl?: string
  /** Authentication mode. Default: 'cookie' */
  authMode?: 'cookie' | 'bearer'
  /** Function to get the bearer token when authMode is 'bearer' */
  getAuthToken?: () => string | null
  /** Callback when a 401 is received */
  onUnauthorized?: () => void
  /** Optional error reporter (e.g. Sentry) */
  onError?: (error: Error, context: { url: string; status: number }) => void
}

export interface CheckDefinition {
  id: string
  uuid: string
  name: string
  project: string
  group_name: string
  type: string
  description: string
  enabled: boolean
  created_at: string
  updated_at: string
  duration: string
  url?: string
  timeout?: string
  host?: string
  port?: number
  answer?: string
  answer_present?: boolean
  code?: number[]
  headers?: Record<string, string>[]
  cookies?: Record<string, string>[]
  skip_check_ssl?: boolean
  ssl_expiration_period?: string
  stop_follow_redirects?: boolean
  auth?: { user?: string; password?: string }
  count?: number
  domain?: string
  record_type?: string
  expected?: string
  expect_banner?: string
  redis_password?: string
  redis_db?: number
  mongodb_uri?: string
  expiry_warning_days?: number
  starttls?: boolean
  username?: string
  password?: string
  validate_chain?: boolean
  use_tls?: boolean
  pgsql?: {
    username?: string; password?: string; dbname?: string; sslmode?: string
    query?: string; response?: string; difference?: string; table_name?: string
    lag?: string; server_list?: string[]; analytic_replicas?: string[]
  }
  mysql?: {
    username?: string; password?: string; dbname?: string
    query?: string; response?: string; difference?: string; table_name?: string
    lag?: string; server_list?: string[]
  }
  actor_type?: string
  alert_channels?: string[]
  target_regions?: string[]
  run_mode?: string
  maintenance_until?: string | null
  re_alert_interval?: string
}

export interface CheckImportResultItem {
  name: string; uuid: string; project: string
}

export interface CheckImportError {
  name: string; index: number; message: string
}

export interface CheckImportResult {
  created: CheckImportResultItem[]
  updated: CheckImportResultItem[]
  deleted: CheckImportResultItem[]
  errors: CheckImportError[]
  summary: { total: number; created: number; updated: number; deleted: number; errors: number }
}

export interface CheckImportValidation {
  valid: boolean
  checks: Array<Record<string, unknown>>
  errors: CheckImportError[]
  count: number
}

export interface AlertEvent {
  ID: number; CheckUUID: string; CheckName: string; Project: string
  GroupName: string; CheckType: string; Message: string; AlertType: string
  CreatedAt: string; ResolvedAt: string | null; IsResolved: boolean
}

export interface AlertsResponse {
  alerts: AlertEvent[]; total: number
}

export interface AlertSilence {
  ID: number; Scope: string; Target: string; Channel: string
  SilencedBy: string; SilencedAt: string; ExpiresAt: string | null
  Reason: string; Active: boolean
}

export interface SilencesResponse {
  silences: AlertSilence[]
}

export interface CreateSilenceRequest {
  scope: 'check' | 'project'; target: string; channel?: string
  duration: string; reason?: string
}

export interface AlertChannel {
  id: number; name: string; type: string
  config: Record<string, unknown>; created_at: string; updated_at: string
}

export interface AlertChannelInput {
  name: string; type: string; config: Record<string, unknown>
}

export interface CheckDefaults {
  retry_count: number; retry_interval: string; check_interval: string
  timeouts: Record<string, string>; re_alert_interval: string
  severity: string; alert_channels: string[]; escalation_policy: string
}

export interface RegionResult {
  region: string; is_healthy: boolean; message: string; created_at: string
}

export interface ProjectSettings {
  project: string
  enabled?: boolean | null
  duration?: string | null
  re_alert_interval?: string | null
  maintenance_until?: string | null
  maintenance_reason: string
  updated_at: string
}

export interface GroupSettings {
  project: string
  group_name: string
  enabled?: boolean | null
  duration?: string | null
  re_alert_interval?: string | null
  maintenance_until?: string | null
  maintenance_reason: string
  updated_at: string
}

export interface TenantRegionsResponse {
  regions: string[]
  quota: { current: number; limit: number }
}

export interface EdgeInstance {
  id: string
  tenant_id: string
  api_key_id: string
  region: string
  status: string
  version?: string
  last_heartbeat_at?: string
  connected_at?: string
  disconnected_at?: string
  remote_addr?: string
  metadata?: unknown
  created_at: string
  updated_at: string
}

export interface EdgeInstancesResponse {
  edge_instances: EdgeInstance[]
  quota: { current: number; limit: number }
}

export interface TestRemoteLocationResult {
  source: 'platform' | 'on-premises'
  region: string
  healthy: boolean
  message: string
  duration_ms: number
}

export type ApiClient = ReturnType<typeof createApiClient>

export function createApiClient(config: ApiClientConfig = {}) {
  const {
    baseUrl = '',
    authMode = 'cookie',
    getAuthToken,
    onUnauthorized = () => { window.location.href = '/login' },
    onError,
  } = config

  async function request<T>(url: string, options?: RequestInit): Promise<T> {
    const headers: Record<string, string> = {
      'Content-Type': 'application/json',
      ...(options?.headers as Record<string, string> || {}),
    }

    if (authMode === 'bearer' && getAuthToken) {
      const token = getAuthToken()
      if (token) headers['Authorization'] = `Bearer ${token}`
    }

    const res = await fetch(`${baseUrl}${url}`, {
      ...options,
      headers,
      credentials: authMode === 'cookie' ? 'include' : 'same-origin',
    })

    if (res.status === 401) {
      onUnauthorized()
      throw new Error('Unauthorized')
    }

    if (!res.ok) {
      const body = await res.text()
      const err = new Error(body || res.statusText)
      if (onError) {
        onError(err, { url, status: res.status })
      }
      throw err
    }
    return res.json()
  }

  return {
    getChecks: () => request<CheckDefinition[]>('/api/check-definitions'),
    getCheck: (uuid: string) => request<CheckDefinition>(`/api/check-definitions/${uuid}`),
    createCheck: (data: Partial<CheckDefinition>) =>
      request<CheckDefinition>('/api/check-definitions', {
        method: 'POST', body: JSON.stringify(data),
      }),
    updateCheck: (uuid: string, data: Partial<CheckDefinition>) =>
      request<CheckDefinition>(`/api/check-definitions/${uuid}`, {
        method: 'PUT', body: JSON.stringify(data),
      }),
    deleteCheck: (uuid: string) =>
      request<void>(`/api/check-definitions/${uuid}`, { method: 'DELETE' }),
    getCheckRegions: (uuid: string) =>
      request<RegionResult[]>(`/api/check-definitions/${uuid}/regions`),
    toggleCheck: (uuid: string, enabled?: boolean) =>
      request<{ enabled: boolean }>(
        `/api/check-definitions/${uuid}/toggle${enabled !== undefined ? `?enabled=${enabled}` : ''}`,
        { method: 'PATCH' }
      ),
    setMaintenance: (uuid: string, until: string) =>
      request<{ message: string; uuid: string; maintenance_until: string }>(
        `/api/check-definitions/${uuid}/maintenance`,
        { method: 'PUT', body: JSON.stringify({ until }) }
      ),
    clearMaintenance: (uuid: string) =>
      request<{ message: string; uuid: string }>(
        `/api/check-definitions/${uuid}/maintenance`,
        { method: 'DELETE' }
      ),
    getAlerts: (params?: { limit?: number; offset?: number; project?: string; status?: string; since?: string; until?: string }) => {
      const searchParams = new URLSearchParams()
      if (params?.limit) searchParams.set('limit', String(params.limit))
      if (params?.offset) searchParams.set('offset', String(params.offset))
      if (params?.project) searchParams.set('project', params.project)
      if (params?.status) searchParams.set('status', params.status)
      if (params?.since) searchParams.set('since', params.since)
      if (params?.until) searchParams.set('until', params.until)
      const query = searchParams.toString()
      return request<AlertsResponse>(`/api/alerts${query ? '?' + query : ''}`)
    },
    getSilences: () => request<SilencesResponse>('/api/silences'),
    createSilence: (data: CreateSilenceRequest) =>
      request<{ message: string; silence: AlertSilence }>('/api/silences', {
        method: 'POST', body: JSON.stringify(data),
      }),
    deleteSilence: (id: number) =>
      request<{ message: string }>(`/api/silences/${id}`, { method: 'DELETE' }),
    getAlertChannels: () => request<AlertChannel[]>('/api/alert-channels'),
    createAlertChannel: (data: AlertChannelInput) =>
      request<{ message: string; name: string }>('/api/alert-channels', {
        method: 'POST', body: JSON.stringify(data),
      }),
    updateAlertChannel: (name: string, data: AlertChannelInput) =>
      request<{ message: string; name: string }>(`/api/alert-channels/${encodeURIComponent(name)}`, {
        method: 'PUT', body: JSON.stringify(data),
      }),
    deleteAlertChannel: (name: string) =>
      request<{ message: string }>(`/api/alert-channels/${encodeURIComponent(name)}`, { method: 'DELETE' }),
    testAlertChannel: (name: string) =>
      request<{ message: string; success: boolean; tested_at: string }>(
        `/api/alert-channels/${encodeURIComponent(name)}/test`, { method: 'POST' }
      ),
    testCheck: (data: Partial<CheckDefinition>) =>
      request<{ success: boolean; duration_ms: number; message: string }>(
        '/api/check-definitions/test', { method: 'POST', body: JSON.stringify(data) }
      ),
    testCheckRemote: (data: Partial<CheckDefinition>) =>
      request<{ results: TestRemoteLocationResult[] }>(
        '/api/check-definitions/test-remote', { method: 'POST', body: JSON.stringify(data) }
      ),
    testCheckByUUID: (uuid: string) =>
      request<{ results: TestRemoteLocationResult[] }>(
        `/api/check-definitions/${uuid}/test-remote`, { method: 'POST' }
      ),
    getProjects: () => request<string[]>('/api/metadata/projects'),
    getCheckTypes: () => request<string[]>('/api/metadata/check-types'),
    getDefaultTimeouts: () => request<Record<string, string>>('/api/metadata/default-timeouts'),
    getCheckDefaults: () => request<CheckDefaults>('/api/settings/check-defaults'),
    getPlatformRegions: () => request<TenantRegionsResponse>('/api/platform-regions').catch(() => null),
    getEdgeInstances: () => request<EdgeInstancesResponse>('/api/edge-instances').catch(() => null),
    updateCheckDefaults: (data: CheckDefaults) =>
      request<CheckDefaults>('/api/settings/check-defaults', {
        method: 'PUT', body: JSON.stringify(data),
      }),
    bulkEnable: (uuids: string[]) =>
      request<{ success: boolean; count: number }>('/api/checks/bulk-enable', {
        method: 'POST', body: JSON.stringify({ uuids }),
      }),
    bulkDisable: (uuids: string[]) =>
      request<{ success: boolean; count: number }>('/api/checks/bulk-disable', {
        method: 'POST', body: JSON.stringify({ uuids }),
      }),
    bulkDelete: (uuids: string[]) =>
      request<{ success: boolean; count: number }>('/api/checks/bulk-delete', {
        method: 'POST', body: JSON.stringify({ uuids }),
      }),
    bulkAlertChannels: (uuids: string[], action: string, channels: string[]) =>
      request<{ success: boolean; count: number }>('/api/checks/bulk-alert-channels', {
        method: 'POST', body: JSON.stringify({ uuids, action, channels }),
      }),
    importChecks: (yamlContent: string) => {
      const headers: Record<string, string> = { 'Content-Type': 'application/x-yaml' }
      if (authMode === 'bearer' && getAuthToken) {
        const token = getAuthToken()
        if (token) headers['Authorization'] = `Bearer ${token}`
      }
      return fetch(`${baseUrl}/api/checks/import`, {
        method: 'POST', headers,
        credentials: authMode === 'cookie' ? 'include' : 'same-origin',
        body: yamlContent,
      }).then(async (res) => {
        if (res.status === 401) { onUnauthorized(); throw new Error('Unauthorized') }
        const data = await res.json()
        if (!res.ok && res.status !== 200) throw new Error(data.error || res.statusText)
        return data as CheckImportResult
      })
    },
    validateImport: (yamlContent: string) => {
      const headers: Record<string, string> = { 'Content-Type': 'application/x-yaml' }
      if (authMode === 'bearer' && getAuthToken) {
        const token = getAuthToken()
        if (token) headers['Authorization'] = `Bearer ${token}`
      }
      return fetch(`${baseUrl}/api/checks/import/validate`, {
        method: 'POST', headers,
        credentials: authMode === 'cookie' ? 'include' : 'same-origin',
        body: yamlContent,
      }).then(async (res) => {
        if (res.status === 401) { onUnauthorized(); throw new Error('Unauthorized') }
        const data = await res.json()
        if (!res.ok) throw new Error(data.error || res.statusText)
        return data as CheckImportValidation
      })
    },
    // Project & Group hierarchical settings
    getAllProjectSettings: () => request<ProjectSettings[]>('/api/project-settings'),
    getProjectSettings: (project: string) =>
      request<ProjectSettings>(`/api/project-settings/${encodeURIComponent(project)}`),
    updateProjectSettings: (project: string, data: { enabled?: boolean | null; duration?: string | null; re_alert_interval?: string | null }) =>
      request<{ message: string }>(`/api/project-settings/${encodeURIComponent(project)}`, {
        method: 'PUT', body: JSON.stringify(data),
      }),
    setProjectMaintenance: (project: string, duration: string, reason: string) =>
      request<{ message: string; maintenance_until: string; maintenance_reason: string }>(
        `/api/project-settings/${encodeURIComponent(project)}/maintenance`,
        { method: 'POST', body: JSON.stringify({ duration, reason }) }
      ),
    clearProjectMaintenance: (project: string) =>
      request<{ message: string }>(
        `/api/project-settings/${encodeURIComponent(project)}/maintenance`,
        { method: 'DELETE' }
      ),
    getAllGroupSettings: () => request<GroupSettings[]>('/api/group-settings'),
    getGroupSettings: (project: string, group: string) =>
      request<GroupSettings>(`/api/group-settings/${encodeURIComponent(project)}/${encodeURIComponent(group)}`),
    updateGroupSettings: (project: string, group: string, data: { enabled?: boolean | null; duration?: string | null; re_alert_interval?: string | null }) =>
      request<{ message: string }>(`/api/group-settings/${encodeURIComponent(project)}/${encodeURIComponent(group)}`, {
        method: 'PUT', body: JSON.stringify(data),
      }),
    setGroupMaintenance: (project: string, group: string, duration: string, reason: string) =>
      request<{ message: string; maintenance_until: string; maintenance_reason: string }>(
        `/api/group-settings/${encodeURIComponent(project)}/${encodeURIComponent(group)}/maintenance`,
        { method: 'POST', body: JSON.stringify({ duration, reason }) }
      ),
    clearGroupMaintenance: (project: string, group: string) =>
      request<{ message: string }>(
        `/api/group-settings/${encodeURIComponent(project)}/${encodeURIComponent(group)}/maintenance`,
        { method: 'DELETE' }
      ),
    exportChecks: (project?: string, environment?: string) => {
      const params = new URLSearchParams()
      if (project) params.set('project', project)
      if (environment) params.set('environment', environment)
      const query = params.toString()
      const headers: Record<string, string> = { Accept: 'application/x-yaml' }
      if (authMode === 'bearer' && getAuthToken) {
        const token = getAuthToken()
        if (token) headers['Authorization'] = `Bearer ${token}`
      }
      return fetch(`${baseUrl}/api/checks/export${query ? '?' + query : ''}`, {
        headers,
        credentials: authMode === 'cookie' ? 'include' : 'same-origin',
      }).then(async (res) => {
        if (res.status === 401) { onUnauthorized(); throw new Error('Unauthorized') }
        if (!res.ok) throw new Error(res.statusText)
        return res.text()
      })
    },
  }
}

/** Default singleton for standalone mode (cookie auth, same-origin) */
export const api = createApiClient()
