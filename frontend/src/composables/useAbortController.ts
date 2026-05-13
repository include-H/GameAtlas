import { onUnmounted, ref } from 'vue'

/**
 * Creates an AbortController that is automatically aborted when the component unmounts.
 * Use the returned signal in API requests to prevent stale responses after route changes.
 */
export function useAbortController() {
  const controller = ref(new AbortController())

  const reset = () => {
    controller.value.abort()
    controller.value = new AbortController()
  }

  const abort = () => {
    controller.value.abort()
  }

  onUnmounted(() => {
    controller.value.abort()
  })

  return {
    signal: () => controller.value.signal,
    reset,
    abort,
  }
}
