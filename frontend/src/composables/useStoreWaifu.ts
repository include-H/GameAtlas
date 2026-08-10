import { ref, type Ref } from 'vue'

interface Live2DWidgetConfig {
  waifuPath: string
  cdnPath: string
  cubism2Path: string
  cubism5Path: string
  tools: string[]
  drag: boolean
  logLevel: string
  modelId: number
  showToggleAfterQuit: boolean
}

declare global {
  interface Window {
    initWidget?: (config: Live2DWidgetConfig) => void
    __waifuManager?: {
      cubism2model?: {
        gl?: unknown
        modelScaling: (factor: number) => void
        viewMatrix?: {
          getScaleX?: () => number
        }
      }
    }
  }
}

const WAIFU_TARGET_ZOOM = 1.35

interface UseStoreWaifuOptions {
  /** 看板娘挂进店内场景用的 stage 容器 ref（对应 .store-stage 元素） */
  stageRef: Ref<HTMLElement | null>
}

/**
 * 看板娘：live2d-widget 脚本会在 body 下创建节点，
 * 这里负责加载脚本/样式、把 #waifu 挪进场景内跟随 1280×720 缩放定位、并校准人物缩放。
 * 全局定位样式在 assets/game-store-waifu.css（#waifu 是 body 级节点，必须全局可访问）。
 */
export const useStoreWaifu = ({ stageRef }: UseStoreWaifuOptions) => {
  const disposed = ref(false)

  let waifuStyleTag: HTMLLinkElement | null = null
  let waifuScriptTag: HTMLScriptElement | null = null
  let waifuZoomTimer: number | null = null
  let initGeneration = 0

  const loadWaifuResource = (url: string, type: 'css' | 'js'): Promise<void> => {
    return new Promise((resolve, reject) => {
      if (type === 'css') {
        const link = document.createElement('link')
        link.rel = 'stylesheet'
        link.href = url
        link.onload = () => resolve()
        link.onerror = () => reject(new Error(`看板娘样式加载失败：${url}`))
        document.head.appendChild(link)
        waifuStyleTag = link
        return
      }
      const script = document.createElement('script')
      script.type = 'module'
      script.src = url
      script.onload = () => resolve()
      script.onerror = () => reject(new Error(`看板娘脚本加载失败：${url}`))
      document.head.appendChild(script)
      waifuScriptTag = script
    })
  }

  const waitForElement = (
    selector: string,
    timeoutMs = 8000,
    isCurrent: () => boolean = () => !disposed.value,
  ): Promise<HTMLElement | null> => {
    if (!isCurrent()) return Promise.resolve(null)
    const existing = document.querySelector<HTMLElement>(selector)
    if (existing) return Promise.resolve(existing)
    return new Promise((resolve) => {
      const startedAt = Date.now()
      const timer = window.setInterval(() => {
        if (!isCurrent()) {
          window.clearInterval(timer)
          resolve(null)
          return
        }
        const element = document.querySelector<HTMLElement>(selector)
        if (element) {
          window.clearInterval(timer)
          resolve(element)
          return
        }
        if (Date.now() - startedAt > timeoutMs) {
          window.clearInterval(timer)
          resolve(null)
        }
      }, 100)
    })
  }

  const init = async () => {
    const generation = ++initGeneration
    const isCurrent = () => generation === initGeneration && !disposed.value
    if (!isCurrent()) return
    // 热更新或重复挂载时先清掉旧的看板娘节点
    cleanupResources()

    await loadWaifuResource('/live2d-widget/waifu.css', 'css')
    if (!isCurrent()) return
    if (!window.initWidget) {
      await loadWaifuResource('/live2d-widget/waifu-tips.js', 'js')
      if (!isCurrent()) return
    }

    // 清掉旧 manager，避免上一次会话的缩放状态被误判为“已生效”
    window.__waifuManager = undefined

    if (!isCurrent()) return
    window.initWidget?.({
      waifuPath: '/live2d-config/waifu-tips.json',
      cdnPath: '/live2d-models/',
      cubism2Path: '/live2d-widget/live2d.min.js',
      cubism5Path: '',
      tools: [],
      drag: false,
      logLevel: 'warn',
      modelId: 0,
      showToggleAfterQuit: false,
    })

    // 把看板娘挂到场景内部，跟随 1280×720 设计稿一起缩放定位
    const stageElement = stageRef.value ?? document.querySelector<HTMLElement>('.store-stage')
    const waifuElement = await waitForElement('#waifu', 8000, isCurrent)
    if (!isCurrent()) return
    if (stageElement && waifuElement) {
      stageElement.appendChild(waifuElement)
    }

    // 模型加载完成后把人物缩放到目标值（鼠标滚轮的缩放不会自动保存）
    const applyTargetZoom = () => {
      const model = window.__waifuManager?.cubism2model
      // 必须等模型初始化完成（gl 就绪、viewMatrix 存在）再设置缩放
      if (!model || !model.gl || !model.viewMatrix) return false
      const current = model.viewMatrix?.getScaleX?.() ?? 1
      if (Math.abs(current - WAIFU_TARGET_ZOOM) > 0.01) {
        model.modelScaling(WAIFU_TARGET_ZOOM / current)
      }
      return true
    }
    if (!isCurrent()) return
    if (!applyTargetZoom()) {
      const startedAt = Date.now()
      waifuZoomTimer = window.setInterval(() => {
        if (!isCurrent() || applyTargetZoom() || Date.now() - startedAt > 15000) {
          if (waifuZoomTimer !== null) {
            window.clearInterval(waifuZoomTimer)
            waifuZoomTimer = null
          }
        }
      }, 200)
    }
  }

  const cleanupResources = () => {
    if (waifuZoomTimer !== null) {
      window.clearInterval(waifuZoomTimer)
      waifuZoomTimer = null
    }
    document.getElementById('waifu')?.remove()
    document.getElementById('waifu-toggle')?.remove()
    waifuScriptTag?.remove()
    waifuStyleTag?.remove()
    waifuScriptTag = null
    waifuStyleTag = null
  }

  const cleanup = () => {
    initGeneration += 1
    cleanupResources()
  }

  const dispose = () => {
    disposed.value = true
    initGeneration += 1
  }

  const reset = () => {
    disposed.value = false
    initGeneration += 1
  }

  return {
    disposed,
    init,
    cleanup,
    dispose,
    reset,
  }
}
