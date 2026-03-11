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
}
