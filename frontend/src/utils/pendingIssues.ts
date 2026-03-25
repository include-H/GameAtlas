import type { GameDetail, GameListItem } from '@/services/types'

export type PendingIssueKey =
  | 'missing-assets'
  | 'missing-wiki'
  | 'missing-files'
  | 'missing-metadata'

export type PendingIssueDetailKey =
  | 'missing-cover'
  | 'missing-banner'
  | 'missing-screenshots'
  | 'missing-wiki-content'
  | 'missing-files-list'
  | 'missing-developer'
  | 'missing-publisher'
  | 'missing-platform'
  | 'missing-summary'

interface PendingIssueDefinition {
  key: PendingIssueKey
  label: string
  description: string
}

interface PendingIssueDetailDefinition {
  key: PendingIssueDetailKey
  label: string
  group: PendingIssueKey
}

export interface PendingIssueEvaluation {
  groups: PendingIssueKey[]
  details: PendingIssueDetailKey[]
  ignoredDetails: PendingIssueDetailKey[]
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

export const pendingIssueDetailDefinitions: PendingIssueDetailDefinition[] = [
  {
    key: 'missing-cover',
    label: '缺封面',
    group: 'missing-assets',
  },
  {
    key: 'missing-banner',
    label: '缺横幅',
    group: 'missing-assets',
  },
  {
    key: 'missing-screenshots',
    label: '缺截图',
    group: 'missing-assets',
  },
  {
    key: 'missing-wiki-content',
    label: '缺 Wiki 内容',
    group: 'missing-wiki',
  },
  {
    key: 'missing-files-list',
    label: '缺下载文件',
    group: 'missing-files',
  },
  {
    key: 'missing-developer',
    label: '缺开发商',
    group: 'missing-metadata',
  },
  {
    key: 'missing-publisher',
    label: '缺发行商',
    group: 'missing-metadata',
  },
  {
    key: 'missing-platform',
    label: '缺平台',
    group: 'missing-metadata',
  },
  {
    key: 'missing-summary',
    label: '缺简介',
    group: 'missing-metadata',
  },
]

const pendingIssueDetailGroupMap = pendingIssueDetailDefinitions.reduce<Record<PendingIssueDetailKey, PendingIssueKey>>(
  (acc, item) => {
    acc[item.key] = item.group
    return acc
  },
  {} as Record<PendingIssueDetailKey, PendingIssueKey>,
)

export function getPendingIssueLabel(key?: string | null) {
  return pendingIssueDefinitions.find((item) => item.key === key)?.label || '待处理'
}

export function getPendingIssueDetailLabel(key?: string | null) {
  return pendingIssueDetailDefinitions.find((item) => item.key === key)?.label || '待补充'
}

type PendingGame = GameListItem | GameDetail

const hasDetailPayload = (game: PendingGame): game is GameDetail => {
  return 'screenshots' in game
}

const hasListCounts = (game: PendingGame): game is GameListItem => {
  return 'screenshot_count' in game
}

function hasMeaningfulWikiContent(content?: string | null, html?: string | null) {
  if (content?.trim()) {
    return true
  }

  if (!html?.trim()) {
    return false
  }

  const plainText = html
    .replace(/<style[\s\S]*?<\/style>/gi, ' ')
    .replace(/<script[\s\S]*?<\/script>/gi, ' ')
    .replace(/<[^>]+>/g, ' ')
    .replace(/&nbsp;/gi, ' ')
    .trim()

  return plainText.length > 0
}

export function evaluatePendingIssues(game: PendingGame, ignoredDetails: PendingIssueDetailKey[] = []): PendingIssueEvaluation {
  const details: PendingIssueDetailKey[] = []
  const ignoredSet = new Set(ignoredDetails)

  const hasCover = !!game.cover_image
  const hasBanner = !!game.banner_image
  const hasScreenshots =
    (hasListCounts(game) && game.screenshot_count > 0) ||
    (hasDetailPayload(game) && game.screenshots.length > 0) ||
    (hasListCounts(game) && !!game.primary_screenshot)
  if (!hasCover) {
    details.push('missing-cover')
  }
  if (!hasBanner) {
    details.push('missing-banner')
  }
  if (!hasScreenshots) {
    details.push('missing-screenshots')
  }
  const hasWiki = hasMeaningfulWikiContent(game.wiki_content, game.wiki_content_html)
  if (!hasWiki) {
    details.push('missing-wiki-content')
  }

  const hasFiles =
    (hasDetailPayload(game) && game.files.length > 0) ||
    (hasListCounts(game) && game.file_count > 0)
  if (!hasFiles) {
    details.push('missing-files-list')
  }

  const hasDeveloper =
    (hasDetailPayload(game) && game.developers.length > 0) ||
    (hasListCounts(game) && game.developer_count > 0)
  const hasPublisher =
    (hasDetailPayload(game) && game.publishers.length > 0) ||
    (hasListCounts(game) && game.publisher_count > 0)
  const hasPlatform =
    (hasDetailPayload(game) && game.platforms.length > 0) ||
    (hasListCounts(game) && game.platform_count > 0)
  const hasSummary = !!game.summary?.trim()
  if (!hasDeveloper) {
    details.push('missing-developer')
  }
  if (!hasPublisher) {
    details.push('missing-publisher')
  }
  if (!hasPlatform) {
    details.push('missing-platform')
  }
  if (!hasSummary) {
    details.push('missing-summary')
  }

  const visibleDetails = details.filter((detail) => !ignoredSet.has(detail))
  const groups = Array.from(new Set(
    visibleDetails
      .map((detail) => pendingIssueDetailGroupMap[detail])
      .filter((group): group is PendingIssueKey => Boolean(group)),
  ))

  return {
    groups,
    details: visibleDetails,
    ignoredDetails: details.filter((detail) => ignoredSet.has(detail)),
  }
}

export function isSeverePendingEvaluation(evaluation: PendingIssueEvaluation) {
  return (
    evaluation.details.length >= 3 ||
    evaluation.groups.includes('missing-files') ||
    (evaluation.groups.includes('missing-assets') && evaluation.groups.includes('missing-wiki'))
  )
}

export function getPendingIssues(game: PendingGame, ignoredDetails: PendingIssueDetailKey[] = []): PendingIssueKey[] {
  return evaluatePendingIssues(game, ignoredDetails).groups
}

export function getPendingIssueDetails(game: PendingGame, ignoredDetails: PendingIssueDetailKey[] = []): PendingIssueDetailKey[] {
  return evaluatePendingIssues(game, ignoredDetails).details
}

export function getIgnoredPendingIssueDetails(game: PendingGame, ignoredDetails: PendingIssueDetailKey[] = []) {
  return evaluatePendingIssues(game, ignoredDetails).ignoredDetails
}

export function isSeverePendingGame(game: PendingGame, ignoredDetails: PendingIssueDetailKey[] = []) {
  return isSeverePendingEvaluation(evaluatePendingIssues(game, ignoredDetails))
}

export function matchesPendingIssueEvaluation(evaluation: PendingIssueEvaluation, key?: string | null) {
  if (!key) return true
  return evaluation.groups.includes(key as PendingIssueKey)
}

export function matchesPendingIssue(game: PendingGame, key?: string | null) {
  return matchesPendingIssueEvaluation(evaluatePendingIssues(game), key)
}
