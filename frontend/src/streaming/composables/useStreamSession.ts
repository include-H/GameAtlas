// MoonlightClient 生命周期管理（StreamView 的 Vue 版）。
//
// 装配顺序（与上游 stream-view.ts 一致，顺序敏感）：
//   1. 请求全屏（作为用户手势窗口）
//   2. 建 <video> + MediaStreamTrackGenerator 表面，把 writable 传给 worker
//   3. 初始化音频（AudioContext + AudioWorklet，等待 Opus 配置包）
//   4. 挂输入（Keyboard Lock 需要手势窗口内调用）
//   5. client.prepare() —— 先加载 wasm + 打开代理 WebSocket
//   6. 再调 /api/launch —— Sunshine 的 launch 会话在 ping_timeout（默认
//      10s）内等 RTSP TCP 连接，因此 prepare 必须先完成
//   7. client.start(launch) —— 立即 mlw_start_async
//
// 退出：Ctrl+Alt+Shift+Q（keyboard.ts 内直接调 client.disconnect()）
// 会触发 worker 'terminated' → onTerminated → 本 composable 清理。

import { computed, onUnmounted, ref } from 'vue'
import type { Capabilities } from '../capabilities'
import type { Host } from '../client/host-store'
import type { StreamSettings } from '../client/stream-settings'
import { toStreamConfig } from '../client/stream-settings'
import { MoonlightClient, type StreamConfig, type VideoStats } from '../client/moonlight-client'
import { launchApp, type AppEntry, type LaunchResult } from '../client/nvhttp'
import { VideoSurface } from '../video/webcodecs-decoder'
import { AudioRenderer } from '../audio/audio-renderer'
import { KeyboardInput } from '../input/keyboard'
import { PointerInput } from '../input/pointer'
import { GamepadInput } from '../input/gamepad'

export type StreamPhase = 'idle' | 'starting' | 'streaming' | 'stopping' | 'failed';

export interface StreamSessionState {
  phase: StreamPhase;
  /** 覆盖层提示文本（连接状态 / 统计）。 */
  status: string;
  error: string | null;
  stats: VideoStats | null;
  audioQueuedMs: number;
}

export function useStreamSession() {
  const phase = ref<StreamPhase>('idle');
  const status = ref('');
  const error = ref<string | null>(null);
  const stats = ref<VideoStats | null>(null);
  const audioQueuedMs = ref(0);

  let client: MoonlightClient | null = null;
  let video: VideoSurface | null = null;
  let audio: AudioRenderer | null = null;
  let keyboard: KeyboardInput | null = null;
  let pointer: PointerInput | null = null;
  let gamepad: GamepadInput | null = null;

  let statsInterval: number | undefined;
  let overlayHideTimer: number | undefined;
  let overlayHidden = false;
  let overlayElement: HTMLElement | null = null;
  let sessionEnded = false;

  const isActive = computed(() => phase.value === 'starting' || phase.value === 'streaming');

  function bumpOverlay() {
    if (!overlayElement) return;
    if (overlayHidden) return;
    overlayElement.classList.remove('hidden');
    if (overlayHideTimer) clearTimeout(overlayHideTimer);
    overlayHideTimer = window.setTimeout(() => {
      overlayElement?.classList.add('hidden');
    }, 3000);
  }

  function onMouseMove() {
    bumpOverlay();
  }

  /** 启动一次串流会话。root 是播放视图根元素（视频会被追加到其中）。 */
  async function start(root: HTMLElement, host: Host, app: AppEntry, settings: StreamSettings, caps: Capabilities): Promise<void> {
    if (phase.value === 'starting' || phase.value === 'streaming') return;
    sessionEnded = false;
    error.value = null;
    stats.value = null;
    phase.value = 'starting';

    const config: StreamConfig = toStreamConfig(settings);

    try {
      // 默认窗口模式起步：自动全屏会把 1080p 流拉伸到屏幕尺寸，配合
      // Sunshine 的"桌面/流分辨率"鼠标映射在异分辨率下产生加速感。
      // 全屏由用户手动触发（StreamPlayer 的全屏按钮 / F11）。

      // 建 <video> + MSTG 表面；解码器在 worker 内。
      video = new VideoSurface(root, config);
      const writable = video.init();
      if (!writable) {
        throw new Error('MediaStreamTrackGenerator unavailable; canvas fallback not implemented');
      }

      audio = new AudioRenderer();
      await audio.init();

      client = new MoonlightClient({
        host,
        config,
        onAudioFrame: (samples) => audio?.submit(samples),
        onStatus: (msg) => { status.value = msg; },
        onTerminated: (err) => { void stop(err); },
        onVideoStats: (s) => {
          stats.value = s;
          video?.updateStats(s);
        },
      });

      // 把 MSTG 的 writable 交给 worker，视频帧不再跨线程回主线程。
      client.attachVideoDecoder(writable, config.codec, config.width, config.height, config.fps);

      // 输入挂在全屏手势窗口内。
      keyboard = new KeyboardInput(root, client, caps);
      pointer = new PointerInput(video.element, client);
      gamepad = new GamepadInput(client);
      await keyboard.attach();
      pointer.attach();
      gamepad.start();

      status.value = 'Loading streaming engine…';
      await client.prepare();

      status.value = 'Launching app on host…';
      const launch: LaunchResult = await launchApp({ host, app, config });

      status.value = 'Connecting RTSP…';
      await client.start({
        rtspSessionUrl: launch.rtspSessionUrl,
        appVersion: launch.appVersion,
        gfeVersion: launch.gfeVersion,
        riKeyHex: launch.riKeyHex,
        riKeyId: launch.riKeyId,
      });

      // 统计浮层：鼠标移动时短暂显示（ChromeOS 在无不透明兄弟合成层时
      // 更激进地把 <video> 放到硬件覆盖平面）。
      overlayElement = root.querySelector('.stream-overlay');
      root.addEventListener('mousemove', onMouseMove);
      statsInterval = window.setInterval(() => updateStats(), 500);
      bumpOverlay();
      phase.value = 'streaming';
    } catch (err) {
      phase.value = 'failed';
      error.value = (err as Error).message ?? String(err);
      console.error('[stream] start failed:', err);
      await stop(err);
    }
  }

  function updateStats() {
    if (!video) return;
    const v = video.stats();
    const a = audio?.stats();
    audioQueuedMs.value = a?.queuedMs ?? 0;
    status.value =
      `${v.width}x${v.height} @ ${v.fps.toFixed(0)} fps · ` +
      `decode ${v.decodeLatencyMs.toFixed(1)} ms · ` +
      `present ${v.presentLatencyMs.toFixed(1)} ms · ` +
      `dropped ${v.dropped} · ` +
      `audio ${audioQueuedMs.value.toFixed(0)} ms`;
  }

  /** 停止会话并清理所有资源。reason 为 worker 上报的终止原因。 */
  async function stop(reason?: unknown): Promise<void> {
    if (sessionEnded && phase.value !== 'failed') return;
    sessionEnded = true;
    if (phase.value === 'stopping') return;
    phase.value = 'stopping';

    if (statsInterval !== undefined) clearInterval(statsInterval);
    statsInterval = undefined;
    if (overlayHideTimer !== undefined) clearTimeout(overlayHideTimer);
    overlayHideTimer = undefined;

    gamepad?.stop();
    gamepad = null;
    keyboard?.detach();
    keyboard = null;
    pointer?.detach();
    pointer = null;

    await client?.disconnect();
    client = null;

    audio?.close();
    audio = null;
    video?.close();
    video = null;

    if (document.fullscreenElement) {
      await document.exitFullscreen().catch(() => {});
    }
    if (reason !== undefined && reason !== null) {
      console.warn('[stream] terminated:', reason);
    }
    phase.value = 'idle';
  }

  onUnmounted(() => {
    void stop();
  });

  return {
    phase,
    status,
    error,
    stats,
    audioQueuedMs,
    isActive,
    start,
    stop,
  };
}
