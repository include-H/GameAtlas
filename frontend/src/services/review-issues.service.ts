import { del, get, put } from './api'
import type { ApiEnvelope, ReviewIssueOverride } from './types'

const reviewIssuesService = {
  async list(gameIds?: Array<number | string>): Promise<ReviewIssueOverride[]> {
    const params = gameIds && gameIds.length > 0
      ? { game_ids: gameIds.map(String).join(',') }
      : undefined
    const response = await get<ApiEnvelope<ReviewIssueOverride[]>>('/review-issue-overrides', { params })
    return response.data
  },

  async ignore(gameId: string, issueKey: string, reason?: string): Promise<ReviewIssueOverride> {
    const response = await put<ApiEnvelope<ReviewIssueOverride>>(
      `/games/${gameId}/review-issues/${issueKey}/ignore`,
      { reason: reason || undefined },
    )
    return response.data
  },

  async restore(gameId: string, issueKey: string): Promise<void> {
    await del<ApiEnvelope<void>>(`/games/${gameId}/review-issues/${issueKey}/ignore`)
  },
}

export default reviewIssuesService
