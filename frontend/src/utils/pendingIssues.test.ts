import { describe, expect, it } from 'vitest'

import { evaluatePendingIssues, getPendingIssueDetailLabel, getPendingIssueLabel } from './pendingIssues'

describe('pendingIssues helpers', () => {
  it('returns fallback labels for unknown keys', () => {
    expect(getPendingIssueLabel('unknown')).toBe('待处理')
    expect(getPendingIssueDetailLabel('unknown')).toBe('待补充')
  })

  it('evaluates missing groups and details from list data', () => {
    const result = evaluatePendingIssues({
      id: 1,
      public_id: 'game-1',
      title: 'Test Game',
      title_alt: null,
      visibility: 'public',
      cover_image: '',
      banner_image: '',
      release_date: null,
      engine: null,
      primary_screenshot: '',
      screenshot_count: 0,
      file_count: 0,
      developer_count: 0,
      publisher_count: 1,
      platform_count: 0,
      downloads: 0,
      needs_review: false,
      summary: '   ',
      wiki_content: '',
      wiki_content_html: '<p></p>',
      created_at: '2026-03-25T00:00:00Z',
      updated_at: '2026-03-25T00:00:00Z',
      isFavorite: false,
    })

    expect(result.details).toEqual([
      'missing-cover',
      'missing-banner',
      'missing-screenshots',
      'missing-wiki-content',
      'missing-files-list',
      'missing-developer',
      'missing-platform',
      'missing-summary',
    ])
    expect(result.groups).toEqual([
      'missing-assets',
      'missing-wiki',
      'missing-files',
      'missing-metadata',
    ])
    expect(result.ignoredDetails).toEqual([])
  })

  it('filters ignored details and keeps the rest visible', () => {
    const result = evaluatePendingIssues(
      {
        id: 2,
        public_id: 'game-2',
        title: 'Half Ready Game',
        title_alt: null,
        visibility: 'public',
        cover_image: '',
        banner_image: null,
        release_date: null,
        engine: null,
        primary_screenshot: '',
        screenshot_count: 0,
        file_count: 0,
        developer_count: 0,
        publisher_count: 0,
        platform_count: 0,
        downloads: 0,
        needs_review: false,
        summary: '',
        wiki_content: '',
        wiki_content_html: '',
        created_at: '2026-03-25T00:00:00Z',
        updated_at: '2026-03-25T00:00:00Z',
        isFavorite: false,
      },
      ['missing-cover', 'missing-files-list'],
    )

    expect(result.details).not.toContain('missing-cover')
    expect(result.details).not.toContain('missing-files-list')
    expect(result.ignoredDetails).toEqual(['missing-cover', 'missing-files-list'])
    expect(result.groups).toContain('missing-assets')
    expect(result.groups).toContain('missing-metadata')
  })
})
