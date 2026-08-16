// Video surface holder for the moonlight client.
//
// The actual VideoDecoder runs INSIDE the wasm worker (see src/wasm/worker.ts).
// This module owns just the display surface and the writable side of the
// MediaStreamTrackGenerator that the worker decodes into.
//
// Rendering surface preference:
//   1. <video> element fed by MediaStreamTrackGenerator. The writable stream
//      is transferred to the worker so decoded VideoFrames go straight onto
//      the track. ChromeOS puts the video on a hardware-overlay plane,
//      bypassing the GPU compositor entirely.
//   2. <canvas> fallback for browsers without MediaStreamTrackGenerator.
//      In this case we ask the worker to post chunks back; only used if
//      MSTG is not available.
//
// In practice all current Chromium-based browsers have MSTG so the fallback
// is rare. The fallback path is intentionally simpler — no in-worker
// decoder, no surface stats — because it's only a safety net.

import type { StreamConfig, VideoStats as ClientVideoStats } from '../client/moonlight-client';

export interface VideoStats {
  width: number;
  height: number;
  fps: number;
  decodeLatencyMs: number;
  /** drawImage time on the canvas fallback. Always 0 on the MSTG path. */
  renderLatencyMs: number;
  /** expectedDisplayTime - now from requestVideoFrameCallback (MSTG only). */
  presentLatencyMs: number;
  dropped: number;
}

export class VideoSurface {
  /** The element actually receiving frames. */
  element!: HTMLVideoElement | HTMLCanvasElement;

  /** Writable side of the MSTG. Transferred to the worker on init. */
  private writable?: WritableStream<VideoFrame>;
  private surfaceKind: 'mstg' | 'canvas' = 'mstg';

  // Last stats reported from the worker.
  private lastDecodeMs = 0;
  private lastFps = 0;
  private lastDropped = 0;
  private currentWidth = 0;
  private currentHeight = 0;
  private presentLatencyEma = 0;
  private emaAlpha = 0.1;

  constructor(private container: HTMLElement, private config: StreamConfig) {}

  /** Build the surface and return the writable stream that should be
   *  transferred to the worker. Returns undefined if MSTG isn't available;
   *  the caller should treat that as fatal for now (canvas fallback is TBD). */
  init(): WritableStream<VideoFrame> | undefined {
    const Ctor = (globalThis as unknown as {
      MediaStreamTrackGenerator?: new (init: { kind: string }) => MediaStreamTrack & {
        writable: WritableStream<VideoFrame>;
      };
    }).MediaStreamTrackGenerator;

    if (!Ctor) {
      console.error('[video] MediaStreamTrackGenerator not available — install/upgrade Chromium');
      this.element = document.createElement('canvas');
      this.element.width = this.config.width;
      this.element.height = this.config.height;
      this.container.appendChild(this.element);
      this.surfaceKind = 'canvas';
      return undefined;
    }

    const track = new Ctor({ kind: 'video' });
    const video = document.createElement('video');
    video.autoplay = true;
    video.muted = true;
    video.playsInline = true;
    video.controls = false;
    video.disablePictureInPicture = true;
    (video as HTMLVideoElement & { disableRemotePlayback?: boolean }).disableRemotePlayback = true;
    video.srcObject = new MediaStream([track]);
    this.container.appendChild(video);
    this.element = video;
    this.surfaceKind = 'mstg';

    this.writable = track.writable;
    this.beginPresentationStats();
    console.info('[video] surface ready: mstg (worker-decoded)');
    return this.writable;
  }

  /** Apply stats reported by the worker. */
  updateStats(stats: ClientVideoStats) {
    this.lastDecodeMs = stats.decodeLatencyMs;
    this.lastFps = stats.fps;
    this.lastDropped = stats.dropped;
    if (stats.firstFrame && this.element instanceof HTMLVideoElement) {
      // Element dimensions may not match config; pull from the video.
      this.currentWidth = this.element.videoWidth || this.config.width;
      this.currentHeight = this.element.videoHeight || this.config.height;
      console.info('[video] first frame on surface: %dx%d', this.currentWidth, this.currentHeight);
    }
  }

  stats(): VideoStats {
    return {
      width: this.currentWidth || this.config.width,
      height: this.currentHeight || this.config.height,
      fps: this.lastFps,
      decodeLatencyMs: this.lastDecodeMs,
      renderLatencyMs: 0,
      presentLatencyMs: this.presentLatencyEma,
      dropped: this.lastDropped,
    };
  }

  close() {
    this.element?.remove();
    this.writable = undefined;
  }

  private beginPresentationStats() {
    if (this.surfaceKind !== 'mstg') return;
    type RVFCMeta = { presentationTime: number; expectedDisplayTime?: number };
    type WithRVFC = HTMLVideoElement & {
      requestVideoFrameCallback?: (cb: (now: number, meta: RVFCMeta) => void) => number;
    };
    const v = this.element as WithRVFC;
    if (!v.requestVideoFrameCallback) return;
    const tick = (_now: number, meta: RVFCMeta) => {
      const displayTime = meta.expectedDisplayTime ?? meta.presentationTime;
      const dt = displayTime - performance.now();
      if (dt > 0 && dt < 100) {
        this.presentLatencyEma = this.presentLatencyEma === 0
          ? dt
          : this.emaAlpha * dt + (1 - this.emaAlpha) * this.presentLatencyEma;
      }
      // Pick up width/height once the first frame lands.
      if (!this.currentWidth && v.videoWidth) {
        this.currentWidth = v.videoWidth;
        this.currentHeight = v.videoHeight;
      }
      v.requestVideoFrameCallback!(tick);
    };
    v.requestVideoFrameCallback(tick);
  }
}
