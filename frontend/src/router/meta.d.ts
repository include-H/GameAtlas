import 'vue-router'
import type { Component } from 'vue'

declare module 'vue-router' {
  interface RouteMeta {
    title?: string
    keepAlive?: boolean
    hideInMenu?: boolean
    hideOnCompactNavigation?: boolean
    activeMenu?: string
    requiresAdmin?: boolean
    icon?: Component
  }
}
