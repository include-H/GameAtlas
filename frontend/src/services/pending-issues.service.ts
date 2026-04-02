import { get } from './api'
import type { ApiEnvelope, PendingIssueCatalog } from './types'

let catalogPromise: Promise<PendingIssueCatalog> | null = null

const pendingIssuesService = {
  getCatalog(): Promise<PendingIssueCatalog> {
    if (!catalogPromise) {
      catalogPromise = get<ApiEnvelope<PendingIssueCatalog>>('/pending-issues')
        .then((response) => response.data)
        .catch((error) => {
          catalogPromise = null
          throw error
        })
    }
    return catalogPromise
  },
}

export default pendingIssuesService
