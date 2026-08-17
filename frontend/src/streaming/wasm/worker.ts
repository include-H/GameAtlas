/// <reference lib="webworker" />
//
// Dedicated worker that hosts the WASM module. The worker owns:
//   - the WASM heap and all moonlight-common-c threads
//   - the WebSocket connection to the host proxy (多路复用，见 proxy-transport)
//   - the inbound packet pump (proxy -> wasm)
//
// The main thread sends commands (start/stop/input) and receives video and
// audio frames as Transferable buffers.

import { loadMoonlightModule, type MoonlightModule } from './bridge';
import { ProxyTransport } from '../transport/proxy-transport';
import type { LaunchInfo, StreamConfig, VideoCodec } from '../client/moonlight-client';
import type { Host } from '../client/host-store';

type MainToWorker =
  | { type: 'prepare'; proxyUrl: string }
  | { type: 'start'; host: Host; config: StreamConfig; launch: LaunchInfo }
  | { type: 'stop' }
  | { type: 'initVideoDecoder'; writable: WritableStream<VideoFrame>; codec: VideoCodec; width: number; height: number; fps: number }
  | { type: 'mouseMove'; dx: number; dy: number }
  | { type: 'mousePosition'; x: number; y: number; rw: number; rh: number }
  | { type: 'mouseButton'; action: number; button: number }
  | { type: 'keyboard'; keyCode: number; action: number; modifiers: number }
  | { type: 'scroll'; amount: number }
  | { type: 'controller'; idx: number; buttons: number; lt: number; rt: number; lsx: number; lsy: number; rsx: number; rsy: number }
  | { type: 'controllerArrival'; idx: number; controllerType: number; supportedButtons: number; capabilities: number }
  | { type: 'requestIdr' };

type WorkerToMain =
  | { type: 'audio'; data: Uint8Array }
  | { type: 'status'; message: string }
  | { type: 'prepared' }
  | { type: 'connected' }
  | { type: 'terminated'; error: number }
  | { type: 'log'; message: string }
  | { type: 'videoFormat'; format: number }
  | { type: 'videoStats'; decodeLatencyMs: number; fps: number; dropped: number; firstFrame: boolean };

let module: MoonlightModule | null = null;
let transport: ProxyTransport | null = null;

// In-worker video decoder. Receives chunks directly from the wasm pthreads
// (via handleForwardedCall) and writes decoded VideoFrames to a writable
// stream that was transferred from the main thread (MediaStreamTrackGenerator).
// This eliminates the chunk-data postMessage hop to main.
type VideoDecoderState = {
  decoder: VideoDecoder;
  writer: WritableStreamDefaultWriter<VideoFrame>;
  codec: VideoCodec;
  width: number;
  height: number;
  fps: number;
  firstChunkSeen: boolean;
  framesSubmitted: number;
  framesDecoded: number;
  dropped: number;
  submitTimes: Map<number, number>;
  decodeLatencyEma: number;
  outputGapEma: number;        // ms between consecutive outputs (= real decode rate)
  lastOutputTime: number;
  emaAlpha: number;
  fpsCounter: number[];
  firstFrameReported: boolean;
  maxQueueSeen: number;
  /** Once we extract SPS+PPS from an H.264 IDR and reconfigure with an AVCC
   *  description, the bitstream must be sent length-prefixed instead of
   *  Annex-B. */
  avccMode: boolean;
};

let videoState: VideoDecoderState | null = null;

self.addEventListener('error', (e) => {
  post({ type: 'log', message: `worker uncaught: ${e.message} @ ${e.filename}:${e.lineno}` });
});
self.addEventListener('unhandledrejection', (e: PromiseRejectionEvent) => {
  post({ type: 'log', message: `worker unhandled rejection: ${(e.reason as Error)?.message ?? e.reason}` });
});

// IMPORTANT: use addEventListener, NOT `self.onmessage = ...`.
// Emscripten's pthread runtime installs its own `self.onmessage` handler
// to receive proxy-queue and pthread-coordination messages from worker
// children. Overwriting it would kill that channel. addEventListener
// leaves emscripten's handler in place; we just filter out either our
// own MainToWorker commands or `__mlw` envelopes forwarded from pthreads.
self.addEventListener('message', async (e: MessageEvent) => {
  const msg = e.data;
  if (!msg || typeof msg !== 'object') return;

  // Pthread workers can't reach Module.transport / Module.videoSink etc.
  // directly, so the JS library (moonlight_imports.js) forwards each call
  // as a `__mlw` envelope via self.postMessage; this is where we re-issue
  // it against the real Module on the main wasm worker.
  if ('__mlw' in msg && typeof (msg as { __mlw?: unknown }).__mlw === 'string') {
    console.info('[worker] received __mlw envelope:', (msg as { __mlw: string }).__mlw);
    handleForwardedCall(msg as Record<string, unknown>);
    return;
  }

  if (typeof (msg as { type?: unknown }).type !== 'string') return;
  const cmd = msg as MainToWorker;
  try {
    switch (cmd.type) {
      case 'prepare':
        await handlePrepare(cmd.proxyUrl);
        break;
      case 'start':
        await handleStart(cmd.host, cmd.config, cmd.launch);
        break;
      case 'stop':
        await handleStop();
        break;
      case 'initVideoDecoder':
        await initVideoDecoder(cmd.writable, cmd.codec, cmd.width, cmd.height, cmd.fps);
        break;
      case 'mouseMove':
        module?.ccall('mlw_send_mouse_move', 'number', ['number', 'number'], [cmd.dx, cmd.dy]);
        break;
      case 'mousePosition':
        module?.ccall('mlw_send_mouse_position', 'number',
                      ['number', 'number', 'number', 'number'],
                      [cmd.x, cmd.y, cmd.rw, cmd.rh]);
        break;
      case 'mouseButton':
        module?.ccall('mlw_send_mouse_button', 'number', ['number', 'number'], [cmd.action, cmd.button]);
        break;
      case 'keyboard':
        module?.ccall('mlw_send_keyboard', 'number',
                      ['number', 'number', 'number'],
                      [cmd.keyCode, cmd.action, cmd.modifiers]);
        break;
      case 'scroll':
        module?.ccall('mlw_send_scroll', 'number', ['number'], [cmd.amount]);
        break;
      case 'controller':
        module?.ccall('mlw_send_controller', 'number',
                      ['number', 'number', 'number', 'number',
                       'number', 'number', 'number', 'number', 'number'],
                      [cmd.idx, cmd.buttons, cmd.lt, cmd.rt, cmd.lsx, cmd.lsy, cmd.rsx, cmd.rsy]);
        break;
      case 'controllerArrival':
        module?.ccall('mlw_send_controller_arrival', 'number',
                      ['number', 'number', 'number', 'number'],
                      [cmd.idx, cmd.controllerType, cmd.supportedButtons, cmd.capabilities]);
        break;
      case 'requestIdr':
        module?.ccall('mlw_request_idr', null, [], []);
        break;
    }
  } catch (err) {
    post({ type: 'log', message: `worker error: ${(err as Error).message}` });
  }
});

// Re-issue a forwarded JS call from a pthread worker against the real
// transport/sinks/event-handlers on this (the main wasm) worker.
function handleForwardedCall(env: Record<string, unknown>) {
  console.info('[worker] handleForwardedCall:', env.__mlw, 'transport=' + (transport ? 'set' : 'null'));
  switch (env.__mlw) {
    case 'open':
      transport?.openChannel(env.channel as number, env.host as string, env.port as number, env.proto as number);
      break;
    case 'send':
      transport?.sendChannel(env.channel as number, env.data as Uint8Array);
      break;
    case 'close':
      transport?.closeChannel(env.channel as number);
      break;
    case 'video':
      submitVideoChunk(env.data as Uint8Array, env.pts as number, env.flags as number);
      break;
    case 'audio':
      post({ type: 'audio', data: env.data as Uint8Array }, [(env.data as Uint8Array).buffer]);
      break;
    case 'event': {
      const kind = env.kind as string;
      if (kind === 'connected') post({ type: 'connected' });
      else if (kind === 'terminated') post({ type: 'terminated', error: env.err as number });
      else if (kind === 'log') post({ type: 'log', message: env.message as string });
      else if (kind === 'stage') post({ type: 'status', message: `stage ${env.stage} ${env.state}${env.err != null ? ` (${env.err})` : ''}` });
      break;
    }
  }
}

// DedicatedWorkerGlobalScope 的 postMessage 重载（消息 + transfer 列表）
// 需要精确类型；lib.webworker 的全局 self 与项目 lib.dom 冲突，因此
// 用最小接口收敛（替代上游的 `self as any`）。
interface WorkerScope {
  postMessage(message: WorkerToMain, transfer?: Transferable[]): void;
}

function post(msg: WorkerToMain, transfers: Transferable[] = []) {
  (self as unknown as WorkerScope).postMessage(msg, transfers);
}

// Phase 1: load wasm + open the proxy WebSocket. We do this BEFORE the
// PWA calls /launch on Sunshine because Sunshine's launch session expires
// after ping_timeout (default 10s) if no RTSP TCP connection arrives in
// that window. Moonlight-android works because its native moonlight-common-c
// lib is preloaded - we have to load wasm here, so we front-load it.
async function handlePrepare(proxyUrl: string) {
  if (module) {
    post({ type: 'prepared' });
    return;
  }
  post({ type: 'log', message: 'loading WASM module…' });
  module = await loadMoonlightModule();
  post({ type: 'log', message: `WASM loaded, connecting transport ${proxyUrl}…` });
  transport = new ProxyTransport(proxyUrl);
  try {
    await transport.connect();
  } catch (e) {
    post({ type: 'log', message: `transport.connect failed: ${(e as Error).message}` });
    throw e;
  }
  post({ type: 'log', message: 'transport connected, wiring hooks…' });

  // Wire JS-side hooks before starting.
  module.transport = {
    open: (ch, h, p, proto) => transport!.openChannel(ch, h, p, proto),
    send: (ch, data) => transport!.sendChannel(ch, data),
    close: (ch) => transport!.closeChannel(ch),
  };
  transport.onPacket = (ch, data) => {
    const ptr = module!._malloc(data.byteLength);
    module!.HEAPU8.set(data, ptr);
    module!.ccall('mlw_inbound_packet', null,
                  ['number', 'number', 'number'],
                  [ch, ptr, data.byteLength]);
    module!._free(ptr);
  };

  module.videoSink = (data, ptsUs, flags) => {
    // Called from the main wasm worker thread (synchronous path). Pthread
    // forwarded callbacks go through handleForwardedCall -> submitVideoChunk
    // instead. Both feed the same in-worker decoder.
    submitVideoChunk(new Uint8Array(data), ptsUs, flags);
  };
  module.audioSink = (data) => {
    const copy = new Uint8Array(data);
    post({ type: 'audio', data: copy }, [copy.buffer]);
  };
  module.events = {
    onConnected: () => post({ type: 'connected' }),
    onTerminated: (err) => post({ type: 'terminated', error: err }),
    onLog: (m) => post({ type: 'log', message: m }),
    onStage: (stage, state, err) =>
      post({ type: 'status', message: `stage ${stage} ${state}${err != null ? ` (${err})` : ''}` }),
    onVideoFormat: (format) => {
      console.info('[worker] onVideoFormat: 0x' + format.toString(16));
      reconfigureForFormat(format);
      post({ type: 'videoFormat', format });
    },
  };

  post({ type: 'log', message: 'wasm ready; waiting for launch info before mlw_start' });
  post({ type: 'prepared' });
}

// Phase 2: now that the PWA has issued /launch on Sunshine (raising the
// 10s ping_timeout window), call mlw_start immediately so moonlight-common-c
// opens the RTSP TCP connection inside that window.
async function handleStart(host: Host, config: StreamConfig, launch: LaunchInfo) {
  if (!module) {
    post({ type: 'terminated', error: -1 });
    return;
  }
  // Decode the RI key the proxy already shared with the host on /api/launch.
  const riKey = new Uint8Array(16);
  for (let i = 0; i < 16; i++) {
    riKey[i] = parseInt(launch.riKeyHex.substr(i * 2, 2), 16);
  }
  const riKeyPtr = module._malloc(16);
  module.HEAPU8.set(riKey, riKeyPtr);

  // From moonlight-common-c/src/Limelight.h:
  //   #define MAKE_AUDIO_CONFIGURATION(channelCount, channelMask) \
  //       (((channelMask) << 16) | (channelCount << 8) | 0xCA)
  const makeAudioConfig = (channels: number, mask: number) =>
    ((mask & 0xffff) << 16) | ((channels & 0xff) << 8) | 0xca;

  // Pass ALL codecs the browser supports as a bitmask (matches
  // moonlight-android's approach). moonlight-common-c negotiates the
  // best format during RTSP; we learn the result via onVideoFormat.
  const VIDEO_FORMAT_H264 = 0x0001;
  const VIDEO_FORMAT_HEVC = 0x0100;
  const VIDEO_FORMAT_AV1_MAIN8 = 0x1000;

  // 声明跟随 UI 选择的编码：全量声明会让 RTSP 协商自动选 AV1 Main 10
  // （10bit），而 codecString 只生成 8bit 描述——解码器按 8bit 解 10bit
  // 流会出"256 色"偏色，且切换 codec 后等 IDR 期间黑屏。默认 H.264 最稳。
  // AV1 只声明 MAIN8：10bit 路径不做（HDR 明确不在范围内）。
  const videoFormat =
    config.codec === 'av1' ? VIDEO_FORMAT_AV1_MAIN8 :
    config.codec === 'hevc' ? VIDEO_FORMAT_HEVC :
    VIDEO_FORMAT_H264;

  const audioConfig =
    config.audioConfiguration === 'surround71' ? makeAudioConfig(8, 0x63f)
    : config.audioConfiguration === 'surround51' ? makeAudioConfig(6, 0x3f)
    : makeAudioConfig(2, 0x3); // stereo default

  post({ type: 'log', message: `calling mlw_start_async (rtsp=${launch.rtspSessionUrl}, audioConfig=0x${audioConfig.toString(16)})` });
  // IMPORTANT: use mlw_start_async, NOT mlw_start. The synchronous variant
  // blocks the wasm worker's JS event loop for ~15s on RTSP failure (and
  // until first frame on success). While blocked, ws.onmessage cannot fire,
  // so the host's RTSP response never reaches us and every session times
  // out. mlw_start_async spawns a pthread to run LiStartConnection and
  // returns immediately; the worker stays free to dispatch WebSocket data.
  const err = module.ccall(
    'mlw_start_async',
    'number',
    ['string', 'string', 'string', 'string',
     'number', 'number', 'number', 'number', 'number',
     'number', 'number', 'number',
     'number', 'number', 'number'],
    [host.address, launch.appVersion, launch.gfeVersion, launch.rtspSessionUrl,
     config.width, config.height, config.fps, config.bitrateKbps, config.packetSize,
     videoFormat, audioConfig, /*encryptionFlags=*/0x00,
     riKeyPtr, 16, launch.riKeyId],
  ) as number;

  module._free(riKeyPtr);

  post({ type: 'log', message: `mlw_start_async returned ${err}` });
  if (err !== 0) {
    // Couldn't spawn the connection thread; real RTSP failures arrive via
    // the onTerminated event from inside the pthread.
    post({ type: 'terminated', error: err });
  }
}

async function handleStop() {
  if (!module) return;
  module.ccall('mlw_stop', null, [], []);
  transport?.close();
  transport = null;
  if (videoState) {
    try { videoState.decoder.close(); } catch { /* ignore */ }
    try { await videoState.writer.close(); } catch { /* ignore */ }
    videoState = null;
  }
}

// -------- in-worker video decoder --------

// Pick a codec string whose level/profile covers the target resolution + fps.
// Chrome will sometimes silently fall back to *software* decode if the hinted
// level is below what the actual SPS asks for — and software decode is what
// kills latency. Err high: hw decoders happily accept higher-level hints.
function codecString(codec: VideoCodec, width: number, height: number, fps: number): string {
  const pixels = width * height;
  switch (codec) {
    case 'h264': {
      // H.264 High profile (64.00.XX). Level needs to cover res+fps:
      //   1080p30 → 4.0 (28)   1080p60 → 4.2 (2A)
      //   1440p60 → 5.1 (33)   4K30    → 5.1 (33)   4K60 → 5.2 (34)
      let levelHex: string;
      if (pixels <= 1920 * 1080 && fps <= 30) levelHex = '28';
      else if (pixels <= 1920 * 1080) levelHex = '2a';
      else if (pixels <= 2560 * 1440) levelHex = '33';
      else if (fps <= 30) levelHex = '33';
      else levelHex = '34';
      return `avc1.6400${levelHex}`;
    }
    case 'hevc': {
      // HEVC Main profile. L120 = 4.0, L150 = 5.0, L153 = 5.1, L156 = 5.2.
      let l: string;
      if (pixels <= 1920 * 1080 && fps <= 30) l = 'L120';
      else if (pixels <= 1920 * 1080) l = 'L123';
      else if (pixels <= 2560 * 1440) l = 'L150';
      else if (fps <= 30) l = 'L150';
      else l = 'L153';
      return `hev1.1.6.${l}.90`;
    }
    case 'av1': {
      // AV1 Main profile, 8-bit. Level index from AV1 spec table A.1.
      //   level_idx 5 (2.0) handles 1080p30. 8 (4.0) handles 1080p60.
      //   13 (5.1) handles 4K30. 14 (5.2) handles 4K60.
      let lvl: string;
      if (pixels <= 1920 * 1080 && fps <= 30) lvl = '05';
      else if (pixels <= 1920 * 1080) lvl = '08';
      else if (pixels <= 2560 * 1440) lvl = '12';
      else if (fps <= 30) lvl = '13';
      else lvl = '14';
      return `av01.0.${lvl}M.08`;
    }
  }
}

async function initVideoDecoder(
  writable: WritableStream<VideoFrame>,
  codec: VideoCodec,
  width: number,
  height: number,
  fps: number,
) {
  if (videoState) {
    post({ type: 'log', message: 'initVideoDecoder called twice; closing previous' });
    try { videoState.decoder.close(); } catch { /* ignore */ }
    try { videoState.writer.close().catch(() => { /* ignore */ }); } catch { /* ignore */ }
    videoState = null;
  }

  const cs = codecString(codec, width, height, fps);

  // Diagnose whether hardware decode is *actually* available for this codec
  // string. A "soft fallback" here is the #1 reason for 50+ms decode
  // latency on FydeOS — VAAPI may not be wired up for H.264.
  try {
    const hw = await VideoDecoder.isConfigSupported({
      codec: cs,
      hardwareAcceleration: 'prefer-hardware',
      codedWidth: width,
      codedHeight: height,
      optimizeForLatency: true,
    });
    const sw = await VideoDecoder.isConfigSupported({
      codec: cs,
      hardwareAcceleration: 'prefer-software',
      codedWidth: width,
      codedHeight: height,
      optimizeForLatency: true,
    });
    const hwActual = hw.supported ? (hw.config?.hardwareAcceleration ?? '?') : 'unsupported';
    const swActual = sw.supported ? (sw.config?.hardwareAcceleration ?? '?') : 'unsupported';
    post({
      type: 'log',
      message: `[video] capability check: hw=${hwActual} (supported=${hw.supported}) sw=${swActual} (supported=${sw.supported})`,
    });
    if (hw.supported && hwActual === 'prefer-software') {
      post({ type: 'log', message: '[video] WARNING: prefer-hardware fell back to software — decode latency will be high. Check chrome://gpu.' });
    }
  } catch (e) {
    post({ type: 'log', message: `[video] capability check failed: ${(e as Error).message}` });
  }

  const writer = writable.getWriter();
  const decoder = new VideoDecoder({
    output: (frame) => onDecodedFrame(frame),
    error: (e) => post({ type: 'log', message: `[video] decoder error: ${e.message}` }),
  });

  decoder.configure({
    codec: cs,
    hardwareAcceleration: 'prefer-hardware',
    optimizeForLatency: true,
    codedWidth: width,
    codedHeight: height,
  });

  videoState = {
    decoder,
    writer,
    codec,
    width,
    height,
    fps,
    firstChunkSeen: false,
    framesSubmitted: 0,
    framesDecoded: 0,
    dropped: 0,
    submitTimes: new Map(),
    decodeLatencyEma: 0,
    outputGapEma: 0,
    lastOutputTime: 0,
    emaAlpha: 0.1,
    fpsCounter: [],
    firstFrameReported: false,
    maxQueueSeen: 0,
    avccMode: false,
  };

  post({ type: 'log', message: `[video] decoder configured: codec=${cs} ${width}x${height}@${fps}` });
}

function reconfigureForFormat(format: number) {
  if (!videoState) return;
  const newCodec: VideoCodec =
    (format & 0xF000) ? 'av1' :
    (format & 0x0F00) ? 'hevc' :
    'h264';
  if (newCodec === videoState.codec) return;
  post({ type: 'log', message: `[video] switching codec: ${videoState.codec} -> ${newCodec}` });
  videoState.codec = newCodec;
  // Reset keyframe gate so we drop deltas until the first IDR for the new codec.
  videoState.firstChunkSeen = false;
  if (videoState.decoder.state === 'configured') {
    videoState.decoder.configure({
      codec: codecString(newCodec, videoState.width, videoState.height, videoState.fps),
      hardwareAcceleration: 'prefer-hardware',
      optimizeForLatency: true,
      codedWidth: videoState.width,
      codedHeight: videoState.height,
    });
  }
}

// Walk an Annex-B bitstream and return [start, end) for each NAL unit
// (excluding the start code). End is exclusive.
function findNalus(buf: Uint8Array): Array<{ start: number; end: number; nalType: number }> {
  const out: Array<{ start: number; end: number; nalType: number }> = [];
  const len = buf.length;
  let i = 0;
  while (i < len) {
    // Find 00 00 00 01 or 00 00 01.
    let sc = -1;
    let scLen = 0;
    for (let j = i; j + 2 < len; j++) {
      if (buf[j] === 0 && buf[j + 1] === 0) {
        if (buf[j + 2] === 1) { sc = j; scLen = 3; break; }
        if (j + 3 < len && buf[j + 2] === 0 && buf[j + 3] === 1) { sc = j; scLen = 4; break; }
      }
    }
    if (sc < 0) break;
    const naluStart = sc + scLen;
    // Find next start code or end of buffer.
    let next = -1;
    for (let j = naluStart; j + 2 < len; j++) {
      if (buf[j] === 0 && buf[j + 1] === 0 &&
          (buf[j + 2] === 1 || (j + 3 < len && buf[j + 2] === 0 && buf[j + 3] === 1))) {
        next = j;
        break;
      }
    }
    const naluEnd = next < 0 ? len : next;
    if (naluEnd > naluStart) {
      out.push({ start: naluStart, end: naluEnd, nalType: buf[naluStart] & 0x1f });
    }
    i = naluEnd;
  }
  return out;
}

// -------- minimal H.264 SPS bit-parser --------
// Just enough to inspect VUI bitstream_restriction without modifying the SPS.
// Operates on RBSP (emulation-prevention bytes 00 00 03 → 00 00 stripped).

function ebspToRbsp(ebsp: Uint8Array): Uint8Array {
  const out = new Uint8Array(ebsp.length);
  let o = 0;
  for (let i = 0; i < ebsp.length; i++) {
    if (i + 2 < ebsp.length && ebsp[i] === 0 && ebsp[i + 1] === 0 && ebsp[i + 2] === 0x03) {
      out[o++] = 0;
      out[o++] = 0;
      i += 2; // skip the 03
    } else {
      out[o++] = ebsp[i];
    }
  }
  return out.subarray(0, o);
}

class BitReader {
  pos = 0;
  constructor(public buf: Uint8Array) {}
  read(n: number): number {
    let v = 0;
    for (let i = 0; i < n; i++) {
      const b = this.buf[this.pos >> 3];
      v = (v << 1) | ((b >> (7 - (this.pos & 7))) & 1);
      this.pos++;
    }
    return v >>> 0;
  }
  readUe(): number {
    let zeros = 0;
    while (zeros < 32 && this.read(1) === 0) zeros++;
    return zeros === 0 ? 0 : ((1 << zeros) - 1) + this.read(zeros);
  }
  readSe(): number {
    const k = this.readUe();
    return (k & 1) ? (k + 1) >> 1 : -(k >> 1);
  }
}

/** Parse an H.264 SPS NAL unit (without the start code) and log the
 *  no-reorder-related VUI fields. Best-effort; bails on unexpected shape. */
function logSpsRestrictions(spsNalu: Uint8Array) {
  try {
    // First byte is NAL header (forbidden_zero | nal_ref_idc | nal_unit_type).
    const rbsp = ebspToRbsp(spsNalu.subarray(1));
    const br = new BitReader(rbsp);
    const profile_idc = br.read(8);
    br.read(8); // constraint flags + reserved
    const level_idc = br.read(8);
    br.readUe(); // seq_parameter_set_id

    if ([100, 110, 122, 244, 44, 83, 86, 118, 128, 138, 139, 134, 135].includes(profile_idc)) {
      const chroma_format_idc = br.readUe();
      if (chroma_format_idc === 3) br.read(1);
      br.readUe(); // bit_depth_luma_minus8
      br.readUe(); // bit_depth_chroma_minus8
      br.read(1);  // qpprime_y_zero_transform_bypass_flag
      const scaling = br.read(1);
      if (scaling) {
        // Skip scaling matrices — bail out, we don't need them for our check.
        post({ type: 'log', message: '[sps] scaling matrices present; skipping VUI inspection' });
        return;
      }
    }
    br.readUe(); // log2_max_frame_num_minus4
    const pic_order_cnt_type = br.readUe();
    if (pic_order_cnt_type === 0) {
      br.readUe(); // log2_max_pic_order_cnt_lsb_minus4
    } else if (pic_order_cnt_type === 1) {
      br.read(1); br.readSe(); br.readSe();
      const N = br.readUe();
      for (let i = 0; i < N; i++) br.readSe();
    }
    br.readUe(); // max_num_ref_frames
    br.read(1);  // gaps_in_frame_num_value_allowed_flag
    br.readUe(); // pic_width_in_mbs_minus1
    br.readUe(); // pic_height_in_map_units_minus1
    const frame_mbs_only = br.read(1);
    if (!frame_mbs_only) br.read(1);
    br.read(1);  // direct_8x8_inference_flag
    const frame_cropping = br.read(1);
    if (frame_cropping) { br.readUe(); br.readUe(); br.readUe(); br.readUe(); }
    const vui_present = br.read(1);
    if (!vui_present) {
      post({ type: 'log', message: `[sps] profile=${profile_idc} level=${level_idc} vui=absent → decoder MUST guess reorder` });
      return;
    }

    // VUI parameters.
    const ar = br.read(1); if (ar) { const idc = br.read(8); if (idc === 255) { br.read(16); br.read(16); } }
    const ov = br.read(1); if (ov) br.read(1);
    const vs = br.read(1); if (vs) { br.read(3); br.read(1); const cd = br.read(1); if (cd) { br.read(8); br.read(8); br.read(8); } }
    const cl = br.read(1); if (cl) { br.readUe(); br.readUe(); }
    const ti = br.read(1); if (ti) { br.read(32); br.read(32); br.read(1); }

    const skipHrd = () => {
      const cpb_cnt = br.readUe() + 1;
      br.read(4); br.read(4);
      for (let i = 0; i < cpb_cnt; i++) { br.readUe(); br.readUe(); br.read(1); }
      br.read(5); br.read(5); br.read(5); br.read(5);
    };
    const nal_hrd = br.read(1); if (nal_hrd) skipHrd();
    const vcl_hrd = br.read(1); if (vcl_hrd) skipHrd();
    if (nal_hrd || vcl_hrd) br.read(1);
    br.read(1); // pic_struct_present_flag
    const restr = br.read(1);
    if (!restr) {
      post({ type: 'log', message: `[sps] profile=${profile_idc} level=${level_idc} bitstream_restriction=ABSENT → decoder guesses reorder buffer (this is why decode latency is high)` });
      return;
    }
    br.read(1);                            // motion_vectors_over_pic_boundaries_flag
    br.readUe(); br.readUe();              // max_bytes_per_pic_denom, max_bits_per_mb_denom
    br.readUe(); br.readUe();              // log2_max_mv_length_h/v
    const max_num_reorder_frames = br.readUe();
    const max_dec_frame_buffering = br.readUe();
    post({
      type: 'log',
      message: `[sps] profile=${profile_idc} level=${level_idc} max_num_reorder_frames=${max_num_reorder_frames} max_dec_frame_buffering=${max_dec_frame_buffering}`,
    });
  } catch (e) {
    post({ type: 'log', message: `[sps] parse failed: ${(e as Error).message}` });
  }
}

class BitWriter {
  buf: Uint8Array;
  pos = 0;
  constructor(maxBytes: number) {
    this.buf = new Uint8Array(maxBytes);
  }
  write(value: number, n: number) {
    for (let i = n - 1; i >= 0; i--) {
      const bit = (value >>> i) & 1;
      const byteIdx = this.pos >> 3;
      const bitIdx = 7 - (this.pos & 7);
      if (bit) this.buf[byteIdx] |= (1 << bitIdx);
      this.pos++;
    }
  }
  writeUe(value: number) {
    const v = value + 1;
    let zeros = 0;
    let temp = v;
    while (temp > 1) { temp >>>= 1; zeros++; }
    this.write(0, zeros);
    this.write(v, zeros + 1);
  }
  writeSe(value: number) {
    const codeNum = value > 0 ? 2 * value - 1 : -2 * value;
    this.writeUe(codeNum);
  }
  output(): Uint8Array {
    return this.buf.subarray(0, (this.pos + 7) >> 3);
  }
}

/** Re-emulate an RBSP into EBSP form (insert 0x03 after any 00 00 sequence
 *  followed by a byte ≤ 0x03). */
function rbspToEbsp(rbsp: Uint8Array): Uint8Array {
  // Worst case: every byte is preceded by 00 00 → 1.5x expansion. Allocate 2x.
  const out = new Uint8Array(rbsp.length * 2);
  let o = 0;
  let zeros = 0;
  for (let i = 0; i < rbsp.length; i++) {
    const b = rbsp[i];
    if (zeros >= 2 && b <= 3) {
      out[o++] = 0x03;
      zeros = 0;
    }
    out[o++] = b;
    zeros = b === 0 ? zeros + 1 : 0;
  }
  return out.subarray(0, o);
}

/** Rewrite an H.264 SPS NAL unit (in EBSP form, including the 1-byte NAL
 *  header) to include `bitstream_restriction_flag=1` with `max_num_reorder
 *  _frames=0`. Returns the modified SPS in EBSP form, or undefined if the
 *  shape is too exotic to safely transform. */
function rewriteSpsForLowLatency(spsNalu: Uint8Array): Uint8Array | undefined {
  try {
    const naluHeader = spsNalu[0];
    const rbsp = ebspToRbsp(spsNalu.subarray(1));
    const br = new BitReader(rbsp);
    const bw = new BitWriter(rbsp.length * 2 + 32);

    const copy = (n: number) => bw.write(br.read(n), n);
    const copyUe = () => bw.writeUe(br.readUe());
    const copySe = () => bw.writeSe(br.readSe());

    const profile_idc = br.read(8); bw.write(profile_idc, 8);
    copy(8); // constraint flags + reserved
    copy(8); // level_idc
    copyUe(); // seq_parameter_set_id

    if ([100, 110, 122, 244, 44, 83, 86, 118, 128, 138, 139, 134, 135].includes(profile_idc)) {
      const chroma_format_idc = br.readUe(); bw.writeUe(chroma_format_idc);
      if (chroma_format_idc === 3) copy(1);
      copyUe(); copyUe();
      copy(1);
      const scaling = br.read(1); bw.write(scaling, 1);
      if (scaling) return undefined; // not handled
    }
    copyUe(); // log2_max_frame_num_minus4
    const pic_order_cnt_type = br.readUe(); bw.writeUe(pic_order_cnt_type);
    if (pic_order_cnt_type === 0) {
      copyUe();
    } else if (pic_order_cnt_type === 1) {
      copy(1); copySe(); copySe();
      const N = br.readUe(); bw.writeUe(N);
      for (let i = 0; i < N; i++) copySe();
    }
    copyUe(); // max_num_ref_frames
    copy(1);
    copyUe(); copyUe();
    const frame_mbs_only = br.read(1); bw.write(frame_mbs_only, 1);
    if (!frame_mbs_only) copy(1);
    copy(1);
    const frame_cropping = br.read(1); bw.write(frame_cropping, 1);
    if (frame_cropping) { copyUe(); copyUe(); copyUe(); copyUe(); }

    const vui_present_orig = br.read(1);
    bw.write(1, 1); // force VUI present

    if (vui_present_orig) {
      // Copy VUI through to bitstream_restriction_flag.
      const ar = br.read(1); bw.write(ar, 1);
      if (ar) {
        const idc = br.read(8); bw.write(idc, 8);
        if (idc === 255) { copy(16); copy(16); }
      }
      const ov = br.read(1); bw.write(ov, 1); if (ov) copy(1);
      const vs = br.read(1); bw.write(vs, 1);
      if (vs) {
        copy(3); copy(1);
        const cd = br.read(1); bw.write(cd, 1);
        if (cd) { copy(8); copy(8); copy(8); }
      }
      const cl = br.read(1); bw.write(cl, 1); if (cl) { copyUe(); copyUe(); }
      const ti = br.read(1); bw.write(ti, 1);
      if (ti) { copy(32); copy(32); copy(1); }

      const copyHrd = () => {
        const cpb_cnt_minus1 = br.readUe(); bw.writeUe(cpb_cnt_minus1);
        copy(4); copy(4);
        for (let i = 0; i <= cpb_cnt_minus1; i++) { copyUe(); copyUe(); copy(1); }
        copy(5); copy(5); copy(5); copy(5);
      };
      const nal_hrd = br.read(1); bw.write(nal_hrd, 1); if (nal_hrd) copyHrd();
      const vcl_hrd = br.read(1); bw.write(vcl_hrd, 1); if (vcl_hrd) copyHrd();
      if (nal_hrd || vcl_hrd) copy(1);
      copy(1); // pic_struct_present_flag

      // Force bitstream_restriction_flag=1 regardless of original.
      const restr_orig = br.read(1);
      bw.write(1, 1);
      if (restr_orig) {
        // Already had restriction; copy fields but force reorder to 0.
        copy(1); copyUe(); copyUe(); copyUe(); copyUe();
        br.readUe();                // drop original max_num_reorder_frames
        bw.writeUe(0);
        copyUe();                   // max_dec_frame_buffering
      } else {
        // No restriction in original; append a fresh one with reorder=0.
        bw.write(1, 1);              // motion_vectors_over_pic_boundaries_flag
        bw.writeUe(2);               // max_bytes_per_pic_denom
        bw.writeUe(1);               // max_bits_per_mb_denom
        bw.writeUe(16);              // log2_max_mv_length_horizontal
        bw.writeUe(16);              // log2_max_mv_length_vertical
        bw.writeUe(0);               // max_num_reorder_frames ← the point
        bw.writeUe(1);               // max_dec_frame_buffering
      }
    } else {
      // VUI was absent entirely; write a minimal one with only restriction.
      bw.write(0, 8);                // ar/ov/vs/cl/ti/nal_hrd/vcl_hrd/pic_struct all 0
      bw.write(1, 1);                // bitstream_restriction_flag
      bw.write(1, 1);                // motion_vectors_over_pic_boundaries_flag
      bw.writeUe(2);
      bw.writeUe(1);
      bw.writeUe(16);
      bw.writeUe(16);
      bw.writeUe(0);                 // max_num_reorder_frames
      bw.writeUe(1);                 // max_dec_frame_buffering
    }

    // rbsp_trailing_bits: 1 bit set, then zeros to byte-align.
    bw.write(1, 1);
    while ((bw.pos & 7) !== 0) bw.write(0, 1);

    const newRbsp = bw.output();
    const out = new Uint8Array(1 + newRbsp.length);
    out[0] = naluHeader;
    out.set(newRbsp, 1);
    return rbspToEbsp(out);
  } catch (e) {
    post({ type: 'log', message: `[sps] rewrite failed: ${(e as Error).message}` });
    return undefined;
  }
}

/** Build an AVCC `description` (extradata) from H.264 SPS + PPS NAL units. */
function buildAvccDescription(sps: Uint8Array, pps: Uint8Array): Uint8Array {
  // configurationVersion(1) profile(1) compat(1) level(1) lengthSize(1) numSps(1) spsLen(2) sps numPps(1) ppsLen(2) pps
  const out = new Uint8Array(1 + 3 + 1 + 1 + 2 + sps.length + 1 + 2 + pps.length);
  let o = 0;
  out[o++] = 1;
  out[o++] = sps[1]; // profile_idc
  out[o++] = sps[2]; // profile_compatibility
  out[o++] = sps[3]; // level_idc
  out[o++] = 0xFF;   // lengthSizeMinusOne = 3 -> 4-byte length prefixes
  out[o++] = 0xE1;   // numOfSequenceParameterSets = 1
  out[o++] = (sps.length >> 8) & 0xff;
  out[o++] = sps.length & 0xff;
  out.set(sps, o); o += sps.length;
  out[o++] = 1;      // numOfPictureParameterSets = 1
  out[o++] = (pps.length >> 8) & 0xff;
  out[o++] = pps.length & 0xff;
  out.set(pps, o);
  return out;
}

/** Re-encode an Annex-B bitstream into AVCC (4-byte length-prefixed NALUs),
 *  dropping SPS/PPS (already in the description) and AUDs. */
function annexBToAvcc(data: Uint8Array): Uint8Array {
  const nalus = findNalus(data);
  let total = 0;
  const keep: Array<{ start: number; end: number }> = [];
  for (const n of nalus) {
    if (n.nalType === 7 || n.nalType === 8 || n.nalType === 9) continue; // SPS/PPS/AUD
    const len = n.end - n.start;
    total += 4 + len;
    keep.push(n);
  }
  const out = new Uint8Array(total);
  let o = 0;
  for (const n of keep) {
    const len = n.end - n.start;
    out[o++] = (len >>> 24) & 0xff;
    out[o++] = (len >>> 16) & 0xff;
    out[o++] = (len >>> 8) & 0xff;
    out[o++] = len & 0xff;
    out.set(data.subarray(n.start, n.end), o);
    o += len;
  }
  return out;
}

/** For an H.264 IDR, extract SPS+PPS and reconfigure with an AVCC description.
 *  This lets Chrome read max_num_reorder_frames from the SPS VUI and disable
 *  internal output buffering. After reconfig, all chunks (including this one)
 *  must be sent as length-prefixed AVCC instead of Annex-B. */
function maybeReconfigureWithDescription(data: Uint8Array) {
  if (!videoState || videoState.codec !== 'h264' || videoState.avccMode) return;
  const nalus = findNalus(data);
  let sps: Uint8Array | undefined;
  let pps: Uint8Array | undefined;
  for (const n of nalus) {
    if (n.nalType === 7 && !sps) sps = data.subarray(n.start, n.end);
    if (n.nalType === 8 && !pps) pps = data.subarray(n.start, n.end);
  }
  if (!sps || !pps) {
    post({ type: 'log', message: '[video] IDR missing SPS/PPS; staying in Annex-B mode' });
    return;
  }
  logSpsRestrictions(sps);

  // Inject bitstream_restriction_flag + max_num_reorder_frames=0 into the
  // SPS so Chrome's H.264 decoder allows its output buffer to drop to 1
  // frame. Without this, Chrome conservatively keeps ~4 frames of buffer.
  const rewritten = rewriteSpsForLowLatency(sps);
  const spsForDesc = rewritten ?? sps;
  if (rewritten) {
    post({ type: 'log', message: `[sps] rewrote: ${sps.length}→${rewritten.length} bytes, max_num_reorder_frames=0` });
    logSpsRestrictions(rewritten); // confirm round-trip
  } else {
    post({ type: 'log', message: '[sps] could not rewrite SPS; using original (decode latency will stay high)' });
  }
  const description = buildAvccDescription(spsForDesc, pps);
  try {
    videoState.decoder.configure({
      codec: codecString(videoState.codec, videoState.width, videoState.height, videoState.fps),
      hardwareAcceleration: 'prefer-hardware',
      optimizeForLatency: true,
      codedWidth: videoState.width,
      codedHeight: videoState.height,
      description,
    });
    videoState.avccMode = true;
    post({
      type: 'log',
      message: `[video] reconfigured with AVCC description (sps=${sps.length} pps=${pps.length}) — decoder should disable reorder buffer`,
    });
  } catch (e) {
    post({ type: 'log', message: `[video] AVCC reconfigure failed, keeping Annex-B: ${(e as Error).message}` });
  }
}

function submitVideoChunk(data: Uint8Array, ptsUs: number, flags: number) {
  if (!videoState) return; // decoder not yet attached; drop silently
  const keyFrame = (flags & 1) !== 0;
  if (!videoState.firstChunkSeen && !keyFrame) {
    // No reference frame yet; dropping is cheap and the renderer is bombarded
    // with deltas until Sunshine sends the IDR moonlight-common-c is requesting.
    return;
  }

  // First IDR for H.264: switch to AVCC description so the decoder knows the
  // stream doesn't reorder. Re-do this on every IDR (cheap; no-op once set).
  if (keyFrame && videoState.codec === 'h264' && !videoState.avccMode) {
    maybeReconfigureWithDescription(data);
  }

  // In AVCC mode the bitstream needs to be length-prefixed, not Annex-B.
  const chunkData = videoState.avccMode ? annexBToAvcc(data) : data;

  videoState.firstChunkSeen = true;
  videoState.framesSubmitted++;

  const qBefore = videoState.decoder.decodeQueueSize;
  if (qBefore > videoState.maxQueueSeen) videoState.maxQueueSeen = qBefore;

  // Periodic queue/throughput report (every ~120 submits ≈ 2 s).
  if (videoState.framesSubmitted % 120 === 1) {
    post({
      type: 'log',
      message: `[video] submit=${videoState.framesSubmitted} decoded=${videoState.framesDecoded} qNow=${qBefore} qMax=${videoState.maxQueueSeen} gap=${videoState.outputGapEma.toFixed(1)}ms lat=${videoState.decodeLatencyEma.toFixed(1)}ms drop=${videoState.dropped} mode=${videoState.avccMode ? 'avcc' : 'annexb'}`,
    });
    videoState.maxQueueSeen = qBefore;
  }

  videoState.submitTimes.set(ptsUs, performance.now());

  try {
    videoState.decoder.decode(
      new EncodedVideoChunk({
        type: keyFrame ? 'key' : 'delta',
        timestamp: ptsUs,
        data: chunkData,
      }),
    );
  } catch (err) {
    videoState.dropped++;
    post({ type: 'log', message: `[video] decode threw: ${(err as Error).message}` });
  }
}

function onDecodedFrame(frame: VideoFrame) {
  if (!videoState) { frame.close(); return; }

  videoState.framesDecoded++;
  const now = performance.now();
  videoState.fpsCounter.push(now);
  while (videoState.fpsCounter.length > 0 && now - videoState.fpsCounter[0] > 1000) {
    videoState.fpsCounter.shift();
  }
  const sentAt = videoState.submitTimes.get(frame.timestamp);
  if (sentAt !== undefined) {
    const dt = now - sentAt;
    videoState.decodeLatencyEma = videoState.decodeLatencyEma === 0
      ? dt
      : videoState.emaAlpha * dt + (1 - videoState.emaAlpha) * videoState.decodeLatencyEma;
    videoState.submitTimes.delete(frame.timestamp);
  }

  // Output-to-output gap: the steady-state cadence of the decoder. If
  // the decoder is keeping up, this equals 1000/fps (16.67ms at 60fps).
  // If decodeLatencyEma >> outputGapEma, the gap between the two is
  // pure pipeline buffering, not per-frame work.
  if (videoState.lastOutputTime > 0) {
    const gap = now - videoState.lastOutputTime;
    videoState.outputGapEma = videoState.outputGapEma === 0
      ? gap
      : videoState.emaAlpha * gap + (1 - videoState.emaAlpha) * videoState.outputGapEma;
  }
  videoState.lastOutputTime = now;

  const firstFrame = !videoState.firstFrameReported;
  if (firstFrame) videoState.firstFrameReported = true;

  // Hand the frame to MSTG. The writer takes ownership and will close it.
  // 色彩修复：Sunshine 的 H.264 SPS 不带 VUI 色彩元数据，硬件解码器输出的
  // VideoFrame.colorSpace 可能是默认 rgb/unknown——Chromium 会把 YUV 数据按
  // RGB 矩阵交给 MSTG，画面偏紫 + 色带（Edge 硬解路径尤甚）。强制 BT.709。
  // new VideoFrame(frame, init) 在同 backend 下共享底层数据（引用计数，
  // 零拷贝），只覆盖元数据。
  let outFrame = frame;
  const cs = frame.colorSpace;
  const needsFix = !cs || cs.matrix === undefined || cs.matrix === 'rgb' || cs.matrix === null;
  if (needsFix) {
    try {
      // VideoFrameInit.colorSpace 是较新的 API，TS DOM lib 未收录，断言绕过。
      const init = {
        colorSpace: { colorSpace: 'bt709', transfer: 'bt709', matrix: 'bt709' },
      } as unknown as VideoFrameInit;
      outFrame = new VideoFrame(frame, init);
      frame.close();
    } catch {
      outFrame = frame;
    }
  }
  videoState.writer.write(outFrame).catch((e) => {
    post({ type: 'log', message: `[video] writer.write failed: ${(e as Error).message}` });
    try { outFrame.close(); } catch { /* already closed */ }
  });

  // Post stats periodically (~ every 60 frames ≈ 1 s at 60 fps).
  if (firstFrame || videoState.framesDecoded % 60 === 0) {
    post({
      type: 'videoStats',
      decodeLatencyMs: videoState.decodeLatencyEma,
      fps: videoState.fpsCounter.length,
      dropped: videoState.dropped,
      firstFrame,
    });
  }
}

export type { MainToWorker, WorkerToMain };
