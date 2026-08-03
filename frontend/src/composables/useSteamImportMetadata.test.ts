import { ref } from 'vue'
import { describe, expect, it, vi } from 'vitest'

import { useSteamImportMetadata } from './useSteamImportMetadata'

describe('useSteamImportMetadata', () => {
  it('treats whitespace-only wiki text as unavailable content', () => {
    const form = ref({
      summary: '',
      title: 'Game One',
      title_alt: '',
      release_date: undefined as string | undefined,
      engine: '',
      developer_ids: [] as number[],
      publisher_ids: [] as number[],
      platform_ids: [] as Array<string | number>,
    })
    const addAlert = vi.fn()

    const metadataImport = useSteamImportMetadata({
      form,
      getWikiContent: () => '   ',
      ensureDeveloperNames: vi.fn(),
      ensurePublisherNames: vi.fn(),
      addAlert,
    })

    metadataImport.importMetadataFromWiki()

    expect(metadataImport.wikiMetadataPickerVisible.value).toBe(false)
    expect(addAlert).toHaveBeenCalledWith('当前游戏没有可解析的 Wiki 内容', 'warning')
  })

  it('applies the prepared wiki metadata snapshot instead of reparsing mutated content', async () => {
    const form = ref({
      summary: '',
      title: 'Game One',
      title_alt: '',
      release_date: undefined as string | undefined,
      engine: '',
      developer_ids: [] as number[],
      publisher_ids: [] as number[],
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
      ensureDeveloperNames: vi.fn(),
      ensurePublisherNames: vi.fn(),
      addAlert,
    })

    metadataImport.importMetadataFromWiki()
    wikiContent.value = `
- 简介：Second summary
- 英文常见译名：Second Alt
`

    await metadataImport.applySelectedWikiMetadata()

    expect(form.value.summary).toBe('First summary')
    expect(form.value.title_alt).toBe('First Alt')
    expect(addAlert).toHaveBeenCalledWith('已应用 Wiki 字段：简介；英文名', 'success')
  })

  it('writes resolved wiki developer and publisher ids into the form', async () => {
    const form = ref({
      summary: '',
      title: 'Game One',
      title_alt: '',
      release_date: undefined as string | undefined,
      engine: '',
      developer_ids: [1] as number[],
      publisher_ids: [2] as number[],
      platform_ids: [] as Array<string | number>,
    })
    const ensureDeveloperNames = vi.fn().mockResolvedValue([3, 4])
    const ensurePublisherNames = vi.fn().mockResolvedValue([5])
    const metadataImport = useSteamImportMetadata({
      form,
      getWikiContent: () => `
- 开发商：开发商 A、开发商 B
- 发行商：发行商 A
`,
      ensureDeveloperNames,
      ensurePublisherNames,
      addAlert: vi.fn(),
    })

    metadataImport.importMetadataFromWiki()
    await metadataImport.applySelectedWikiMetadata()

    expect(ensureDeveloperNames).toHaveBeenCalledWith(['开发商 A', '开发商 B'])
    expect(ensurePublisherNames).toHaveBeenCalledWith(['发行商 A'])
    expect(form.value.developer_ids).toEqual([1, 3, 4])
    expect(form.value.publisher_ids).toEqual([2, 5])
  })
})
