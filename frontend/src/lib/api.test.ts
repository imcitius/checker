import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { api } from './api'

// Mock Sentry before importing
vi.mock('@sentry/react', () => ({
  captureException: vi.fn(),
}))

const mockFetch = vi.fn()

beforeEach(() => {
  vi.stubGlobal('fetch', mockFetch)
  mockFetch.mockReset()
  // Prevent actual navigation
  Object.defineProperty(window, 'location', {
    value: { href: '', protocol: 'https:', host: 'localhost' },
    writable: true,
  })
})

afterEach(() => {
  vi.unstubAllGlobals()
})

function jsonResponse(data: unknown, status = 200) {
  return Promise.resolve({
    ok: status >= 200 && status < 300,
    status,
    statusText: 'OK',
    json: () => Promise.resolve(data),
    text: () => Promise.resolve(JSON.stringify(data)),
  })
}

function textResponse(data: string, status = 200) {
  return Promise.resolve({
    ok: status >= 200 && status < 300,
    status,
    statusText: 'OK',
    text: () => Promise.resolve(data),
    json: () => Promise.resolve(data),
  })
}

function errorResponse(status: number, body = '') {
  return Promise.resolve({
    ok: false,
    status,
    statusText: 'Error',
    json: () => Promise.resolve({ error: body }),
    text: () => Promise.resolve(body),
  })
}

// ── getChecks ──────────────────────────────────────────────────────────

describe('api.getChecks', () => {
  it('fetches check definitions', async () => {
    const data = [{ id: '1', name: 'test' }]
    mockFetch.mockReturnValueOnce(jsonResponse(data))

    const result = await api.getChecks()
    expect(result).toEqual(data)
    expect(mockFetch).toHaveBeenCalledWith(
      '/api/check-definitions',
      expect.objectContaining({
        headers: { 'Content-Type': 'application/json' },
        credentials: 'include',
      })
    )
  })
})

// ── getCheck ───────────────────────────────────────────────────────────

describe('api.getCheck', () => {
  it('fetches a single check by uuid', async () => {
    const data = { id: '1', uuid: 'abc', name: 'test' }
    mockFetch.mockReturnValueOnce(jsonResponse(data))

    const result = await api.getCheck('abc')
    expect(result).toEqual(data)
    expect(mockFetch).toHaveBeenCalledWith(
      '/api/check-definitions/abc',
      expect.objectContaining({ credentials: 'include' })
    )
  })
})

// ── createCheck ────────────────────────────────────────────────────────

describe('api.createCheck', () => {
  it('sends POST with check data', async () => {
    const input = { name: 'new-check', type: 'http' }
    mockFetch.mockReturnValueOnce(jsonResponse({ ...input, uuid: 'xyz' }))

    await api.createCheck(input)
    expect(mockFetch).toHaveBeenCalledWith(
      '/api/check-definitions',
      expect.objectContaining({
        method: 'POST',
        body: JSON.stringify(input),
      })
    )
  })
})

// ── updateCheck ────────────────────────────────────────────────────────

describe('api.updateCheck', () => {
  it('sends PUT with check data', async () => {
    const input = { name: 'updated' }
    mockFetch.mockReturnValueOnce(jsonResponse({ uuid: 'abc', ...input }))

    await api.updateCheck('abc', input)
    expect(mockFetch).toHaveBeenCalledWith(
      '/api/check-definitions/abc',
      expect.objectContaining({
        method: 'PUT',
        body: JSON.stringify(input),
      })
    )
  })
})

// ── deleteCheck ────────────────────────────────────────────────────────

describe('api.deleteCheck', () => {
  it('sends DELETE request', async () => {
    mockFetch.mockReturnValueOnce(jsonResponse(undefined))

    await api.deleteCheck('abc')
    expect(mockFetch).toHaveBeenCalledWith(
      '/api/check-definitions/abc',
      expect.objectContaining({ method: 'DELETE' })
    )
  })
})

// ── toggleCheck ────────────────────────────────────────────────────────

describe('api.toggleCheck', () => {
  it('sends PATCH without enabled param', async () => {
    mockFetch.mockReturnValueOnce(jsonResponse({ enabled: true }))

    await api.toggleCheck('abc')
    expect(mockFetch).toHaveBeenCalledWith(
      '/api/check-definitions/abc/toggle',
      expect.objectContaining({ method: 'PATCH' })
    )
  })

  it('sends PATCH with enabled=true', async () => {
    mockFetch.mockReturnValueOnce(jsonResponse({ enabled: true }))

    await api.toggleCheck('abc', true)
    expect(mockFetch).toHaveBeenCalledWith(
      '/api/check-definitions/abc/toggle?enabled=true',
      expect.objectContaining({ method: 'PATCH' })
    )
  })

  it('sends PATCH with enabled=false', async () => {
    mockFetch.mockReturnValueOnce(jsonResponse({ enabled: false }))

    await api.toggleCheck('abc', false)
    expect(mockFetch).toHaveBeenCalledWith(
      '/api/check-definitions/abc/toggle?enabled=false',
      expect.objectContaining({ method: 'PATCH' })
    )
  })
})

// ── setMaintenance / clearMaintenance ──────────────────────────────────

describe('api.setMaintenance', () => {
  it('sends PUT with until', async () => {
    mockFetch.mockReturnValueOnce(jsonResponse({ message: 'ok' }))

    await api.setMaintenance('abc', '2026-12-31T00:00:00Z')
    expect(mockFetch).toHaveBeenCalledWith(
      '/api/check-definitions/abc/maintenance',
      expect.objectContaining({
        method: 'PUT',
        body: JSON.stringify({ until: '2026-12-31T00:00:00Z' }),
      })
    )
  })
})

describe('api.clearMaintenance', () => {
  it('sends DELETE', async () => {
    mockFetch.mockReturnValueOnce(jsonResponse({ message: 'ok' }))

    await api.clearMaintenance('abc')
    expect(mockFetch).toHaveBeenCalledWith(
      '/api/check-definitions/abc/maintenance',
      expect.objectContaining({ method: 'DELETE' })
    )
  })
})

// ── getAlerts ──────────────────────────────────────────────────────────

describe('api.getAlerts', () => {
  it('fetches alerts without params', async () => {
    mockFetch.mockReturnValueOnce(jsonResponse({ alerts: [], total: 0 }))

    await api.getAlerts()
    expect(mockFetch).toHaveBeenCalledWith(
      '/api/alerts',
      expect.objectContaining({ credentials: 'include' })
    )
  })

  it('builds query params', async () => {
    mockFetch.mockReturnValueOnce(jsonResponse({ alerts: [], total: 0 }))

    await api.getAlerts({ limit: 10, offset: 20, project: 'infra', status: 'active' })
    const url = mockFetch.mock.calls[0][0] as string
    expect(url).toContain('limit=10')
    expect(url).toContain('offset=20')
    expect(url).toContain('project=infra')
    expect(url).toContain('status=active')
  })
})

// ── Silences ──────────────────────────────────────────────────────────

describe('api.getSilences', () => {
  it('fetches silences', async () => {
    mockFetch.mockReturnValueOnce(jsonResponse({ silences: [] }))

    await api.getSilences()
    expect(mockFetch).toHaveBeenCalledWith(
      '/api/silences',
      expect.objectContaining({ credentials: 'include' })
    )
  })
})

describe('api.createSilence', () => {
  it('sends POST', async () => {
    const input = { scope: 'check' as const, target: 'abc', duration: '1h' }
    mockFetch.mockReturnValueOnce(jsonResponse({ message: 'ok' }))

    await api.createSilence(input)
    expect(mockFetch).toHaveBeenCalledWith(
      '/api/silences',
      expect.objectContaining({
        method: 'POST',
        body: JSON.stringify(input),
      })
    )
  })
})

describe('api.deleteSilence', () => {
  it('sends DELETE with numeric id', async () => {
    mockFetch.mockReturnValueOnce(jsonResponse({ message: 'ok' }))

    await api.deleteSilence(42)
    expect(mockFetch).toHaveBeenCalledWith(
      '/api/silences/42',
      expect.objectContaining({ method: 'DELETE' })
    )
  })
})

// ── Alert channels ─────────────────────────────────────────────────────

describe('api.getAlertChannels', () => {
  it('fetches alert channels', async () => {
    mockFetch.mockReturnValueOnce(jsonResponse([]))

    await api.getAlertChannels()
    expect(mockFetch).toHaveBeenCalledWith(
      '/api/alert-channels',
      expect.objectContaining({ credentials: 'include' })
    )
  })
})

describe('api.createAlertChannel', () => {
  it('sends POST', async () => {
    const input = { name: 'slack-main', type: 'slack', config: { webhook: 'https://...' } }
    mockFetch.mockReturnValueOnce(jsonResponse({ message: 'ok', name: 'slack-main' }))

    await api.createAlertChannel(input)
    expect(mockFetch).toHaveBeenCalledWith(
      '/api/alert-channels',
      expect.objectContaining({
        method: 'POST',
        body: JSON.stringify(input),
      })
    )
  })
})

describe('api.updateAlertChannel', () => {
  it('encodes channel name in URL', async () => {
    const input = { name: 'slack main', type: 'slack', config: {} }
    mockFetch.mockReturnValueOnce(jsonResponse({ message: 'ok' }))

    await api.updateAlertChannel('slack main', input)
    expect(mockFetch).toHaveBeenCalledWith(
      '/api/alert-channels/slack%20main',
      expect.objectContaining({ method: 'PUT' })
    )
  })
})

describe('api.deleteAlertChannel', () => {
  it('sends DELETE with encoded name', async () => {
    mockFetch.mockReturnValueOnce(jsonResponse({ message: 'ok' }))

    await api.deleteAlertChannel('my-channel')
    expect(mockFetch).toHaveBeenCalledWith(
      '/api/alert-channels/my-channel',
      expect.objectContaining({ method: 'DELETE' })
    )
  })
})

describe('api.testAlertChannel', () => {
  it('sends POST to test endpoint', async () => {
    mockFetch.mockReturnValueOnce(jsonResponse({ message: 'ok', success: true, tested_at: '' }))

    await api.testAlertChannel('my-channel')
    expect(mockFetch).toHaveBeenCalledWith(
      '/api/alert-channels/my-channel/test',
      expect.objectContaining({ method: 'POST' })
    )
  })
})

// ── testCheck ──────────────────────────────────────────────────────────

describe('api.testCheck', () => {
  it('sends POST to test endpoint', async () => {
    const input = { type: 'http', url: 'https://example.com' }
    mockFetch.mockReturnValueOnce(jsonResponse({ success: true, duration_ms: 50, message: 'ok' }))

    await api.testCheck(input)
    expect(mockFetch).toHaveBeenCalledWith(
      '/api/check-definitions/test',
      expect.objectContaining({
        method: 'POST',
        body: JSON.stringify(input),
      })
    )
  })
})

// ── Metadata ──────────────────────────────────────────────────────────

describe('api.getProjects', () => {
  it('fetches projects', async () => {
    mockFetch.mockReturnValueOnce(jsonResponse(['proj-a', 'proj-b']))

    const result = await api.getProjects()
    expect(result).toEqual(['proj-a', 'proj-b'])
  })
})

describe('api.getCheckTypes', () => {
  it('fetches check types', async () => {
    mockFetch.mockReturnValueOnce(jsonResponse(['http', 'tcp']))

    const result = await api.getCheckTypes()
    expect(result).toEqual(['http', 'tcp'])
  })
})

describe('api.getDefaultTimeouts', () => {
  it('fetches default timeouts', async () => {
    mockFetch.mockReturnValueOnce(jsonResponse({ http: '10s' }))

    const result = await api.getDefaultTimeouts()
    expect(result).toEqual({ http: '10s' })
  })
})

// ── Settings ──────────────────────────────────────────────────────────

describe('api.getCheckDefaults', () => {
  it('fetches check defaults', async () => {
    mockFetch.mockReturnValueOnce(jsonResponse({ retry_count: 3 }))

    const result = await api.getCheckDefaults()
    expect(result).toEqual({ retry_count: 3 })
  })
})

describe('api.updateCheckDefaults', () => {
  it('sends PUT with defaults', async () => {
    const input = {
      retry_count: 5,
      retry_interval: '30s',
      check_interval: '60s',
      timeouts: {},
      re_alert_interval: '5m',
      severity: 'critical',
      alert_channels: [],
      escalation_policy: '',
    }
    mockFetch.mockReturnValueOnce(jsonResponse(input))

    await api.updateCheckDefaults(input)
    expect(mockFetch).toHaveBeenCalledWith(
      '/api/settings/check-defaults',
      expect.objectContaining({
        method: 'PUT',
        body: JSON.stringify(input),
      })
    )
  })
})

// ── Bulk actions ──────────────────────────────────────────────────────

describe('api.bulkEnable', () => {
  it('sends POST with uuids', async () => {
    mockFetch.mockReturnValueOnce(jsonResponse({ success: true, count: 2 }))

    await api.bulkEnable(['a', 'b'])
    expect(mockFetch).toHaveBeenCalledWith(
      '/api/checks/bulk-enable',
      expect.objectContaining({
        method: 'POST',
        body: JSON.stringify({ uuids: ['a', 'b'] }),
      })
    )
  })
})

describe('api.bulkDisable', () => {
  it('sends POST with uuids', async () => {
    mockFetch.mockReturnValueOnce(jsonResponse({ success: true, count: 2 }))

    await api.bulkDisable(['a', 'b'])
    expect(mockFetch).toHaveBeenCalledWith(
      '/api/checks/bulk-disable',
      expect.objectContaining({
        method: 'POST',
        body: JSON.stringify({ uuids: ['a', 'b'] }),
      })
    )
  })
})

describe('api.bulkDelete', () => {
  it('sends POST with uuids', async () => {
    mockFetch.mockReturnValueOnce(jsonResponse({ success: true, count: 1 }))

    await api.bulkDelete(['x'])
    expect(mockFetch).toHaveBeenCalledWith(
      '/api/checks/bulk-delete',
      expect.objectContaining({
        method: 'POST',
        body: JSON.stringify({ uuids: ['x'] }),
      })
    )
  })
})

// ── Import / Export ───────────────────────────────────────────────────

describe('api.importChecks', () => {
  it('sends YAML content with correct content-type', async () => {
    const yaml = 'checks:\n  - name: test'
    mockFetch.mockReturnValueOnce(jsonResponse({ created: [], updated: [], deleted: [], errors: [], summary: {} }))

    await api.importChecks(yaml)
    expect(mockFetch).toHaveBeenCalledWith(
      '/api/checks/import',
      expect.objectContaining({
        method: 'POST',
        headers: { 'Content-Type': 'application/x-yaml' },
        body: yaml,
        credentials: 'include',
      })
    )
  })

  it('redirects on 401', async () => {
    mockFetch.mockReturnValueOnce(
      Promise.resolve({ ok: false, status: 401, statusText: 'Unauthorized', json: () => Promise.resolve({}) })
    )

    await expect(api.importChecks('test')).rejects.toThrow('Unauthorized')
    expect(window.location.href).toBe('/login')
  })
})

describe('api.validateImport', () => {
  it('sends YAML for validation', async () => {
    const yaml = 'checks:\n  - name: test'
    mockFetch.mockReturnValueOnce(jsonResponse({ valid: true, checks: [], errors: [], count: 1 }))

    await api.validateImport(yaml)
    expect(mockFetch).toHaveBeenCalledWith(
      '/api/checks/import/validate',
      expect.objectContaining({
        method: 'POST',
        headers: { 'Content-Type': 'application/x-yaml' },
        body: yaml,
      })
    )
  })
})

describe('api.exportChecks', () => {
  it('fetches YAML without params', async () => {
    mockFetch.mockReturnValueOnce(textResponse('checks: []'))

    const result = await api.exportChecks()
    expect(result).toBe('checks: []')
    expect(mockFetch).toHaveBeenCalledWith(
      '/api/checks/export',
      expect.objectContaining({
        headers: { Accept: 'application/x-yaml' },
        credentials: 'include',
      })
    )
  })

  it('appends project query param', async () => {
    mockFetch.mockReturnValueOnce(textResponse('checks: []'))

    await api.exportChecks('infra')
    const url = mockFetch.mock.calls[0][0] as string
    expect(url).toContain('project=infra')
  })

  it('appends environment query param', async () => {
    mockFetch.mockReturnValueOnce(textResponse('checks: []'))

    await api.exportChecks(undefined, 'prod')
    const url = mockFetch.mock.calls[0][0] as string
    expect(url).toContain('environment=prod')
  })
})

// ── Error handling (request helper) ───────────────────────────────────

describe('error handling', () => {
  it('redirects to /login on 401', async () => {
    mockFetch.mockReturnValueOnce(errorResponse(401))

    await expect(api.getChecks()).rejects.toThrow('Unauthorized')
    expect(window.location.href).toBe('/login')
  })

  it('throws with response body on non-ok', async () => {
    mockFetch.mockReturnValueOnce(errorResponse(500, 'Internal server error'))

    await expect(api.getChecks()).rejects.toThrow('Internal server error')
  })

  it('throws with statusText when body is empty', async () => {
    mockFetch.mockReturnValueOnce(errorResponse(503))

    await expect(api.getChecks()).rejects.toThrow('Error')
  })
})

// ── getCheckRegions ───────────────────────────────────────────────────

describe('api.getCheckRegions', () => {
  it('fetches regions for a check', async () => {
    const data = [{ region: 'us-east', is_healthy: true, message: '', created_at: '' }]
    mockFetch.mockReturnValueOnce(jsonResponse(data))

    const result = await api.getCheckRegions('abc')
    expect(result).toEqual(data)
    expect(mockFetch).toHaveBeenCalledWith(
      '/api/check-definitions/abc/regions',
      expect.objectContaining({ credentials: 'include' })
    )
  })
})
