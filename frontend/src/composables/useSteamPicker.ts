import { readonly, ref } from 'vue'
import steamService from '@/services/steam.service'
import type { SteamGameSearchResult } from '@/services/types'
import { getHttpErrorMessage } from '@/utils/http-error'
import { createRequestGeneration, type RequestGenerationGuard } from '@/utils/request-generation'

interface UseSteamPickerOptions<TSelection> {
  onSelect: (
    game: SteamGameSearchResult,
    request: SteamPickerRequest,
  ) => Promise<TSelection> | TSelection
  onError?: (message: string) => void
}

export type SteamPickerRequest = RequestGenerationGuard

export const useSteamPicker = <TSelection>(options: UseSteamPickerOptions<TSelection>) => {
  const query = ref('')
  const results = ref<SteamGameSearchResult[]>([])
  const selectedGame = ref<SteamGameSearchResult | null>(null)
  const selectedData = ref<TSelection | null>(null)
  const isSearching = ref(false)
  const requests = createRequestGeneration()

  const invalidateRequest = () => {
    requests.invalidate()
    isSearching.value = false
  }

  const beginRequest = (): SteamPickerRequest => requests.begin()

  const clear = () => {
    invalidateRequest()
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
    const searchQuery = query.value.trim()
    if (!searchQuery) {
      invalidateRequest()
      results.value = []
      resetSelection()
      return
    }

    const request = beginRequest()
    resetSelection()
    isSearching.value = true
    try {
      const nextResults = await steamService.searchGames(searchQuery)
      if (request.isCurrent()) {
        results.value = nextResults
      }
    } catch (error) {
      if (request.isCurrent()) {
        options.onError?.(getHttpErrorMessage(error))
      }
    } finally {
      if (request.isCurrent()) {
        isSearching.value = false
      }
    }
  }

  const select = async (game: SteamGameSearchResult) => {
    const request = beginRequest()
    selectedGame.value = game
    selectedData.value = null
    isSearching.value = true
    try {
      const data = await options.onSelect(game, request)
      if (request.isCurrent()) {
        selectedData.value = data
      }
    } catch (error) {
      if (request.isCurrent()) {
        options.onError?.(getHttpErrorMessage(error))
        resetSelection()
      }
    } finally {
      if (request.isCurrent()) {
        isSearching.value = false
      }
    }
  }

  const back = () => {
    invalidateRequest()
    resetSelection()
  }

  const setQuery = (next: string) => {
    query.value = next
  }

  const setSelectedGame = (game: SteamGameSearchResult | null) => {
    invalidateRequest()
    selectedData.value = null
    selectedGame.value = game
  }

  const setSearching = (next: boolean) => {
    isSearching.value = next
  }

  const clearResults = () => {
    invalidateRequest()
    results.value = []
  }

  const selectExternal = async (
    game: SteamGameSearchResult,
    load: (request: SteamPickerRequest) => Promise<void> | void,
  ) => {
    const request = beginRequest()
    resetSelection()
    selectedGame.value = game
    isSearching.value = true
    try {
      await load(request)
    } catch (error) {
      if (request.isCurrent()) {
        resetSelection()
        throw error
      }
      return
    } finally {
      if (request.isCurrent()) {
        isSearching.value = false
      }
    }
  }

  return {
    query,
    results: readonly(results),
    selectedGame: readonly(selectedGame),
    selectedData: readonly(selectedData),
    isSearching: readonly(isSearching),
    clear,
    clearResults,
    search,
    select,
    selectExternal,
    setQuery,
    setSelectedGame,
    setSearching,
    back,
  }
}
