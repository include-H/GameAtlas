import { ref, type Ref } from 'vue'
import { useRouter } from 'vue-router'
import gamesService, { mapGameVersions } from '@/services/games.service'
import type { GameVersion } from '@/services/types'

export interface LaunchOption {
  id: string
  version: string
  url: string
}

export interface UseLaunchGameOptions {
  /** 开盒动画进行中标记（父组件持有），用于防重复开盒与放回时立即收场的判断 */
  isOpening: Ref<boolean>
  /** 无可启动版本 / 详情拉取失败时回调，触发 GameInspect.putBack() */
  requestPutBack: () => void
}

/** 开盒动画（盒盖 0.6s 翻转）播完即出手，不再额外等待 */
const OPEN_CASE_DELAY_MS = 600

/**
 * 开盒即开玩（与开始屏幕一致）：详情拉取与开盒动画并行，动画结束直接启动；
 * 无可启动版本或拉取失败才回退详情页，多个版本时弹窗选择。
 */
export const useLaunchGame = ({ isOpening, requestPutBack }: UseLaunchGameOptions) => {
  const router = useRouter()

  const launchModalVisible = ref(false)
  const launchTitle = ref('')
  const launchOptions = ref<LaunchOption[]>([])
  const launchHint = ref('点击游戏盒直接开始游戏')
  const launchHintSuccess = ref(false)

  let pendingLaunchPublicId: string | null = null
  let openCaseTimer: number | null = null

  const launchVersion = (version: GameVersion) => {
    if (!version.launchScriptUrl) return
    window.location.assign(version.launchScriptUrl)
    launchHint.value = `已生成启动脚本：${version.version}，请查看浏览器下载`
    launchHintSuccess.value = true
  }

  const handleOpenCase = (publicId: string) => {
    if (isOpening.value) return
    isOpening.value = true
    pendingLaunchPublicId = publicId
    const detailPromise = gamesService.getGameDetail(publicId).catch(() => null)

    openCaseTimer = window.setTimeout(async () => {
      openCaseTimer = null
      const detail = await detailPromise
      if (pendingLaunchPublicId !== publicId) return
      pendingLaunchPublicId = null
      if (!detail) {
        requestPutBack()
        router.push({ name: 'game-detail', params: { publicId } })
        return
      }
      const launchable = mapGameVersions(detail).filter(
        (version) => version.canLaunch && version.launchScriptUrl,
      )
      if (launchable.length === 0) {
        requestPutBack()
        router.push({ name: 'game-detail', params: { publicId } })
        return
      }
      if (launchable.length === 1) {
        launchVersion(launchable[0])
        return
      }
      launchTitle.value = detail.title
      launchOptions.value = launchable.map((version) => ({
        id: version.id,
        version: version.version,
        url: version.launchScriptUrl!,
      }))
      launchModalVisible.value = true
    }, OPEN_CASE_DELAY_MS)
  }

  const handleLaunchVersion = (option: LaunchOption) => {
    launchModalVisible.value = false
    window.location.assign(option.url)
    launchHint.value = `已生成启动脚本：${option.version}，请查看浏览器下载`
    launchHintSuccess.value = true
  }

  /** 放回 / 卸载时清掉挂起的开盒定时器与待启动版本，阻断开盒回调的后续动作 */
  const cancelPending = () => {
    if (openCaseTimer !== null) {
      window.clearTimeout(openCaseTimer)
      openCaseTimer = null
    }
    pendingLaunchPublicId = null
  }

  const resetHint = () => {
    launchHint.value = '点击游戏盒直接开始游戏'
    launchHintSuccess.value = false
  }

  const dispose = () => {
    cancelPending()
  }

  return {
    launchModalVisible,
    launchTitle,
    launchOptions,
    launchHint,
    launchHintSuccess,
    handleOpenCase,
    handleLaunchVersion,
    cancelPending,
    resetHint,
    dispose,
  }
}
