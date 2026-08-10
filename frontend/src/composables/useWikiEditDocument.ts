import { computed, getCurrentInstance, onUnmounted, ref, type Ref } from 'vue'
import wikiService, { type WikiDocumentResponse } from '@/services/wiki.service'
import { getHttpErrorMessage } from '@/utils/http-error'
import { useGamesStore } from '@/stores/games'
import { useUiStore } from '@/stores/ui'
import { createRequestGeneration } from '@/utils/request-generation'

interface UseWikiEditDocumentOptions {
  gamesStore: ReturnType<typeof useGamesStore>
  uiStore: ReturnType<typeof useUiStore>
  requestedGameId: Ref<string>
  onLoadGameFailed: () => void | Promise<void>
  onSaveSuccess?: (gameId: string) => void | Promise<void>
}

export const useWikiEditDocument = ({
  gamesStore,
  uiStore,
  requestedGameId,
  onLoadGameFailed,
  onSaveSuccess,
}: UseWikiEditDocumentOptions) => {
  const documentRequests = createRequestGeneration()
  const game = computed(() => {
    // 2026-04-08: wiki edit reads the route-targeted game only.
    // Impact: route changes or failed loads must not leave the editor rendering a stale
    // currentGame from a previous page.
    return gamesStore.currentGame?.public_id === requestedGameId.value
      ? gamesStore.currentGame
      : null
  })
  const wiki = ref<WikiDocumentResponse | null>(null)
  const isSaving = ref(false)
  const wikiData = ref({
    content: '',
    change_summary: '',
  })

  const isExisting = computed(() => Boolean(wiki.value && wiki.value.content !== null))

  const resetWikiEditorState = () => {
    wiki.value = null
    wikiData.value = {
      content: '',
      change_summary: '',
    }
  }

  const loadWikiEditorData = async (gameId: string): Promise<boolean> => {
    const request = documentRequests.begin()
    resetWikiEditorState()
    try {
      await gamesStore.fetchGame(gameId)
      if (!request.isCurrent() || requestedGameId.value !== gameId || !game.value) {
        return false
      }
    } catch {
      if (!request.isCurrent() || requestedGameId.value !== gameId) {
        return false
      }
      uiStore.addAlert('加载游戏失败', 'error')
      await onLoadGameFailed()
      return false
    }

    try {
      const wikiContent = await wikiService.getWikiPage(gameId)
      if (!request.isCurrent() || requestedGameId.value !== gameId) {
        return false
      }
      // 2026-04-06: the wiki endpoint returns a document envelope for existing games
      // even when content is empty, so the editor must not treat empty text as "no wiki".
      // A 404 here is a broken cross-endpoint contract, not a valid "missing document" state.
      wiki.value = wikiContent
      wikiData.value = {
        content: wikiContent.content ?? '',
        change_summary: wikiContent.content === null ? '首次添加' : '',
      }
    } catch (error) {
      if (!request.isCurrent() || requestedGameId.value !== gameId) {
        return false
      }
      uiStore.addAlert(getHttpErrorMessage(error, '加载 Wiki 失败'), 'error')
      await onLoadGameFailed()
      return false
    }

    return true
  }

  const handleSave = async () => {
    const gameId = requestedGameId.value
    if (!gameId || !game.value?.public_id || game.value.public_id !== gameId) return
    const request = documentRequests.begin()

    isSaving.value = true

    try {
      const wasExisting = isExisting.value
      const document = await wikiService.updateWikiPage(gameId, {
        content: wikiData.value.content,
        change_summary: wikiData.value.change_summary.trim() || undefined,
      })
      if (!request.isCurrent() || requestedGameId.value !== gameId) return
      wiki.value = document

      uiStore.addAlert(wasExisting ? 'Wiki 已更新' : 'Wiki 已创建', 'success')
      wikiData.value.change_summary = ''
      await onSaveSuccess?.(gameId)
    } catch (error) {
      if (!request.isCurrent() || requestedGameId.value !== gameId) return
      const errorMessage = getHttpErrorMessage(error, '保存 Wiki 失败')
      uiStore.addAlert(errorMessage, 'error')
    } finally {
      if (request.isCurrent()) {
        isSaving.value = false
      }
    }
  }

  if (getCurrentInstance()) {
    onUnmounted(() => {
      documentRequests.invalidate()
    })
  }

  return {
    game,
    wiki,
    wikiData,
    isSaving,
    isExisting,
    loadWikiEditorData,
    handleSave,
  }
}
