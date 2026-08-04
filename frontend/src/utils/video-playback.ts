// 预告片轮播的播放状态复位规则。
// 聚合保存后详情刷新会传入新 props（但视频 URL 通常不变）；如果无条件重置 videoReady，
// 已加载/正在播放的 <video> 不会重新触发 canplay/playing，画面会卡在透明加载态。
// 只有视频源真正变化时才需要重建播放状态。
export function shouldResetVideoPlaybackState(previousUrl: string | undefined, nextUrl: string): boolean {
  return previousUrl !== nextUrl
}
