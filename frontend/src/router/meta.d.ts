import 'vue-router'
import type { Component } from 'vue'

declare module 'vue-router' {
  interface RouteMeta {
    title?: string
    hideInMenu?: boolean
    hideOnCompactNavigation?: boolean
    activeMenu?: string
    requiresAdmin?: boolean
    icon?: Component
  }
}
