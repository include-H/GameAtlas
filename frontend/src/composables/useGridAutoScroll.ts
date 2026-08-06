import { onBeforeUnmount } from 'vue'

const SCROLL_ZONE = 60
const SCROLL_SPEED = 12

export function useGridAutoScroll() {
  let dragScrollRaf = 0

  const stopGridAutoScroll = () => {
    cancelAnimationFrame(dragScrollRaf)
    dragScrollRaf = 0
  }

  const onGridDragOver = (event: DragEvent, container: HTMLElement | null) => {
    if (!container) return

    const rect = container.getBoundingClientRect()
    let scrollX = 0
    let scrollY = 0
    if (event.clientX < rect.left + SCROLL_ZONE) scrollX = -SCROLL_SPEED
    else if (event.clientX > rect.right - SCROLL_ZONE) scrollX = SCROLL_SPEED
    if (event.clientY < rect.top + SCROLL_ZONE) scrollY = -SCROLL_SPEED
    else if (event.clientY > rect.bottom - SCROLL_ZONE) scrollY = SCROLL_SPEED

    if (scrollX === 0 && scrollY === 0) {
      stopGridAutoScroll()
      return
    }

    if (!dragScrollRaf) {
      const tick = () => {
        container.scrollLeft += scrollX
        container.scrollTop += scrollY
        dragScrollRaf = requestAnimationFrame(tick)
      }
      dragScrollRaf = requestAnimationFrame(tick)
    }
  }

  onBeforeUnmount(stopGridAutoScroll)

  return { onGridDragOver, stopGridAutoScroll }
}
