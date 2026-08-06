export function resolveAssetUrl(path?: string | null): string {
  if (!path) return ''
  return path
}

export function resolveAssetCandidates(path?: string | null): string[] {
  if (!path) return []
  return [resolveAssetUrl(path)].filter(Boolean)
}

// 请求后端按宽度生成的 WebP 变体（懒生成、永久缓存）。
// 非 /assets 路径原样返回，避免给外部 URL 或 data URI 拼参数。
export function withAssetWidth(path?: string | null, width?: number): string {
  if (!path || !width || width <= 0) return path || ''
  if (!path.startsWith('/assets/')) return path
  const separator = path.includes('?') ? '&' : '?'
  return `${path}${separator}w=${width}`
}
