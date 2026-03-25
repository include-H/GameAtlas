import { DirectiveBinding } from 'vue'

/**
 * Permission check placeholder for element-level controls.
 * This directive currently does not hide or disable elements.
 */
function checkPermission(_el: HTMLElement, _binding: DirectiveBinding) {
  // Element-level permission enforcement has not been implemented yet.
  return
}

/**
 * v-permission directive for Arco Design Vue Pro
 * Usage: <a-button v-permission="['admin']">Delete</a-button>
 * The directive is currently a no-op and always leaves the element visible.
 */
export default {
  mounted(el: HTMLElement, binding: DirectiveBinding) {
    checkPermission(el, binding)
  },
  updated(el: HTMLElement, binding: DirectiveBinding) {
    checkPermission(el, binding)
  },
}
