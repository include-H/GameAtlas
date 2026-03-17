/**
 * Arco Design Mobile Adaptation - Flexible.js
 * 基于 rem 的移动端适配方案
 *
 * 设计稿宽度: 375px (iPhone 6/7/8 标准)
 * 基准字体大小: 50px (1rem = 50px)
 * 换算比例: 1px = 0.02rem
 *
 * 使用示例:
 * 设计稿中 100px -> CSS 中 1rem (或 100/50 = 2rem)
 */

interface FlexibleOptions {
  /** 基准字体大小，默认 50 */
  baseFontSize?: number
  /** 设计稿宽度，默认 375 */
  sketchWidth?: number
  /** 最大字体大小限制，默认 64 */
  maxFontSize?: number
  /** 最小宽度限制，默认 320 */
  minWidth?: number
  /** 最大宽度限制，默认 540 (适合移动端) */
  maxWidth?: number
}

/**
 * 设置根元素字体大小，实现 rem 适配
 * @param options 配置选项
 * @returns 取消监听的函数
 */
export function setRootPixel(options: FlexibleOptions = {}): () => void {
  const {
    baseFontSize = 50,
    sketchWidth = 375,
    maxFontSize = 64,
    minWidth = 320,
    maxWidth = 540,
  } = options

  const docEl = document.documentElement

  function setRem(): void {
    const clientWidth = docEl.clientWidth

    // 只在移动端生效
    if (clientWidth > maxWidth) {
      docEl.style.fontSize = ''
      return
    }

    // 限制最小宽度
    const width = Math.max(clientWidth, minWidth)

    // 计算 rem 比例
    const rem = (width / sketchWidth) * baseFontSize

    // 限制最大字体大小
    const finalRem = Math.min(rem, maxFontSize)

    docEl.style.fontSize = `${finalRem}px`
  }

  // 初始化
  setRem()

  // 监听窗口变化
  window.addEventListener('resize', setRem)
  window.addEventListener('orientationchange', setRem)

  // 返回取消监听的函数
  return function removeRootPixel(): void {
    window.removeEventListener('resize', setRem)
    window.removeEventListener('orientationchange', setRem)
    docEl.style.fontSize = ''
  }
}

/**
 * 将 px 转换为 rem
 * @param px 设计稿中的像素值
 * @param baseFontSize 基准字体大小
 * @returns rem 值
 */
export function px2rem(px: number, baseFontSize: number = 50): string {
  return `${px / baseFontSize}rem`
}

/**
 * 检查是否为移动端设备
 * @returns boolean
 */
export function isMobile(): boolean {
  return window.matchMedia('(max-width: 768px)').matches
}

/**
 * 检查是否为触摸设备
 * @returns boolean
 */
export function isTouchDevice(): boolean {
  return 'ontouchstart' in window || navigator.maxTouchPoints > 0
}

export default setRootPixel
