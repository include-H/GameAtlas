interface CreatableNamedOption {
  id: number
  name: string
}

export const normalizeOptionId = (value: unknown): number | null => {
  if (typeof value === 'number' && !Number.isNaN(value)) return value
  return null
}

export const sortCreatableOptionsByName = <T extends CreatableNamedOption>(options: T[]) => {
  return [...options].sort((a, b) => a.name.localeCompare(b.name, 'zh-Hans-CN'))
}

const mergeSelectedOptions = <T extends { id: number }>(
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

  return merged
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
  return options.find((item) => item.name.trim().toLowerCase() === name.toLowerCase())
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
