import { computed } from 'vue'
import { useRoute, type RouteRecordRaw } from 'vue-router'
import { appRoutes } from '@/router'
import usePermission from './permission'

export interface MenuItem {
  name: string
  path: string
  locale: string
  icon?: any
  children?: MenuItem[]
  parentNames?: string[]
  hideInMenu?: boolean
  roles?: string[]
  requiresAuth?: boolean
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

    // Check if user has permission to see this menu item
    if (!permission.accessRouter(route)) {
      continue
    }

    const menuItem: MenuItem = {
      name: route.name as string,
      path: route.path,
      locale: route.meta.locale as string,
      icon: route.meta?.icon,
      parentNames,
      hideInMenu: route.meta?.hideInMenu as boolean | undefined,
      roles: route.meta?.roles as string[],
      requiresAuth: route.meta?.requiresAuth as boolean,
    }

    // Process children recursively
    if (route.children && route.children.length > 0) {
      const children = generateMenuItems(route.children, permission, [...parentNames, menuItem.name])
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

  /**
   * All menu items generated from routes
   */
  const menuList = computed(() => {
    return generateMenuItems(appRoutes, permission)
  })

  /**
   * Get the currently active menu key
   * Uses route.meta.activeMenu if specified, otherwise uses route name
   */
  const activeKey = computed(() => {
    const activeMenu = route.meta?.activeMenu as string | undefined
    return activeMenu || (route.name as string)
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
