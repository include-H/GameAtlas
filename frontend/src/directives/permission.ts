import { DirectiveBinding } from 'vue'

/**
 * Element-level permission control is not supported in this project.
 *
 * If this directive is registered manually, it fails closed by hiding the
 * element and logging an error so the UI does not silently expose actions.
 */
function checkPermission(el: HTMLElement, _binding: DirectiveBinding) {
  el.style.display = 'none'

  if (el.dataset.permissionUnsupported !== 'true') {
    el.dataset.permissionUnsupported = 'true'
    console.error(
      '[v-permission] Element-level permission control is not implemented in this project. Use route/menu permission checks instead.',
    )
  }
}

/**
 * Deprecated placeholder kept only as a fail-closed fallback.
 */
export default {
  mounted(el: HTMLElement, binding: DirectiveBinding) {
    checkPermission(el, binding)
  },
  updated(el: HTMLElement, binding: DirectiveBinding) {
    checkPermission(el, binding)
  },
}
