import type { Host } from './host-store';
import type { MainToWorker, WorkerToMain } from '../wasm/worker';
import { streamProxyWsUrl } from './api-base';

export type VideoCodec = 'h264' | 'hevc' | 'av1';
export type AudioConfiguration = 'stereo' | 'surround51' | 'surround71';

export interface StreamConfig {
  width: number;
  height: number;
  fps: number;
  bitrateKbps: number;
  codec: VideoCodec;
  audioConfiguration: AudioConfiguration;
  packetSize: number;
}

export interface LaunchInfo {
  rtspSessionUrl: string;
  appVersion: string;
  gfeVersion: string;
  riKeyHex: string;
  riKeyId: number;
}

export interface VideoStats {
  decodeLatencyMs: number;
  fps: number;
  dropped: number;
  firstFrame: boolean;
}

export interface ClientOptions {
  host: Host;
  config: StreamConfig;
  proxyUrl?: string; // ws[s]://proxy/<id>
  onAudioFrame: (samples: Uint8Array) => void;
  onStatus: (msg: string) => void;
  onTerminated: (err?: unknown) => void;
  onVideoFormat?: (format: number) => void;
  onVideoStats?: (stats: VideoStats) => void;
}

export class MoonlightClient {
  private worker: Worker;
  private connected = false;
  private preparedResolve?: () => void;
  private preparedPromise: Promise<void>;

  constructor(private opts: ClientOptions) {
    this.worker = new Worker(new URL('../wasm/worker.ts', import.meta.url), {
      type: 'module',
      name: 'moonlight-wasm',
    });
    this.worker.onmessage = (e: MessageEvent<WorkerToMain>) => this.handleWorkerMessage(e.data);
    this.worker.onerror = (e) => {
      console.error('[moonlight] worker error', e);
      this.opts.onTerminated(e);
    };
    this.preparedPromise = new Promise<void>((resolve) => {
      this.preparedResolve = resolve;
    });
  }

  /** Phase 1: load wasm + open the proxy WebSocket. Call this BEFORE
   *  /launch on the host - otherwise Sunshine's 10s ping_timeout expires
   *  before our RTSP TCP connection arrives. */
  async prepare(): Promise<void> {
    this.post({
      type: 'prepare',
      proxyUrl: this.opts.proxyUrl ?? streamProxyWsUrl(),
    });
    await this.preparedPromise;
  }

  /** Phase 2: with launch info from /launch in hand, kick off mlw_start.
   *  Must be called immediately after /launch returns so we connect RTSP
   *  inside Sunshine's ping_timeout window. */
  async start(launch: LaunchInfo): Promise<void> {
    this.post({
      type: 'start',
      host: this.opts.host,
      config: this.opts.config,
      launch,
    });
  }

  async disconnect(): Promise<void> {
    this.post({ type: 'stop' });
    this.connected = false;
    this.worker.terminate();
  }

  /** Send the MediaStreamTrackGenerator's writable to the worker so the
   *  VideoDecoder runs in the worker and writes decoded frames straight
   *  into the track. Must be called before start(). */
  attachVideoDecoder(writable: WritableStream<VideoFrame>, codec: VideoCodec, width: number, height: number, fps: number) {
    this.worker.postMessage(
      { type: 'initVideoDecoder', writable, codec, width, height, fps },
      [writable as unknown as Transferable],
    );
  }

  // ---------- input forwarders ----------

  sendMouseMove(dx: number, dy: number) {
    if (this.connected) this.post({ type: 'mouseMove', dx, dy });
  }
  sendMousePosition(x: number, y: number, refW: number, refH: number) {
    if (this.connected) this.post({ type: 'mousePosition', x, y, rw: refW, rh: refH });
  }
  sendMouseButton(action: 'press' | 'release', button: 1 | 2 | 3 | 4 | 5) {
    if (!this.connected) return;
    this.post({ type: 'mouseButton', action: action === 'press' ? 0x07 : 0x08, button });
  }
  sendKeyboard(keyCode: number, action: 'down' | 'up', modifiers: number) {
    if (!this.connected) return;
    this.post({
      type: 'keyboard',
      keyCode,
      action: action === 'down' ? 0x03 : 0x04,
      modifiers,
    });
  }
  sendScroll(amount: number) {
    if (this.connected) this.post({ type: 'scroll', amount });
  }
  sendController(state: ControllerState) {
    if (!this.connected) return;
    this.post({
      type: 'controller',
      idx: state.index,
      buttons: state.buttons,
      lt: state.leftTrigger,
      rt: state.rightTrigger,
      lsx: state.leftStickX,
      lsy: state.leftStickY,
      rsx: state.rightStickX,
      rsy: state.rightStickY,
    });
  }
  sendControllerArrival(index: number, controllerType: number, supportedButtons: number, capabilities: number) {
    if (!this.connected) return;
    this.post({ type: 'controllerArrival', idx: index, controllerType, supportedButtons, capabilities });
  }
  requestIdr() {
    if (this.connected) this.post({ type: 'requestIdr' });
  }

  // ---------- internals ----------

  private post(msg: MainToWorker) {
    this.worker.postMessage(msg);
  }

  private handleWorkerMessage(msg: WorkerToMain) {
    switch (msg.type) {
      case 'prepared':
        this.preparedResolve?.();
        break;
      case 'videoStats':
        this.opts.onVideoStats?.({
          decodeLatencyMs: msg.decodeLatencyMs,
          fps: msg.fps,
          dropped: msg.dropped,
          firstFrame: msg.firstFrame,
        });
        break;
      case 'audio':
        this.opts.onAudioFrame(msg.data);
        break;
      case 'status':
        this.opts.onStatus(msg.message);
        break;
      case 'connected':
        this.connected = true;
        this.opts.onStatus('Connected');
        break;
      case 'terminated':
        this.connected = false;
        this.opts.onTerminated(msg.error);
        break;
      case 'log':
        // Use console.info so the messages survive the default DevTools
        // filter while still being distinguishable from real errors.
        console.info('[moonlight-wasm]', msg.message);
        break;
      case 'videoFormat': {
        const fmtHex = '0x' + msg.format.toString(16);
        console.info('[client] onVideoFormat:', fmtHex);
        this.opts.onVideoFormat?.(msg.format);
      }
        break;
    }
  }
}

export interface ControllerState {
  index: number;
  buttons: number;
  leftTrigger: number;
  rightTrigger: number;
  leftStickX: number;
  leftStickY: number;
  rightStickX: number;
  rightStickY: number;
}
