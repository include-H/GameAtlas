import { del, get, put } from './api'
import type { ApiEnvelope, ReviewIssueOverride } from './types'

const reviewIssuesService = {
  async list(gamePublicIds?: string[]): Promise<ReviewIssueOverride[]> {
    const params = gamePublicIds && gamePublicIds.length > 0
      ? { game_ids: gamePublicIds.join(',') }
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
