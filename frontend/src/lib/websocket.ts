export type WSMessage =
  | { type: 'checks'; checks: Check[]; count: number; timestamp: number }
  | { type: 'update'; check: Check }

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
}

type OnMessage = (msg: WSMessage) => void
type OnStatus = (status: 'connected' | 'disconnected' | 'connecting') => void

export class WebSocketManager {
  private ws: WebSocket | null = null
  private url: string
  private onMessage: OnMessage
  private onStatus: OnStatus
  private reconnectTimer: ReturnType<typeof setTimeout> | null = null
  private shouldReconnect = true

  constructor(onMessage: OnMessage, onStatus: OnStatus) {
    const proto = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
    this.url = `${proto}//${window.location.host}/ws`
    this.onMessage = onMessage
    this.onStatus = onStatus
  }

  connect() {
    this.shouldReconnect = true
    this.onStatus('connecting')

    try {
      this.ws = new WebSocket(this.url)
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
    this.reconnectTimer = setTimeout(() => this.connect(), 3000)
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
