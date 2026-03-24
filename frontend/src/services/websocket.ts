import type { Ref } from 'vue'

interface BaseWebSocketEvent {
  timestamp: string
  message?: string
}

interface ConnectedWebSocketEvent extends BaseWebSocketEvent {
  type: 'connected'
}

interface PongWebSocketEvent extends BaseWebSocketEvent {
  type: 'pong'
}

interface GameWebSocketEvent extends BaseWebSocketEvent {
  type: 'game:created' | 'game:updated' | 'game:deleted' | 'game:wiki_updated'
  gameId: number
}

export type WebSocketEvent =
  | ConnectedWebSocketEvent
  | PongWebSocketEvent
  | GameWebSocketEvent

export interface WebSocketSendPayload {
  type: string
  gameId?: number
  [key: string]: string | number | boolean | null | undefined
}

type WebSocketStatus = 'connecting' | 'connected' | 'disconnected' | 'error'

class WebSocketService {
  private status: WebSocketStatus = 'disconnected'
  private statusRef: Ref<WebSocketStatus> | null = null
  private eventCallbacks: Set<(event: WebSocketEvent) => void> = new Set()

  connect(): void {
    this.setStatus('disconnected')
  }

  disconnect(): void {
    this.setStatus('disconnected')
  }

  send(_data: WebSocketSendPayload): void {}

  isConnected(): boolean {
    return false
  }

  getStatus(): WebSocketStatus {
    return this.status
  }

  setStatusRef(ref: Ref<WebSocketStatus>): void {
    this.statusRef = ref
  }

  subscribe(callback: (event: WebSocketEvent) => void): () => void {
    this.eventCallbacks.add(callback)
    return () => {
      this.eventCallbacks.delete(callback)
    }
  }

  private setStatus(status: WebSocketStatus): void {
    this.status = status
    if (this.statusRef) {
      this.statusRef.value = status
    }
  }
}

let instance: WebSocketService | null = null

export function getWebSocketService(): WebSocketService {
  if (!instance) {
    instance = new WebSocketService()
  }
  return instance
}
