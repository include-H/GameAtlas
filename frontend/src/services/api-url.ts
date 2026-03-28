import { API_BASE_URL } from './api'

function normalizeApiBase(baseUrl: string): string {
  return baseUrl.endsWith('/') ? baseUrl.slice(0, -1) : baseUrl
}

function normalizeApiPath(path: string): string {
  return path.startsWith('/') ? path : `/${path}`
}

// Browser-driven API URLs (href/action/src) do not use the axios client,
// so they must be built from the shared API base in one explicit place.
export function buildApiUrl(path: string): string {
  return `${normalizeApiBase(API_BASE_URL)}${normalizeApiPath(path)}`
}
