// extractVideoPoster 在浏览器里读取视频文件并抽取一帧作为封面（JPEG）。
// 局域网场景避免引入 ffmpeg 依赖：MP4/WebM 在现代浏览器均可用 <video> + canvas 抽取。
export const extractVideoPoster = (file: File): Promise<Blob> => {
  return new Promise((resolve, reject) => {
    const url = URL.createObjectURL(file)
    const video = document.createElement('video')
    video.muted = true
    video.playsInline = true
    video.preload = 'metadata'
    video.src = url

    const cleanup = () => {
      video.removeAttribute('src')
      video.load()
      URL.revokeObjectURL(url)
    }

    video.onloadedmetadata = () => {
      const duration = Number.isFinite(video.duration) ? video.duration : 0
      // 跳过片头黑帧：抓 10% 处的一帧，但限定在 1s~3s 区间。
      // 第 1 帧（~0.03s）与"第 10 帧"（~0.33s）都大概率落在黑场淡入里，
      // 1s 起步、3s 封顶能稳定落在预告片正文画面。
      video.currentTime = duration > 0 ? Math.min(3, Math.max(1, duration * 0.1)) : 0.01
    }

    video.onseeked = () => {
      try {
        const canvas = document.createElement('canvas')
        const targetWidth = 640
        const sourceWidth = video.videoWidth || targetWidth
        const sourceHeight = video.videoHeight || targetWidth
        const scale = targetWidth / sourceWidth
        canvas.width = targetWidth
        canvas.height = Math.max(1, Math.round(sourceHeight * scale))

        const ctx = canvas.getContext('2d')
        if (!ctx) {
          throw new Error('canvas unavailable')
        }
        ctx.drawImage(video, 0, 0, canvas.width, canvas.height)
        canvas.toBlob((blob) => {
          cleanup()
          if (blob) {
            resolve(blob)
          } else {
            reject(new Error('poster encode failed'))
          }
        }, 'image/jpeg', 0.85)
      } catch (error) {
        cleanup()
        reject(error instanceof Error ? error : new Error('poster extraction failed'))
      }
    }

    video.onerror = () => {
      cleanup()
      reject(new Error('video load failed'))
    }
  })
}
