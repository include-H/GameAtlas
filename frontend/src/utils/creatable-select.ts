interface CreatableNamedOption {
  id: number
  name: string
}

type CreatableSelectionValue = string | number | boolean | null | undefined

export const normalizeOptionId = (value: CreatableSelectionValue): number | null => {
  if (typeof value === 'number' && !Number.isNaN(value)) return value
  return null
}

export const normalizeCreatableOptionName = (value: string) => {
  return value.trim().replace(/\s+/g, ' ').toLowerCase()
}

export const sortCreatableOptionsByName = <T extends CreatableNamedOption>(options: T[]) => {
  return [...options].sort((a, b) => a.name.localeCompare(b.name, 'zh-Hans-CN'))
}

export const hasCreatableOptionName = <T extends CreatableNamedOption>(name: string, options: T[]) => {
  const normalizedName = normalizeCreatableOptionName(name)
  return normalizedName.length > 0 && options.some((item) => normalizeCreatableOptionName(item.name) === normalizedName)
}

export const canCreateRemoteSearchedOption = <T extends CreatableNamedOption>(
  query: string,
  resolvedQuery: string,
  options: T[],
) => {
  const normalizedQuery = normalizeCreatableOptionName(query)
  return normalizedQuery.length > 0
    && normalizedQuery === resolvedQuery
    && !hasCreatableOptionName(normalizedQuery, options)
}

export const dedupeCreatableOptionsByName = <T extends CreatableNamedOption>(options: T[]) => {
  const seenNames = new Set<string>()
  const deduped: T[] = []

  for (const option of options) {
    const normalizedName = normalizeCreatableOptionName(option.name)
    if (normalizedName && seenNames.has(normalizedName)) {
      continue
    }
    if (normalizedName) {
      seenNames.add(normalizedName)
    }
    deduped.push(option)
  }

  return deduped
}

const mergeSelectedOptions = <T extends CreatableNamedOption>(
  results: T[],
  selectedValues: Array<string | number>,
  currentOptions: T[],
) => {
  const selectedIds = new Set(
    selectedValues
      .map((item) => normalizeOptionId(item))
      .filter((item): item is number => item !== null),
  )
  const selectedItems = currentOptions.filter((item) => selectedIds.has(item.id))

  const merged = [...results]
  for (const selectedItem of selectedItems) {
    if (!merged.find((item) => item.id === selectedItem.id)) {
      merged.push(selectedItem)
    }
  }

  return dedupeCreatableOptionsByName(merged)
}

export const searchCreatableOptions = async <T extends CreatableNamedOption>(params: {
  query: string
  selectedValues: Array<string | number>
  currentOptions: T[]
  search: (query: string) => Promise<T[]>
}) => {
  const results = await params.search(params.query)
  return mergeSelectedOptions(results, params.selectedValues, params.currentOptions)
}

const defaultFindExisting = <T extends CreatableNamedOption>(name: string, options: T[]) => {
  const normalizedName = normalizeCreatableOptionName(name)
  return options.find((item) => normalizeCreatableOptionName(item.name) === normalizedName)
}

export const resolveCreatableSelections = async <T extends CreatableNamedOption>(params: {
  values: Array<string | number>
  options: T[]
  createItem: (name: string) => Promise<T>
  findExisting?: (name: string, options: T[]) => T | undefined
}) => {
  const ids: number[] = []
  const nextOptions = [...params.options]
  const findExisting = params.findExisting || defaultFindExisting<T>

  for (const value of params.values) {
    const normalizedId = normalizeOptionId(value)
    if (normalizedId !== null) {
      ids.push(normalizedId)
      continue
    }

    if (typeof value !== 'string' || !value.trim()) {
      continue
    }

    const name = value.trim()
    const existing = findExisting(name, nextOptions)
    if (existing) {
      ids.push(existing.id)
      continue
    }

    const created = await params.createItem(name)
    nextOptions.push(created)
    ids.push(created.id)
  }

  return {
    ids: Array.from(new Set(ids)),
    options: nextOptions,
  }
}
