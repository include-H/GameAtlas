import 'vue-router'
import type { Component } from 'vue'

declare module 'vue-router' {
  interface RouteMeta {
    locale?: string
    keepAlive?: boolean
    hideInMenu?: boolean
    hideOnCompactNavigation?: boolean
    activeMenu?: string
    requiresAdmin?: boolean
    icon?: Component
  }
}
