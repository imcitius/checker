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
    window.location.href = '/auth/login'
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
  pgsql?: {
    username?: string
    dbname?: string
    query?: string
    server_list?: string[]
  }
  mysql?: {
    username?: string
    dbname?: string
    query?: string
    server_list?: string[]
  }
  actor_type?: string
  alert_type?: string
  alert_destination?: string
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
  toggleCheck: (uuid: string) =>
    request<{ enabled: boolean }>(`/api/check-definitions/${uuid}/toggle`, {
      method: 'PATCH',
    }),
  getProjects: () => request<string[]>('/api/metadata/projects'),
  getCheckTypes: () => request<string[]>('/api/metadata/check-types'),
  getDefaultTimeouts: () => request<Record<string, string>>('/api/metadata/default-timeouts'),

  // Bulk import/export
  importChecks: (yamlContent: string) => {
    return fetch(`${BASE}/api/checks/import`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/x-yaml' },
      credentials: 'include',
      body: yamlContent,
    }).then(async (res) => {
      if (res.status === 401) {
        window.location.href = '/auth/login'
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
        window.location.href = '/auth/login'
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
        window.location.href = '/auth/login'
        throw new Error('Unauthorized')
      }
      if (!res.ok) {
        throw new Error(res.statusText)
      }
      return res.text()
    })
  },
}
