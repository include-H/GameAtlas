import type { App } from 'vue'
import permission from './permission'

/**
 * Register all directives
 */
export default function registerDirectives(app: App) {
  app.directive('permission', permission)
}
