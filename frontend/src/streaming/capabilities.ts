// Feature detection - run once at boot so the UI can warn early if the
// runtime can't actually stream.
//
// 与上游差异：transport 只有 WebSocket 一条路径（Go 代理未实现
// WebTransport），webTransport 能力项保留仅为上报参考；新增
// detectCodecSupport() 供设置页按硬件过滤编码选项。

export interface Capabilities {
  crossOriginIsolated: boolean;
  sharedArrayBuffer: boolean;
  webCodecs: boolean;
  h264Hardware: boolean;
  hevcHardware: boolean;
  av1Hardware: boolean;
  webTransport: boolean;
  webSockets: boolean;
  keyboardLock: boolean;
  pointerLock: boolean;
  gamepad: boolean;
  audioWorklet: boolean;
  fullscreen: boolean;
  installed: boolean;
}

/** 单个 codec 的 1080p60 硬件解码探测结果。 */
export interface CodecSupport {
  h264: boolean;
  hevc: boolean;
  av1: boolean;
}

async function checkCodecSupport(config: VideoDecoderConfig): Promise<boolean> {
  if (typeof VideoDecoder === 'undefined') return false;
  try {
    const r = await VideoDecoder.isConfigSupported(config);
    return r.supported === true;
  } catch {
    return false;
  }
}

/** 探测三个视频编码在 1080p60 下的解码支持（prefer-hardware）。
 *  供设置页过滤不可用的编码选项。 */
export async function detectCodecSupport(): Promise<CodecSupport> {
  // 与 worker 内 codecString() 的 H.264 level 4.2 / HEVC L123 / AV1 08
  // 保持一致（1080p60 所需的 level）。
  const [h264, hevc, av1] = await Promise.all([
    checkCodecSupport({
      codec: 'avc1.64002a', // High profile, level 4.2 (1080p60)
      hardwareAcceleration: 'prefer-hardware',
      codedWidth: 1920,
      codedHeight: 1080,
    }),
    checkCodecSupport({
      codec: 'hev1.1.6.L123.90', // Main profile, level 4.1 (1080p60)
      hardwareAcceleration: 'prefer-hardware',
      codedWidth: 1920,
      codedHeight: 1080,
    }),
    checkCodecSupport({
      codec: 'av01.0.08M.08', // Main profile, level 4.0 (1080p60)
      hardwareAcceleration: 'prefer-hardware',
      codedWidth: 1920,
      codedHeight: 1080,
    }),
  ]);
  return { h264, hevc, av1 };
}

export async function detectCapabilities(): Promise<Capabilities> {
  const installed = window.matchMedia('(display-mode: standalone)').matches;

  // Probing codec configs forces the browser to consult the HW decoder list.
  // We pick conservative configs that any modern GPU should accept.
  const [h264, hevc, av1] = await Promise.all([
    checkCodecSupport({
      codec: 'avc1.640028', // High profile, level 4.0
      hardwareAcceleration: 'prefer-hardware',
      codedWidth: 1920,
      codedHeight: 1080,
    }),
    checkCodecSupport({
      codec: 'hev1.1.6.L120.90', // Main profile, level 4.0
      hardwareAcceleration: 'prefer-hardware',
      codedWidth: 1920,
      codedHeight: 1080,
    }),
    checkCodecSupport({
      codec: 'av01.0.08M.08',
      hardwareAcceleration: 'prefer-hardware',
      codedWidth: 1920,
      codedHeight: 1080,
    }),
  ]);

  return {
    crossOriginIsolated: self.crossOriginIsolated === true,
    sharedArrayBuffer: typeof SharedArrayBuffer !== 'undefined',
    webCodecs: typeof VideoDecoder !== 'undefined',
    h264Hardware: h264,
    hevcHardware: hevc,
    av1Hardware: av1,
    webTransport: typeof WebTransport !== 'undefined',
    webSockets: typeof WebSocket !== 'undefined',
    keyboardLock: !!(navigator as unknown as { keyboard?: { lock?: unknown } }).keyboard?.lock,
    pointerLock: 'requestPointerLock' in HTMLElement.prototype,
    gamepad: typeof navigator.getGamepads === 'function',
    audioWorklet: typeof AudioWorkletNode !== 'undefined',
    fullscreen: 'requestFullscreen' in HTMLElement.prototype,
    installed,
  };
}
