/**
 * WebSocket manager factory for the Checker UI.
 *
 * Usage:
 *   // Standalone (no auth):
 *   const ws = createWebSocket(onMessage, onStatus)
 *
 *   // Cloud (JWT auth):
 *   const ws = createWebSocket(onMessage, onStatus, {
 *     getAuthToken: () => localStorage.getItem('token'),
 *   })
 */

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

export type WSMessage =
  | { type: 'checks'; checks: Check[]; count: number; timestamp: number }
  | { type: 'update'; check: Check }
  | { type: 'alert_new'; alert: AlertEvent }
  | { type: 'alert_resolved'; check_uuid: string }

export interface Check {
  ID: string
  Name: string
  Project: string
  Healthcheck: string
  LastResult: boolean
  LastExec: string
  LastPing: string
  Enabled: boolean
  UUID: string
  CheckType: string
  Message: string
  Host: string
  Periodicity: string
  URL: string
  IsSilenced: boolean
  SilencedChannels?: string[]
  RunMode?: string
  TargetRegions?: string[]
}

type OnMessage = (msg: WSMessage) => void
type OnStatus = (status: 'connected' | 'disconnected' | 'connecting') => void

export interface WebSocketConfig {
  /** Custom WebSocket URL. Default: auto-detected from window.location */
  url?: string
  /** Function to get auth token (appended as query param) */
  getAuthToken?: () => string | null
  /** Reconnect delay in ms. Default: 3000 */
  reconnectDelay?: number
}

export class WebSocketManager {
  private ws: WebSocket | null = null
  private wsUrl: string
  private onMessage: OnMessage
  private onStatus: OnStatus
  private reconnectTimer: ReturnType<typeof setTimeout> | null = null
  private shouldReconnect = true
  private config: WebSocketConfig

  constructor(onMessage: OnMessage, onStatus: OnStatus, config: WebSocketConfig = {}) {
    this.config = config
    const proto = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
    this.wsUrl = config.url || `${proto}//${window.location.host}/ws`
    this.onMessage = onMessage
    this.onStatus = onStatus
  }

  connect() {
    this.shouldReconnect = true
    this.onStatus('connecting')

    let url = this.wsUrl
    if (this.config.getAuthToken) {
      const token = this.config.getAuthToken()
      if (token) {
        const separator = url.includes('?') ? '&' : '?'
        url = `${url}${separator}token=${encodeURIComponent(token)}`
      }
    }

    try {
      this.ws = new WebSocket(url)
    } catch {
      this.scheduleReconnect()
      return
    }

    this.ws.onopen = () => {
      this.onStatus('connected')
      this.ws?.send(JSON.stringify({ action: 'getChecks' }))
    }

    this.ws.onmessage = (event) => {
      try {
        const data = JSON.parse(event.data) as WSMessage
        this.onMessage(data)
      } catch {
        // ignore parse errors
      }
    }

    this.ws.onclose = () => {
      this.onStatus('disconnected')
      this.scheduleReconnect()
    }

    this.ws.onerror = () => {
      this.ws?.close()
    }
  }

  private scheduleReconnect() {
    if (!this.shouldReconnect) return
    this.reconnectTimer = setTimeout(
      () => this.connect(),
      this.config.reconnectDelay || 3000
    )
  }

  disconnect() {
    this.shouldReconnect = false
    if (this.reconnectTimer) clearTimeout(this.reconnectTimer)
    this.ws?.close()
  }

  send(data: object) {
    if (this.ws?.readyState === WebSocket.OPEN) {
      this.ws.send(JSON.stringify(data))
    }
  }
}

/** Factory function for creating WebSocket managers */
export function createWebSocket(
  onMessage: OnMessage,
  onStatus: OnStatus,
  config?: WebSocketConfig
): WebSocketManager {
  return new WebSocketManager(onMessage, onStatus, config)
}
