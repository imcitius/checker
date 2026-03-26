const BASE = ''

async function request<T>(url: string, options?: RequestInit): Promise<T> {
  const res = await fetch(`${BASE}${url}`, {
    ...options,
    headers: {
      'Content-Type': 'application/json',
      ...options?.headers,
    },
    credentials: 'include',
  })
  if (res.status === 401) {
    window.location.href = '/login'
    throw new Error('Unauthorized')
  }
  if (!res.ok) {
    const body = await res.text()
    throw new Error(body || res.statusText)
  }
  return res.json()
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
  // HTTP advanced fields
  answer?: string
  answer_present?: boolean
  code?: number[]
  headers?: Record<string, string>[]
  cookies?: Record<string, string>[]
  skip_check_ssl?: boolean
  ssl_expiration_period?: string
  stop_follow_redirects?: boolean
  auth?: {
    user?: string
    password?: string
  }
  // ICMP fields
  count?: number
  // DNS fields
  domain?: string
  record_type?: string
  expected?: string
  // SSH fields
  expect_banner?: string
  // Redis fields
  redis_password?: string
  redis_db?: number
  // MongoDB fields
  mongodb_uri?: string
  // Domain expiry fields
  expiry_warning_days?: number
  // Database config
  pgsql?: {
    username?: string
    password?: string
    dbname?: string
    sslmode?: string
    query?: string
    response?: string
    difference?: string
    table_name?: string
    lag?: string
    server_list?: string[]
    analytic_replicas?: string[]
  }
  mysql?: {
    username?: string
    password?: string
    dbname?: string
    query?: string
    response?: string
    difference?: string
    table_name?: string
    lag?: string
    server_list?: string[]
  }
  actor_type?: string
  alert_type?: string
  alert_destination?: string
  alert_channels?: string[]
  // Maintenance window
  maintenance_until?: string | null
}

export interface CheckImportResultItem {
  name: string
  uuid: string
  project: string
}

export interface CheckImportError {
  name: string
  index: number
  message: string
}

export interface CheckImportResult {
  created: CheckImportResultItem[]
  updated: CheckImportResultItem[]
  deleted: CheckImportResultItem[]
  errors: CheckImportError[]
  summary: {
    total: number
    created: number
    updated: number
    deleted: number
    errors: number
  }
}

export interface CheckImportValidation {
  valid: boolean
  checks: Array<Record<string, unknown>>
  errors: CheckImportError[]
  count: number
}

export interface AlertEvent {
  ID: number
  CheckUUID: string
  CheckName: string
  Project: string
  GroupName: string
  CheckType: string
  Message: string
  AlertType: string
  CreatedAt: string
  ResolvedAt: string | null
  IsResolved: boolean
}

export interface AlertsResponse {
  alerts: AlertEvent[]
  total: number
}

export interface AlertSilence {
  ID: number
  Scope: string
  Target: string
  Channel: string
  SilencedBy: string
  SilencedAt: string
  ExpiresAt: string | null
  Reason: string
  Active: boolean
}

export interface SilencesResponse {
  silences: AlertSilence[]
}

export interface CreateSilenceRequest {
  scope: 'check' | 'project'
  target: string
  channel?: string
  duration: string
  reason?: string
}

export interface AlertChannel {
  id: number
  name: string
  type: string
  config: Record<string, unknown>
  created_at: string
  updated_at: string
}

export interface AlertChannelInput {
  name: string
  type: string
  config: Record<string, unknown>
}

export const api = {
  getChecks: () => request<CheckDefinition[]>('/api/check-definitions'),
  getCheck: (uuid: string) => request<CheckDefinition>(`/api/check-definitions/${uuid}`),
  createCheck: (data: Partial<CheckDefinition>) =>
    request<CheckDefinition>('/api/check-definitions', {
      method: 'POST',
      body: JSON.stringify(data),
    }),
  updateCheck: (uuid: string, data: Partial<CheckDefinition>) =>
    request<CheckDefinition>(`/api/check-definitions/${uuid}`, {
      method: 'PUT',
      body: JSON.stringify(data),
    }),
  deleteCheck: (uuid: string) =>
    request<void>(`/api/check-definitions/${uuid}`, { method: 'DELETE' }),
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
  // Alerts & Silences
  getAlerts: (params?: { limit?: number; offset?: number; project?: string; status?: string }) => {
    const searchParams = new URLSearchParams()
    if (params?.limit) searchParams.set('limit', String(params.limit))
    if (params?.offset) searchParams.set('offset', String(params.offset))
    if (params?.project) searchParams.set('project', params.project)
    if (params?.status) searchParams.set('status', params.status)
    const query = searchParams.toString()
    return request<AlertsResponse>(`/api/alerts${query ? '?' + query : ''}`)
  },
  getSilences: () => request<SilencesResponse>('/api/silences'),
  createSilence: (data: CreateSilenceRequest) =>
    request<{ message: string; silence: AlertSilence }>('/api/silences', {
      method: 'POST',
      body: JSON.stringify(data),
    }),
  deleteSilence: (id: number) =>
    request<{ message: string }>(`/api/silences/${id}`, { method: 'DELETE' }),

  // Alert channels
  getAlertChannels: () => request<AlertChannel[]>('/api/alert-channels'),
  createAlertChannel: (data: AlertChannelInput) =>
    request<{ message: string; name: string }>('/api/alert-channels', {
      method: 'POST',
      body: JSON.stringify(data),
    }),
  updateAlertChannel: (name: string, data: AlertChannelInput) =>
    request<{ message: string; name: string }>(`/api/alert-channels/${encodeURIComponent(name)}`, {
      method: 'PUT',
      body: JSON.stringify(data),
    }),
  deleteAlertChannel: (name: string) =>
    request<{ message: string }>(`/api/alert-channels/${encodeURIComponent(name)}`, {
      method: 'DELETE',
    }),
  testAlertChannel: (name: string) =>
    request<{ message: string; success: boolean; tested_at: string }>(
      `/api/alert-channels/${encodeURIComponent(name)}/test`,
      { method: 'POST' }
    ),

  testCheck: (data: Partial<CheckDefinition>) =>
    request<{ success: boolean; duration_ms: number; message: string }>(
      '/api/check-definitions/test',
      { method: 'POST', body: JSON.stringify(data) }
    ),

  getProjects: () => request<string[]>('/api/metadata/projects'),
  getCheckTypes: () => request<string[]>('/api/metadata/check-types'),
  getDefaultTimeouts: () => request<Record<string, string>>('/api/metadata/default-timeouts'),

  // Bulk actions
  bulkEnable: (uuids: string[]) =>
    request<{ success: boolean; count: number }>('/api/checks/bulk-enable', {
      method: 'POST',
      body: JSON.stringify({ uuids }),
    }),
  bulkDisable: (uuids: string[]) =>
    request<{ success: boolean; count: number }>('/api/checks/bulk-disable', {
      method: 'POST',
      body: JSON.stringify({ uuids }),
    }),
  bulkDelete: (uuids: string[]) =>
    request<{ success: boolean; count: number }>('/api/checks/bulk-delete', {
      method: 'POST',
      body: JSON.stringify({ uuids }),
    }),

  // Bulk import/export
  importChecks: (yamlContent: string) => {
    return fetch(`${BASE}/api/checks/import`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/x-yaml' },
      credentials: 'include',
      body: yamlContent,
    }).then(async (res) => {
      if (res.status === 401) {
        window.location.href = '/login'
        throw new Error('Unauthorized')
      }
      const data = await res.json()
      if (!res.ok && res.status !== 200) {
        throw new Error(data.error || res.statusText)
      }
      return data as CheckImportResult
    })
  },
  validateImport: (yamlContent: string) => {
    return fetch(`${BASE}/api/checks/import/validate`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/x-yaml' },
      credentials: 'include',
      body: yamlContent,
    }).then(async (res) => {
      if (res.status === 401) {
        window.location.href = '/login'
        throw new Error('Unauthorized')
      }
      const data = await res.json()
      if (!res.ok) {
        throw new Error(data.error || res.statusText)
      }
      return data as CheckImportValidation
    })
  },
  exportChecks: (project?: string, environment?: string) => {
    const params = new URLSearchParams()
    if (project) params.set('project', project)
    if (environment) params.set('environment', environment)
    const query = params.toString()
    return fetch(`${BASE}/api/checks/export${query ? '?' + query : ''}`, {
      headers: { Accept: 'application/x-yaml' },
      credentials: 'include',
    }).then(async (res) => {
      if (res.status === 401) {
        window.location.href = '/login'
        throw new Error('Unauthorized')
      }
      if (!res.ok) {
        throw new Error(res.statusText)
      }
      return res.text()
    })
  },
}
