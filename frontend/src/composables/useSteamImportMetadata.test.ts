import { ref } from 'vue'
import { describe, expect, it, vi } from 'vitest'

import { useSteamImportMetadata } from './useSteamImportMetadata'

describe('useSteamImportMetadata', () => {
  it('applies the prepared wiki metadata snapshot instead of reparsing mutated content', () => {
    const form = ref({
      summary: '',
      title: 'Game One',
      title_alt: '',
      release_date: undefined as string | undefined,
      engine: '',
      developer_ids: [] as Array<string | number>,
      publisher_ids: [] as Array<string | number>,
      platform_ids: [] as Array<string | number>,
    })
    const wikiContent = ref(`
- 简介：First summary
- 英文常见译名：First Alt
`)
    const addAlert = vi.fn()

    const metadataImport = useSteamImportMetadata({
      form,
      getWikiContent: () => wikiContent.value,
      addAlert,
    })

    metadataImport.importMetadataFromWiki()
    wikiContent.value = `
- 简介：Second summary
- 英文常见译名：Second Alt
`

    metadataImport.applySelectedWikiMetadata()

    expect(form.value.summary).toBe('First summary')
    expect(form.value.title_alt).toBe('First Alt')
    expect(addAlert).toHaveBeenCalledWith('已应用 Wiki 字段：简介；英文名', 'success')
  })
})
