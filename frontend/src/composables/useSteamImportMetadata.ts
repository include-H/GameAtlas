import { ref, watch, type Ref } from 'vue'
import type { EditGameForm } from '@/composables/edit-game-form'
import steamService from '@/services/steam.service'
import { useSteamPicker } from '@/composables/useSteamPicker'
import type { SteamGameDetails, SteamGameSearchResult } from '@/services/types'
import { extractWikiMetadata, type WikiMetadataExtraction } from '@/utils/wiki-metadata-parser'

type AlertType = 'success' | 'warning' | 'error'

export interface WikiMetadataCandidateSelection {
  key: string
  label: string
  value: string
  selected: boolean
  group?: 'title_alt'
}

interface UseSteamImportMetadataOptions {
  form: Ref<Pick<EditGameForm, 'summary' | 'title' | 'title_alt' | 'release_date' | 'engine' | 'developer_ids' | 'publisher_ids' | 'platform_ids'>>
  getWikiContent: () => string
  addAlert: (message: string, type: AlertType) => void
}

const splitTitleAltValues = (value: string) => {
  return value
    .split(/\s*\/\s*/g)
    .map((item) => item.trim())
    .filter(Boolean)
}

const stripHtmlToText = (html: string) => {
  if (!html.trim()) return ''

  if (typeof window !== 'undefined' && typeof DOMParser !== 'undefined') {
    const doc = new DOMParser().parseFromString(html, 'text/html')
    return (doc.body.textContent || '')
      .replace(/\u00a0/g, ' ')
      .replace(/\s+\n/g, '\n')
      .replace(/\n{3,}/g, '\n\n')
      .trim()
  }

  return html
    .replace(/<br\s*\/?>/gi, '\n')
    .replace(/<\/p>/gi, '\n\n')
    .replace(/<[^>]+>/g, ' ')
    .replace(/&nbsp;/gi, ' ')
    .replace(/\s+/g, ' ')
    .trim()
}

export const useSteamImportMetadata = (options: UseSteamImportMetadataOptions) => {
  const showSummarySelector = ref(false)
  const steamSummaryPreview = ref('')
  const steamSummaryDetails = ref<SteamGameDetails | null>(null)
  const isPreparingWikiMetadataCandidates = ref(false)
  const isApplyingWikiMetadata = ref(false)
  const wikiMetadataPickerVisible = ref(false)
  const wikiMetadataCandidates = ref<WikiMetadataCandidateSelection[]>([])
  const wikiMetadataSnapshot = ref<WikiMetadataExtraction | null>(null)

  const pickSteamSearchQuery = () => {
    const preferred = options.form.value.title_alt?.trim()
    if (preferred) return preferred
    return options.form.value.title?.trim() || ''
  }

  const applySteamMetadataToForm = (details: SteamGameDetails) => {
    if (details.releaseDate) {
      options.form.value.release_date = details.releaseDate
    }
    if (details.developers && details.developers.length > 0) {
      // 2026-04-04: Steam import writes names into the form first because the edit submit flow is
      // responsible for resolving new metadata into persistent ids.
      const merged = new Set<string | number>(options.form.value.developer_ids)
      for (const name of details.developers) {
        if (name.trim()) merged.add(name.trim())
      }
      options.form.value.developer_ids = Array.from(merged)
    }
    if (details.publishers && details.publishers.length > 0) {
      const merged = new Set<string | number>(options.form.value.publisher_ids)
      for (const name of details.publishers) {
        if (name.trim()) merged.add(name.trim())
      }
      options.form.value.publisher_ids = Array.from(merged)
    }
  }

  const prepareWikiMetadataCandidates = () => {
    const metadata = extractWikiMetadata(options.getWikiContent())
    const candidates: WikiMetadataCandidateSelection[] = []

    if (metadata.summary) {
      candidates.push({
        key: 'summary',
        label: '简介',
        value: metadata.summary,
        selected: true,
      })
    }

    const englishTitleAlts = splitTitleAltValues(metadata.englishTitleAlt)
    const chineseTitleAlts = splitTitleAltValues(metadata.chineseTitleAlt)

    englishTitleAlts.forEach((value, index) => {
      candidates.push({
        key: `title_alt_en:${index}`,
        label: '英文名',
        value,
        selected: index === 0,
        group: 'title_alt',
      })
    })

    chineseTitleAlts.forEach((value, index) => {
      candidates.push({
        key: `title_alt_cn:${index}`,
        label: '别名',
        value,
        selected: englishTitleAlts.length === 0 && index === 0,
        group: 'title_alt',
      })
    })

    if (metadata.releaseDate) {
      candidates.push({
        key: 'release_date',
        label: '发行日期',
        value: metadata.releaseDate,
        selected: true,
      })
    }
    if (metadata.engine) {
      candidates.push({
        key: 'engine',
        label: '游戏引擎',
        value: metadata.engine,
        selected: true,
      })
    }
    if (metadata.developers.length > 0) {
      candidates.push({
        key: 'developers',
        label: '开发商',
        value: metadata.developers.join(' / '),
        selected: true,
      })
    }
    if (metadata.publishers.length > 0) {
      candidates.push({
        key: 'publishers',
        label: '发行商',
        value: metadata.publishers.join(' / '),
        selected: true,
      })
    }
    if (metadata.platforms.length > 0) {
      candidates.push({
        key: 'platforms',
        label: '平台',
        value: metadata.platforms.join(' / '),
        selected: true,
      })
    }
    if (candidates.length === 0) {
      options.addAlert('当前 Wiki 没有可提取的信息', 'warning')
      return
    }

    wikiMetadataSnapshot.value = metadata
    wikiMetadataCandidates.value = candidates
    wikiMetadataPickerVisible.value = true
  }

  const importMetadataFromWiki = () => {
    const content = options.getWikiContent().trim()
    if (!content) {
      options.addAlert('当前游戏没有可解析的 Wiki 内容', 'warning')
      return
    }

    isPreparingWikiMetadataCandidates.value = true
    try {
      prepareWikiMetadataCandidates()
    } catch (error) {
      console.error('Failed to extract wiki metadata:', error)
      options.addAlert('从 Wiki 提取元数据失败', 'warning')
    } finally {
      isPreparingWikiMetadataCandidates.value = false
    }
  }

  const handleWikiMetadataCandidateSelectionChange = (key: string, selected: boolean) => {
    wikiMetadataCandidates.value = wikiMetadataCandidates.value.map((item) =>
      item.key === key
        ? {
            ...item,
            selected,
          }
        : selected && item.group && item.group === wikiMetadataCandidates.value.find((candidate) => candidate.key === key)?.group
          ? {
              ...item,
              selected: false,
            }
          : item,
    )
  }

  const applySelectedWikiMetadata = () => {
    const selected = wikiMetadataCandidates.value.filter((item) => item.selected)
    if (selected.length === 0) {
      options.addAlert('还没有选择要应用的字段', 'warning')
      return
    }

    isApplyingWikiMetadata.value = true

    try {
      const metadata = wikiMetadataSnapshot.value
      if (!metadata) {
        options.addAlert('当前没有可应用的 Wiki 提取结果', 'warning')
        return
      }
      const appliedLabels: string[] = []

      for (const item of selected) {
        if (item.key.startsWith('title_alt_en:')) {
          if (item.value) {
            options.form.value.title_alt = item.value
            appliedLabels.push('英文名')
          }
          continue
        }

        if (item.key.startsWith('title_alt_cn:')) {
          if (item.value) {
            options.form.value.title_alt = item.value
            appliedLabels.push('别名')
          }
          continue
        }

        switch (item.key) {
          case 'summary':
            if (metadata.summary) {
              options.form.value.summary = metadata.summary
              appliedLabels.push('简介')
            }
            break
          case 'engine':
            if (metadata.engine) {
              options.form.value.engine = metadata.engine
              appliedLabels.push('游戏引擎')
            }
            break
          case 'release_date':
            if (metadata.releaseDate) {
              options.form.value.release_date = metadata.releaseDate
              appliedLabels.push('发行日期')
            }
            break
          case 'developers':
            if (metadata.developers.length > 0) {
              const merged = new Set<string | number>(options.form.value.developer_ids)
              for (const name of metadata.developers) {
                merged.add(name)
              }
              options.form.value.developer_ids = Array.from(merged)
              appliedLabels.push('开发商')
            }
            break
          case 'publishers':
            if (metadata.publishers.length > 0) {
              const merged = new Set<string | number>(options.form.value.publisher_ids)
              for (const name of metadata.publishers) {
                merged.add(name)
              }
              options.form.value.publisher_ids = Array.from(merged)
              appliedLabels.push('发行商')
            }
            break
          case 'platforms':
            if (metadata.platforms.length > 0) {
              const merged = new Set<string | number>(options.form.value.platform_ids)
              for (const name of metadata.platforms) {
                merged.add(name)
              }
              options.form.value.platform_ids = Array.from(merged)
              appliedLabels.push('平台')
            }
            break
        }
      }

      wikiMetadataPickerVisible.value = false

      if (appliedLabels.length === 0) {
        options.addAlert('已选择字段，但没有成功应用到表单', 'warning')
        return
      }

      options.addAlert(`已应用 Wiki 字段：${appliedLabels.join('；')}`, 'success')
    } finally {
      isApplyingWikiMetadata.value = false
    }
  }

  const summarySteamPicker = useSteamPicker<SteamGameDetails>({
    onSelect: async (game) => {
      const details = await steamService.getGameDetails(game.id)
      steamSummaryDetails.value = details
      steamSummaryPreview.value = stripHtmlToText(details.description || '')
      return details
    },
    onError: (message) => {
      options.addAlert('Steam 简介处理失败：' + message, 'error')
    },
  })

  const steamSummarySearchQuery = summarySteamPicker.query
  const steamSummarySearchResults = summarySteamPicker.results
  const selectedSteamSummaryGame = summarySteamPicker.selectedGame
  const isSearchingSteamSummary = summarySteamPicker.isSearching

  const handleSummarySearchClear = () => {
    summarySteamPicker.clear()
    steamSummaryPreview.value = ''
    steamSummaryDetails.value = null
    wikiMetadataPickerVisible.value = false
    wikiMetadataCandidates.value = []
  }

  const searchSteamForSummary = async () => {
    steamSummaryPreview.value = ''
    steamSummaryDetails.value = null
    await summarySteamPicker.search()
  }

  const selectSteamSummaryGame = async (game: SteamGameSearchResult) => {
    steamSummaryPreview.value = ''
    steamSummaryDetails.value = null
    await summarySteamPicker.select(game)
  }

  const backToSummarySearch = () => {
    summarySteamPicker.back()
    steamSummaryPreview.value = ''
    steamSummaryDetails.value = null
  }

  const confirmSummaryImport = () => {
    const details = steamSummaryDetails.value
    const hasImportableMetadata = !!details?.releaseDate || !!details?.developers?.[0] || !!details?.publishers?.[0]
    if (!steamSummaryPreview.value && !hasImportableMetadata) {
      options.addAlert('当前没有可导入的 Steam 信息', 'warning')
      return
    }

    if (steamSummaryPreview.value) {
      options.form.value.summary = steamSummaryPreview.value
    }
    if (details) {
      applySteamMetadataToForm(details)
    }
    showSummarySelector.value = false
    options.addAlert(
      `已导入 Steam 信息：${selectedSteamSummaryGame.value?.name || 'Steam 游戏'}`,
      'success',
    )
  }

  watch(showSummarySelector, (isOpen) => {
    if (!isOpen) return
    const query = pickSteamSearchQuery()
    if (!query) return
    steamSummarySearchQuery.value = query
    void searchSteamForSummary()
  })

  const resetMetadataImportState = () => {
    showSummarySelector.value = false
    steamSummarySearchQuery.value = ''
    steamSummarySearchResults.value = []
    selectedSteamSummaryGame.value = null
    steamSummaryPreview.value = ''
    steamSummaryDetails.value = null
    wikiMetadataPickerVisible.value = false
    wikiMetadataCandidates.value = []
    wikiMetadataSnapshot.value = null
  }

  return {
    applySelectedWikiMetadata,
    backToSummarySearch,
    confirmSummaryImport,
    handleSummarySearchClear,
    handleWikiMetadataCandidateSelectionChange,
    importMetadataFromWiki,
    isApplyingWikiMetadata,
    isPreparingWikiMetadataCandidates,
    isSearchingSteamSummary,
    resetMetadataImportState,
    searchSteamForSummary,
    selectSteamSummaryGame,
    selectedSteamSummaryGame,
    showSummarySelector,
    steamSummaryPreview,
    steamSummarySearchQuery,
    steamSummarySearchResults,
    wikiMetadataCandidates,
    wikiMetadataPickerVisible,
  }
}
