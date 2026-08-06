import { computed, type Ref } from 'vue'
import {
  normalizeMetadataPickerID,
  normalizeMetadataPickerIDs,
  useRemoteMetadataPicker,
} from '@/composables/useRemoteMetadataPicker'
import { sortCreatableOptionsByName } from '@/utils/creatable-select'
import { seriesService } from '@/services/series.service'
import { developersService } from '@/services/developers.service'
import { publishersService } from '@/services/publishers.service'
import type { EditGameForm } from '@/utils/edit-game-form'
import type { Developer, Publisher, Series } from '@/services/types'

type AlertType = 'success' | 'warning' | 'error'

interface UseEditGameMetadataPickersOptions {
  form: Ref<Pick<EditGameForm, 'series_id' | 'developer_ids' | 'publisher_ids'>>
  addAlert: (message: string, type: AlertType) => void
}

const CREATE_SERIES_OPTION_VALUE = '__create_series__'
const CREATE_DEVELOPER_OPTION_VALUE = '__create_developer__'
const CREATE_PUBLISHER_OPTION_VALUE = '__create_publisher__'

export const useEditGameMetadataPickers = (options: UseEditGameMetadataPickersOptions) => {
  const seriesPicker = useRemoteMetadataPicker<Series>({
    selectedIds: () => options.form.value.series_id === null ? [] : [options.form.value.series_id],
    search: (query) => seriesService.searchSeries(query),
    create: (name) => seriesService.createSeries({ name }),
  })
  const developerPicker = useRemoteMetadataPicker<Developer>({
    selectedIds: () => options.form.value.developer_ids,
    search: (query) => developersService.listDevelopers({ query }),
    create: (name) => developersService.createDeveloper({ name }),
  })
  const publisherPicker = useRemoteMetadataPicker<Publisher>({
    selectedIds: () => options.form.value.publisher_ids,
    search: (query) => publishersService.listPublishers({ query }),
    create: (name) => publishersService.createPublisher({ name }),
  })

  const seriesOptions = seriesPicker.options
  const developerOptions = developerPicker.options
  const publisherOptions = publisherPicker.options
  const isSearchingSeries = seriesPicker.isSearching
  const isSearchingDevelopers = developerPicker.isSearching
  const isSearchingPublishers = publisherPicker.isSearching
  const isCreatingSeries = seriesPicker.isCreating
  const isCreatingDevelopers = developerPicker.isCreating
  const isCreatingPublishers = publisherPicker.isCreating
  const canCreateSeriesOption = seriesPicker.canCreate
  const canCreateDeveloperOption = developerPicker.canCreate
  const canCreatePublisherOption = publisherPicker.canCreate
  const seriesSearchQuery = seriesPicker.query
  const developerSearchQuery = developerPicker.query
  const publisherSearchQuery = publisherPicker.query

  const filteredSeriesOptions = computed(() => {
    // 2026-04-04: keep authoring pickers alphabetized for scan speed.
    // Impact: this is UI-only option ordering; do not treat it as backend metadata sort semantics.
    return sortCreatableOptionsByName(seriesOptions.value)
  })

  const filteredDeveloperOptions = computed(() => {
    // 2026-04-04: keep authoring pickers alphabetized for scan speed.
    // Impact: this is UI-only option ordering; do not treat it as backend metadata sort semantics.
    return sortCreatableOptionsByName(developerOptions.value)
  })

  const filteredPublisherOptions = computed(() => {
    // 2026-04-04: keep authoring pickers alphabetized for scan speed.
    // Impact: this is UI-only option ordering; do not treat it as backend metadata sort semantics.
    return sortCreatableOptionsByName(publisherOptions.value)
  })

  const handleSeriesSearch = async (query: string) => {
    try {
      await seriesPicker.search(query)
    } catch {
      options.addAlert('系列搜索失败', 'error')
    }
  }

  const handleDeveloperSearch = async (query: string) => {
    try {
      await developerPicker.search(query)
    } catch {
      options.addAlert('开发商搜索失败', 'error')
    }
  }

  const handlePublisherSearch = async (query: string) => {
    try {
      await publisherPicker.search(query)
    } catch {
      options.addAlert('发行商搜索失败', 'error')
    }
  }

  const createSeriesFromSearch = async () => {
    try {
      const item = await seriesPicker.createFromQuery()
      if (!item) return
      options.form.value.series_id = item.id
      seriesPicker.clearSearch()
    } catch {
      options.addAlert('创建系列失败', 'error')
    }
  }

  const handleSeriesEnter = async () => {
    try {
      const item = await seriesPicker.resolveQuery()
      if (!item) return
      options.form.value.series_id = item.id
      seriesPicker.clearSearch()
    } catch {
      options.addAlert('选择或创建系列失败', 'error')
    }
  }

  const createDeveloperFromSearch = async () => {
    try {
      const item = await developerPicker.createFromQuery()
      if (!item) return
      options.form.value.developer_ids = Array.from(new Set([...options.form.value.developer_ids, item.id]))
      developerPicker.clearSearch()
    } catch {
      options.addAlert('创建开发商失败', 'error')
    }
  }

  const createPublisherFromSearch = async () => {
    try {
      const item = await publisherPicker.createFromQuery()
      if (!item) return
      options.form.value.publisher_ids = Array.from(new Set([...options.form.value.publisher_ids, item.id]))
      publisherPicker.clearSearch()
    } catch {
      options.addAlert('创建发行商失败', 'error')
    }
  }

  const handleSeriesSelection = (value: unknown) => {
    if (value === CREATE_SERIES_OPTION_VALUE) {
      void createSeriesFromSearch()
      return
    }
    options.form.value.series_id = normalizeMetadataPickerID(value)
    seriesPicker.clearSearch()
  }

  const handleDeveloperSelection = (value: unknown) => {
    const values = Array.isArray(value) ? value : []
    options.form.value.developer_ids = normalizeMetadataPickerIDs(values)
    if (values.includes(CREATE_DEVELOPER_OPTION_VALUE)) {
      void createDeveloperFromSearch()
    }
  }

  const handlePublisherSelection = (value: unknown) => {
    const values = Array.isArray(value) ? value : []
    options.form.value.publisher_ids = normalizeMetadataPickerIDs(values)
    if (values.includes(CREATE_PUBLISHER_OPTION_VALUE)) {
      void createPublisherFromSearch()
    }
  }

  return {
    CREATE_DEVELOPER_OPTION_VALUE,
    CREATE_PUBLISHER_OPTION_VALUE,
    CREATE_SERIES_OPTION_VALUE,
    seriesPicker,
    developerPicker,
    publisherPicker,
    seriesOptions,
    developerOptions,
    publisherOptions,
    isSearchingSeries,
    isSearchingDevelopers,
    isSearchingPublishers,
    isCreatingSeries,
    isCreatingDevelopers,
    isCreatingPublishers,
    canCreateSeriesOption,
    canCreateDeveloperOption,
    canCreatePublisherOption,
    seriesSearchQuery,
    developerSearchQuery,
    publisherSearchQuery,
    filteredSeriesOptions,
    filteredDeveloperOptions,
    filteredPublisherOptions,
    handleSeriesSearch,
    handleDeveloperSearch,
    handlePublisherSearch,
    handleSeriesEnter,
    handleSeriesSelection,
    handleDeveloperSelection,
    handlePublisherSelection,
  }
}
