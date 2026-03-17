import type { Game } from '@/services/types'

export type PendingIssueKey =
  | 'missing-assets'
  | 'missing-wiki'
  | 'missing-files'
  | 'missing-metadata'

export interface PendingIssueDefinition {
  key: PendingIssueKey
  label: string
  description: string
}

export const pendingIssueDefinitions: PendingIssueDefinition[] = [
  {
    key: 'missing-assets',
    label: '缺少图片',
    description: '封面、横幅或截图未补齐',
  },
  {
    key: 'missing-wiki',
    label: '缺少 Wiki',
    description: '还没有游戏介绍内容',
  },
  {
    key: 'missing-files',
    label: '缺少文件',
    description: '还没有可下载文件条目',
  },
  {
    key: 'missing-metadata',
    label: '基础信息不完整',
    description: '开发商、发行商、平台或简介缺失',
  },
]

export function getPendingIssueLabel(key?: string | null) {
  return pendingIssueDefinitions.find((item) => item.key === key)?.label || '待处理'
}

export function getPendingIssues(game: Game): PendingIssueKey[] {
  const issues: PendingIssueKey[] = []

  const hasCover = !!game.cover_image
  const hasBanner = !!game.banner_image
  const hasScreenshots = !!game.screenshots && game.screenshots.length > 0
  if (!hasCover || !hasBanner || !hasScreenshots) {
    issues.push('missing-assets')
  }

  const hasWiki = !!game.wiki_content?.trim() || !!game.wiki_content_html?.trim()
  if (!hasWiki) {
    issues.push('missing-wiki')
  }

  const hasFiles =
    (!!game.files && game.files.length > 0) ||
    (!!game.file_paths && game.file_paths.length > 0) ||
    !!game.file_path
  if (!hasFiles) {
    issues.push('missing-files')
  }

  const hasDeveloper = !!game.developers && game.developers.length > 0
  const hasPublisher = !!game.publishers && game.publishers.length > 0
  const hasPlatform = (!!game.platforms && game.platforms.length > 0) || !!game.platform
  const hasSummary = !!game.summary?.trim()
  if (!hasDeveloper || !hasPublisher || !hasPlatform || !hasSummary) {
    issues.push('missing-metadata')
  }

  return issues
}

export function matchesPendingIssue(game: Game, key?: string | null) {
  if (!key) return true
  return getPendingIssues(game).includes(key as PendingIssueKey)
}
