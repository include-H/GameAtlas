import { DirectiveBinding } from 'vue'

/**
 * Check permission for element
 * (Authentication removed - all permissions granted)
 */
function checkPermission(_el: HTMLElement, _binding: DirectiveBinding) {
  // No auth required - all users have all permissions
  return
}

/**
 * v-permission directive for Arco Design Vue Pro
 * Usage: <a-button v-permission="['admin']">Delete</a-button>
 *
 * (Authentication removed - element is always shown)
 */
export default {
  mounted(el: HTMLElement, binding: DirectiveBinding) {
    checkPermission(el, binding)
  },
  updated(el: HTMLElement, binding: DirectiveBinding) {
    checkPermission(el, binding)
  },
}
