import type { Ref } from 'vue'

export interface WebSocketEvent {
  type: 'connected' | 'game:created' | 'game:updated' | 'game:deleted' | 'game:wiki_updated' | 'pong'
  gameId?: number
  timestamp: string
  data?: any
  message?: string
}

export type WebSocketStatus = 'connecting' | 'connected' | 'disconnected' | 'error'

export class WebSocketService {
  private status: WebSocketStatus = 'disconnected'
  private statusRef: Ref<WebSocketStatus> | null = null
  private eventCallbacks: Set<(event: WebSocketEvent) => void> = new Set()

  connect(): void {
    this.setStatus('disconnected')
  }

  disconnect(): void {
    this.setStatus('disconnected')
  }

  send(_data: any): void {}

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

export function createWebSocketService(): WebSocketService {
  return new WebSocketService()
}
