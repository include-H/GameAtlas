const MIME_EXTENSION_MAP: Record<string, string> = {
  'image/jpeg': 'jpg',
  'image/jpg': 'jpg',
  'image/png': 'png',
  'image/webp': 'webp',
  'image/gif': 'gif',
  'image/avif': 'avif',
  'image/svg+xml': 'svg',
  'video/mp4': 'mp4',
  'video/webm': 'webm',
}

export function getAssetFileExtension(
  mimeType: string | undefined,
  assetType: 'cover' | 'banner' | 'screenshot' | 'video',
): string {
  const normalized = (mimeType || '').trim().toLowerCase().split(';')[0]?.trim() || ''
  if (normalized && MIME_EXTENSION_MAP[normalized]) {
    return MIME_EXTENSION_MAP[normalized]
  }

  return assetType === 'video' ? 'mp4' : 'jpg'
}
