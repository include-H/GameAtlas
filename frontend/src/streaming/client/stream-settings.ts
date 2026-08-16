// Per-device streaming preferences (resolution / fps / codec / bitrate /
// audio), persisted to localStorage. Matches the option set exposed by
// moonlight-android so the defaults and ranges feel familiar.

import type { StreamConfig, VideoCodec, AudioConfiguration } from './moonlight-client';

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

export function loadSettings(): StreamSettings {
  try {
    const raw = localStorage.getItem(KEY);
    if (!raw) return defaultSettings();
    const parsed = JSON.parse(raw) as Partial<StreamSettings>;
    const def = defaultSettings();
    return {
      width: parsed.width ?? def.width,
      height: parsed.height ?? def.height,
      fps: parsed.fps ?? def.fps,
      bitrateMbps: parsed.bitrateMbps ?? def.bitrateMbps,
      codec: parsed.codec ?? def.codec,
      audio: parsed.audio ?? def.audio,
      showStats: parsed.showStats ?? def.showStats,
    };
  } catch {
    return defaultSettings();
  }
}

/** True when (width, height) is not one of the named presets. */
export function isCustomResolution(width: number, height: number): boolean {
  return !RESOLUTION_OPTIONS.some((r) => r.width === width && r.height === height);
}

export function saveSettings(s: StreamSettings): void {
  localStorage.setItem(KEY, JSON.stringify(s));
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
