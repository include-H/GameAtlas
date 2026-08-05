export function resolveAssetUrl(path?: string | null): string {
  if (!path) return ''
  return path
}

export function resolveAssetCandidates(path?: string | null): string[] {
  if (!path) return []
  return [resolveAssetUrl(path)].filter(Boolean)
}
