export function resolveAssetUrl(path?: string | null): string {
  if (!path) return ''
  return path
}

export function resolveAssetCandidates(path?: string | null): string[] {
  if (!path) return []
  return [resolveAssetUrl(path)].filter(Boolean)
}

// getAssetThumbUrl derives the deterministic thumbnail URL the backend generates
// next to an uploaded image. Small originals and videos have no thumbnail file,
// so callers should fall back to the original path when the thumb 404s.
export function getAssetThumbUrl(path?: string | null): string {
  if (!path) return ''
  const match = /^(.*)\.([a-zA-Z0-9]+)$/.exec(path)
  if (!match) return path
  return `${match[1]}.thumb.jpg`
}
