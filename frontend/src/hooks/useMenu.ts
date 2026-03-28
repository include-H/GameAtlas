import { computed, onMounted, onUnmounted, ref } from 'vue'
import type { Component } from 'vue'
import { useRoute, type RouteRecordRaw } from 'vue-router'
import { appRoutes } from '@/router'
import usePermission from './permission'

export interface MenuItem {
  name: string
  path: string
  locale: string
  icon?: Component
  children?: MenuItem[]
  parentNames?: string[]
  hideInMenu?: boolean
}

/**
 * Generate menu items from routes
 * @param routes - Route records to convert to menu items
 * @param permission - Permission hook to check access
 * @returns Array of menu items
 */
function generateMenuItems(
  routes: RouteRecordRaw[],
  permission: ReturnType<typeof usePermission>,
  isCompactNavigation: boolean,
  parentNames: string[] = []
): MenuItem[] {
  const menuItems: MenuItem[] = []

  for (const route of routes) {
    // Skip routes that should be hidden in menu
    if (route.meta?.hideInMenu) {
      continue
    }

    // Skip routes without locale key
    if (!route.meta?.locale) {
      continue
    }

    if (isCompactNavigation && route.meta?.hideOnCompactNavigation) {
      continue
    }

    // Check if user has permission to see this menu item
    if (!permission.accessRouter(route)) {
      continue
    }

    const routeIcon = route.meta?.icon
    const iconComponent =
      typeof routeIcon === 'function' || (typeof routeIcon === 'object' && routeIcon !== null)
        ? (routeIcon as Component)
        : undefined

    const menuItem: MenuItem = {
      name: route.name as string,
      path: route.path,
      locale: route.meta.locale as string,
      icon: iconComponent,
      parentNames,
      hideInMenu: route.meta?.hideInMenu as boolean | undefined,
    }

    // Process children recursively
    if (route.children && route.children.length > 0) {
      const children = generateMenuItems(route.children, permission, isCompactNavigation, [...parentNames, menuItem.name])
      if (children.length > 0) {
        menuItem.children = children
      }
    }

    menuItems.push(menuItem)
  }

  return menuItems
}

/**
 * Menu hook for Arco Design Vue Pro
 * Generates menu from routes and handles active menu state
 */
export default function useMenu() {
  const route = useRoute()
  const permission = usePermission()
  const viewportWidth = ref(typeof window !== 'undefined' ? window.innerWidth : 1280)

  const syncViewportWidth = () => {
    if (typeof window === 'undefined') return
    viewportWidth.value = window.innerWidth
  }

  onMounted(() => {
    syncViewportWidth()
    window.addEventListener('resize', syncViewportWidth)
  })

  onUnmounted(() => {
    window.removeEventListener('resize', syncViewportWidth)
  })

  /**
   * All menu items generated from routes
   */
  const menuList = computed(() => {
    return generateMenuItems(appRoutes, permission, viewportWidth.value < 992)
  })

  /**
   * Get the currently active menu key
   * Uses route.meta.activeMenu if specified, otherwise uses route name
   */
  const activeKey = computed(() => {
    const activeMenu = route.meta?.activeMenu as string | undefined
    return activeMenu || ((route.name as string | undefined) ?? '')
  })

  const findMenuItemByName = (items: MenuItem[], name: string): MenuItem | undefined => {
    for (const item of items) {
      if (item.name === name) {
        return item
      }

      if (item.children?.length) {
        const target = findMenuItemByName(item.children, name)
        if (target) {
          return target
        }
      }
    }

    return undefined
  }

  /**
   * Get open menu keys (for submenus)
   * Currently returns empty array, can be expanded to track open state
   */
  const openKeys = computed<string[]>(() => {
    const current = findMenuItemByName(menuList.value, activeKey.value)
    return current?.parentNames ?? []
  })

  return {
    menuList,
    activeKey,
    openKeys,
  }
}
