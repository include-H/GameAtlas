import { ref } from 'vue'
import steamService from '@/services/steam.service'
import type { SteamGameSearchResult } from '@/services/types'
import { getHttpErrorMessage } from '@/utils/http-error'

interface UseSteamPickerOptions<TSelection> {
  onSelect: (game: SteamGameSearchResult) => Promise<TSelection> | TSelection
  onError?: (message: string) => void
}

export const useSteamPicker = <TSelection>(options: UseSteamPickerOptions<TSelection>) => {
  const query = ref('')
  const results = ref<SteamGameSearchResult[]>([])
  const selectedGame = ref<SteamGameSearchResult | null>(null)
  const selectedData = ref<TSelection | null>(null)
  const isSearching = ref(false)

  const clear = () => {
    query.value = ''
    results.value = []
    selectedGame.value = null
    selectedData.value = null
  }

  const resetSelection = () => {
    selectedGame.value = null
    selectedData.value = null
  }

  const search = async () => {
    if (!query.value.trim()) return

    resetSelection()
    isSearching.value = true
    try {
      results.value = await steamService.searchGames(query.value.trim())
    } catch (error) {
      options.onError?.(getHttpErrorMessage(error))
    } finally {
      isSearching.value = false
    }
  }

  const select = async (game: SteamGameSearchResult) => {
    selectedGame.value = game
    selectedData.value = null
    isSearching.value = true
    try {
      selectedData.value = await options.onSelect(game)
    } catch (error) {
      options.onError?.(getHttpErrorMessage(error))
      resetSelection()
    } finally {
      isSearching.value = false
    }
  }

  const back = () => {
    resetSelection()
  }

  return {
    query,
    results,
    selectedGame,
    selectedData,
    isSearching,
    clear,
    search,
    select,
    back,
  }
}
