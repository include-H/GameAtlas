export function resolveAssetUrl(path?: string | null): string {
  if (!path) return ''
  if (!path.startsWith('/assets/')) return path
  if (!import.meta.env.DEV) return path

  const apiBase = import.meta.env.VITE_API_BASE_URL || '/api'
  if (apiBase.startsWith('http://') || apiBase.startsWith('https://')) {
    try {
      const url = new URL(apiBase)
      return `${url.origin}${path}`
    } catch {
      return `http://127.0.0.1:3000${path}`
    }
  }

  if (typeof window !== 'undefined') {
    const protocol = window.location.protocol === 'https:' ? 'https:' : 'http:'
    const host = window.location.hostname || '127.0.0.1'
    const backendPort = import.meta.env.VITE_BACKEND_PORT || '3000'
    return `${protocol}//${host}:${backendPort}${path}`
  }

  return `http://127.0.0.1:3000${path}`
}

export function resolveAssetCandidates(path?: string | null): string[] {
  if (!path) return []
  if (!path.startsWith('/assets/')) return [path]
  if (!import.meta.env.DEV) return [path]

  const candidates: string[] = []
  const direct = resolveAssetUrl(path)
  if (direct) candidates.push(direct)
  candidates.push(path)

  return Array.from(new Set(candidates.filter(Boolean)))
}
