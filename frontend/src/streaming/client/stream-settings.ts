// Per-device streaming preferences (resolution / fps / codec / bitrate /
// audio), persisted to localStorage AND synced to the backend
// (<dataDir>/stream-settings.json via the Go proxy) so they survive
// browser storage clears. Matches the option set exposed by
// moonlight-android so the defaults and ranges feel familiar.

import type { StreamConfig, VideoCodec, AudioConfiguration } from './moonlight-client';
import { streamApiUrl } from './api-base';

const KEY = 'moonlight.streamSettings.v1';

export const RESOLUTION_OPTIONS = [
  { label: '720p',  width: 1280, height: 720  },
  { label: '1080p', width: 1920, height: 1080 },
  { label: '1440p', width: 2560, height: 1440 },
  { label: '4K',    width: 3840, height: 2160 },
] as const;

export const FPS_OPTIONS = [30, 60, 90, 120] as const;

export const CODEC_OPTIONS: { value: VideoCodec; label: string }[] = [
  { value: 'h264', label: 'H.264' },
  { value: 'hevc', label: 'H.265 / HEVC' },
  { value: 'av1',  label: 'AV1' },
];

export const AUDIO_OPTIONS: { value: AudioConfiguration; label: string }[] = [
  { value: 'stereo',     label: 'Stereo' },
  { value: 'surround51', label: '5.1 Surround' },
  { value: 'surround71', label: '7.1 Surround' },
];

export interface StreamSettings {
  width: number;
  height: number;
  fps: number;
  /** Mbps — UI uses Mbps, StreamConfig uses Kbps. */
  bitrateMbps: number;
  codec: VideoCodec;
  audio: AudioConfiguration;
  /** Show the live FPS / latency / dropped-frame overlay during streaming. */
  showStats: boolean;
}

export function defaultBitrateMbps(width: number, height: number, fps: number): number {
  // From moonlight-android's PreferenceConfiguration.getDefaultBitrate(),
  // which is itself adapted from moonlight-qt. Linear interpolation between
  // these resolution points, scaled by frame rate.
  const pixels = width * height;
  const points: [number, number][] = [
    [ 640 *  360,  1],
    [ 854 *  480,  2],
    [1280 *  720,  5],
    [1920 * 1080, 10],
    [2560 * 1440, 20],
    [3840 * 2160, 40],
  ];
  let factor = points[points.length - 1][1];
  for (let i = 0; i < points.length; i++) {
    if (pixels <= points[i][0]) {
      if (i === 0) factor = points[0][1];
      else {
        const [pPrev, fPrev] = points[i - 1];
        const [pCurr, fCurr] = points[i];
        factor = ((pixels - pPrev) / (pCurr - pPrev)) * (fCurr - fPrev) + fPrev;
      }
      break;
    }
  }
  // Don't scale linearly past 60 FPS.
  const frameFactor = (fps <= 60 ? fps : Math.sqrt(fps / 60) * 60) / 30;
  return Math.round(factor * frameFactor);
}

export function defaultSettings(): StreamSettings {
  return {
    width: 1920,
    height: 1080,
    fps: 60,
    bitrateMbps: defaultBitrateMbps(1920, 1080, 60),
    codec: 'h264',
    audio: 'stereo',
    showStats: true,
  };
}

/** 把 partial 合并到默认设置（缺字段回退默认值）。 */
function mergeSettings(partial: Partial<StreamSettings>): StreamSettings {
  const def = defaultSettings();
  return {
    width: partial.width ?? def.width,
    height: partial.height ?? def.height,
    fps: partial.fps ?? def.fps,
    bitrateMbps: partial.bitrateMbps ?? def.bitrateMbps,
    codec: partial.codec ?? def.codec,
    audio: partial.audio ?? def.audio,
    showStats: partial.showStats ?? def.showStats,
  };
}

export function loadSettings(): StreamSettings {
  try {
    const raw = localStorage.getItem(KEY);
    if (!raw) return defaultSettings();
    return mergeSettings(JSON.parse(raw) as Partial<StreamSettings>);
  } catch {
    return defaultSettings();
  }
}

/**
 * 优先后端设置（GET /api/stream-settings，异步）；后端缺失/失败时回退
 * localStorage。调用方应等它返回后再用合并后的设置。
 */
export async function loadSettingsFromServer(): Promise<StreamSettings> {
  try {
    const res = await fetch(streamApiUrl('/api/stream-settings'), {
      headers: { Accept: 'application/json' },
    });
    if (res.ok) {
      const data = (await res.json()) as { settings?: Partial<StreamSettings> };
      if (data.settings && typeof data.settings === 'object' && Object.keys(data.settings).length > 0) {
        return mergeSettings(data.settings);
      }
    }
  } catch (err) {
    console.warn('[stream] 从后端加载串流设置失败，回退 localStorage:', err);
  }
  return loadSettings();
}

/** True when (width, height) is not one of the named presets. */
export function isCustomResolution(width: number, height: number): boolean {
  return !RESOLUTION_OPTIONS.some((r) => r.width === width && r.height === height);
}

export function saveSettings(s: StreamSettings): void {
  localStorage.setItem(KEY, JSON.stringify(s));
  // 同步到后端（失败仅告警，不阻塞本地即时生效）。
  void syncSettingsToServer(s);
}

/** PUT /api/stream-settings，全量保存。失败 console.warn 不抛错。 */
async function syncSettingsToServer(s: StreamSettings): Promise<void> {
  try {
    const res = await fetch(streamApiUrl('/api/stream-settings'), {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ settings: s }),
    });
    if (!res.ok) {
      console.warn(`[stream] 保存串流设置到后端失败：HTTP ${res.status}`);
    }
  } catch (err) {
    console.warn('[stream] 保存串流设置到后端失败:', err);
  }
}

/** Convert UI settings to the StreamConfig the wasm layer expects. */
export function toStreamConfig(s: StreamSettings): StreamConfig {
  return {
    width: s.width,
    height: s.height,
    fps: s.fps,
    bitrateKbps: s.bitrateMbps * 1000,
    codec: s.codec,
    audioConfiguration: s.audio,
    packetSize: 1392,
  };
}
