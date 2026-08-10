import { computed, ref, type Ref } from 'vue'
import {
  canCreateRemoteSearchedOption,
  dedupeCreatableOptionsByName,
  normalizeCreatableOptionName,
  searchCreatableOptions,
} from '@/utils/creatable-select'
import { createRequestGeneration, type RequestGenerationGuard } from '@/utils/request-generation'

interface MetadataOption {
  id: number
  name: string
}

interface UseRemoteMetadataPickerOptions<T extends MetadataOption> {
  selectedIds: () => number[]
  search: (query: string) => Promise<T[]>
  create: (name: string) => Promise<T>
}

export const normalizeMetadataPickerID = (value: unknown): number | null => {
  return typeof value === 'number' && Number.isInteger(value) && value > 0 ? value : null
}

export const normalizeMetadataPickerIDs = (value: unknown): number[] => {
  if (!Array.isArray(value)) return []
  return Array.from(new Set(value
    .map((item) => normalizeMetadataPickerID(item))
    .filter((item): item is number => item !== null)))
}

export const useRemoteMetadataPicker = <T extends MetadataOption>(
  pickerOptions: UseRemoteMetadataPickerOptions<T>,
) => {
  const options = ref<T[]>([]) as Ref<T[]>
  const query = ref('')
  const resolvedQuery = ref('')
  const isSearching = ref(false)
  const isCreating = ref(false)
  let searchRequestID = 0
  const creationRequests = createRequestGeneration()

  const canCreate = computed(() => {
    return canCreateRemoteSearchedOption(query.value, resolvedQuery.value, options.value)
  })

  const search = async (value: string) => {
    query.value = value
    resolvedQuery.value = ''
    creationRequests.invalidate()
    isCreating.value = false
    const requestID = ++searchRequestID
    const normalizedQuery = normalizeCreatableOptionName(value)
    if (!normalizedQuery) {
      isSearching.value = false
      return
    }

    isSearching.value = true
    try {
      const results = await searchCreatableOptions({
        query: normalizedQuery,
        selectedValues: pickerOptions.selectedIds(),
        currentOptions: options.value,
        search: pickerOptions.search,
      })
      if (requestID !== searchRequestID) return
      options.value = results
      resolvedQuery.value = normalizedQuery
    } catch (error) {
      if (requestID === searchRequestID) {
        throw error
      }
    } finally {
      if (requestID === searchRequestID) {
        isSearching.value = false
      }
    }
  }

  const ensureNamesForRequest = async (names: string[], request: RequestGenerationGuard) => {
    const uniqueNames = new Map<string, string>()
    for (const rawName of names) {
      const name = rawName.trim()
      const normalizedName = normalizeCreatableOptionName(name)
      if (normalizedName && !uniqueNames.has(normalizedName)) {
        uniqueNames.set(normalizedName, name)
      }
    }

    const ids: number[] = []
    for (const [normalizedName, name] of uniqueNames) {
      if (!request.isCurrent()) return []
      const existing = options.value.find(
        (item) => normalizeCreatableOptionName(item.name) === normalizedName,
      )
      const item = existing || await pickerOptions.create(name)
      if (!request.isCurrent()) return []
      if (!existing) {
        options.value = dedupeCreatableOptionsByName([item, ...options.value])
      }
      ids.push(item.id)
    }

    return ids
  }

  const ensureNames = async (names: string[]) => {
    return ensureNamesForRequest(names, creationRequests.begin())
  }

  const createFromQuery = async () => {
    if (!canCreate.value || isCreating.value) return null

    const request = creationRequests.begin()
    searchRequestID += 1
    resolvedQuery.value = ''
    isSearching.value = false
    isCreating.value = true
    try {
      const [id] = await ensureNamesForRequest([query.value], request)
      if (!request.isCurrent()) return null
      return options.value.find((item) => item.id === id) || null
    } finally {
      if (request.isCurrent()) {
        isCreating.value = false
      }
    }
  }

  const resolveQuery = async () => {
    const normalizedQuery = normalizeCreatableOptionName(query.value)
    if (!normalizedQuery || isCreating.value) return null

    if (isSearching.value || resolvedQuery.value !== normalizedQuery) {
      await search(query.value)
    }

    const existing = options.value.find(
      (item) => normalizeCreatableOptionName(item.name) === normalizedQuery,
    )
    return existing || createFromQuery()
  }

  const clearSearch = () => {
    query.value = ''
    resolvedQuery.value = ''
    searchRequestID += 1
    creationRequests.invalidate()
    isSearching.value = false
    isCreating.value = false
  }

  const reset = () => {
    clearSearch()
    isCreating.value = false
  }

  return {
    canCreate,
    clearSearch,
    createFromQuery,
    ensureNames,
    isCreating,
    isSearching,
    options,
    query,
    reset,
    resolveQuery,
    search,
  }
}
