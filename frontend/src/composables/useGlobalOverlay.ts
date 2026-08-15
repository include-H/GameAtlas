import { ref } from 'vue'

// 全局覆盖层（开始屏幕等全屏浮层）打开时置 true：底层正在播放的媒体
// （详情页轮播、游戏店 CRT 等）应当暂停，关闭后按播放前状态恢复。
const globalOverlayOpen = ref(false)

export const useGlobalOverlay = () => ({
  overlayOpen: globalOverlayOpen,
  setOverlayOpen: (value: boolean) => {
    globalOverlayOpen.value = value
  },
})
