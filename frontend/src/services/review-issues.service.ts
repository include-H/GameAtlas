import { del, put } from './api'
import type { ApiEnvelope, ReviewIssueOverride } from './types'

const reviewIssuesService = {
  async ignore(gameId: string, issueKey: string, reason?: string): Promise<ReviewIssueOverride> {
    const response = await put<ApiEnvelope<ReviewIssueOverride>>(
      `/games/${gameId}/review-issues/${issueKey}/ignore`,
      // 2026-04-07: keep review override reason semantics backend-owned.
      // Impact: the client forwards explicit empty input instead of inventing its
      // own nil/fallback rules before service-layer normalization runs.
      { reason },
    )
    return response.data
  },

  async restore(gameId: string, issueKey: string): Promise<void> {
    await del<ApiEnvelope<void>>(`/games/${gameId}/review-issues/${issueKey}/ignore`)
  },
}

export default reviewIssuesService
