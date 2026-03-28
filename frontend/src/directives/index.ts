import type { App } from 'vue'

/**
 * Register custom directives.
 *
 * The current project does not support element-level permission control.
 * `v-permission` is intentionally not registered to avoid creating a false
 * sense of security.
 */
export default function registerDirectives(_app: App) {
  return
}
