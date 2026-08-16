// Audio renderer for Moonlight streams.
//
// Moonlight delivers multistream Opus frames at 48 kHz. We decode with
// WebCodecs AudioDecoder (Chrome 94+) which produces AudioData; PCM is
// posted to an AudioWorklet that does low-latency ring-buffer playback.
//
// The AudioWorkletNode is created lazily — we wait for the Opus config
// packet so we can set the correct channel count and build the OpusHead
// `description` the AudioDecoder needs for multistream.

import { OpusWorkletProcessorUrl } from './opus-worklet-url';

export interface AudioStats {
  queuedMs: number;
  dropped: number;
}

interface OpusConfig {
  sampleRate: number;
  channels: number;
  streams: number;
  coupledStreams: number;
  samplesPerFrame: number;
  mapping: Uint8Array;
}

export class AudioRenderer {
  private ctx?: AudioContext;
  private node?: AudioWorkletNode;
  private decoder?: AudioDecoder;
  private opusConfig?: OpusConfig;
  private pcmBuffersPending = 0;
  private dropped = 0;
  private useWebCodecs = false;
  private frameTs = 0;

  async init(): Promise<void> {
    this.ctx = new AudioContext({
      latencyHint: 'interactive',
      sampleRate: 48000,
    });
    await this.ctx.audioWorklet.addModule(OpusWorkletProcessorUrl);

    // AudioDecoder for Opus → PCM. We'll configure it once the config
    // packet arrives (multistream Opus needs the OpusHead description).
    if (typeof AudioDecoder !== 'undefined') {
      try {
        this.decoder = new AudioDecoder({
          output: async (audioData: AudioData) => {
            const nFrames = audioData.numberOfFrames;
            const nCh = audioData.numberOfChannels;
            const fmt = audioData.format;
            // Log the first frame's format so we know what the decoder produces.
            if (this.frameTs === 0) {
              console.info('[audio] first output: %dframes %dch format=%s',
                nFrames, nCh, fmt);
            }
            // Extract each channel as planar f32, then interleave manually.
            // Relying on copyTo({format:'f32'}) to interleave can be buggy
            // across Chrome versions when the internal format is planar.
            const pcm = new Float32Array(nFrames * nCh);
            for (let c = 0; c < nCh; c++) {
              const plane = new Float32Array(nFrames);
              await audioData.copyTo(plane, { planeIndex: c, format: 'f32-planar' });
              for (let f = 0; f < nFrames; f++) pcm[f * nCh + c] = plane[f];
            }
            this.node?.port.postMessage(
              { type: 'pcm', data: pcm, channels: nCh },
              [pcm.buffer],
            );
            audioData.close();
          },
          error: (err: Error) =>
            console.error('[audio] decoder error:', err.message, err),
        });
        this.useWebCodecs = true;
        console.info('[audio] using WebCodecs AudioDecoder for Opus');
      } catch (e) {
        console.warn('[audio] AudioDecoder ctor failed, falling back', e);
      }
    } else {
      console.warn('[audio] AudioDecoder not available');
    }
  }

  // First packet is the 17-byte config block from bindings.c:audInit.
  // We use it to build the OpusHead description and create the worklet node.
  submit(data: Uint8Array): void {
    if (!this.opusConfig) {
      if (data.byteLength === 17) {
        this.opusConfig = parseOpusConfig(data);
        this.onConfig(this.opusConfig);
        return;
      }
      return; // no config yet, drop
    }

    if (this.decoder && this.useWebCodecs && this.decoder.state === 'configured') {
      this.decoder.decode(
        new EncodedAudioChunk({
          type: 'key',
          timestamp: this.frameTs,
          duration: this.opusConfig.samplesPerFrame,
          data,
        }),
      );
      this.frameTs += this.opusConfig.samplesPerFrame;
    } else {
      const copy = new Uint8Array(data);
      this.node?.port.postMessage({ type: 'opus', data: copy }, [copy.buffer]);
    }
  }

  stats(): AudioStats {
    if (!this.ctx) return { queuedMs: 0, dropped: 0 };
    return {
      queuedMs: (this.pcmBuffersPending / this.ctx.sampleRate) * 1000,
      dropped: this.dropped,
    };
  }

  close() {
    this.node?.disconnect();
    this.decoder?.close();
    this.ctx?.close();
  }

  // Called once when the config packet arrives.
  private onConfig(cfg: OpusConfig) {
    console.info('[audio] config: %dch %dHz %dstreams %dcoupled %dsamp',
      cfg.channels, cfg.sampleRate, cfg.streams, cfg.coupledStreams,
      cfg.samplesPerFrame);
    console.info('[audio] mapping:', Array.from(cfg.mapping));

    // Create the AudioWorkletNode with the correct channel count.
    if (this.ctx && !this.node) {
      this.node = new AudioWorkletNode(this.ctx, 'opus-worklet', {
        outputChannelCount: [cfg.channels],
        numberOfOutputs: 1,
        processorOptions: {
          bufferSamples: 24000, // 500 ms at 48 kHz
          channels: cfg.channels,
        },
      });
      this.node.port.onmessage = (e) => {
        if (e.data?.type === 'underrun') this.dropped++;
        if (e.data?.type === 'queue') this.pcmBuffersPending = e.data.frames;
      };
      this.node.connect(this.ctx.destination);
    }

    // Build the OpusHead description for multistream Opus (RFC 7845 §5.1).
    const desc = buildOpusHead(cfg);

    if (this.decoder && this.useWebCodecs) {
      this.decoder.configure({
        codec: 'opus',
        sampleRate: cfg.sampleRate,
        numberOfChannels: cfg.channels,
        description: desc,
      });
      console.info('[audio] decoder configured: opus %dch %dHz (multistream)',
        cfg.channels, cfg.sampleRate);
    }
  }
}

function parseOpusConfig(buf: Uint8Array): OpusConfig {
  const dv = new DataView(buf.buffer, buf.byteOffset, buf.byteLength);
  return {
    sampleRate: dv.getUint32(0, true),
    channels: dv.getUint8(4),
    streams: dv.getUint8(5),
    coupledStreams: dv.getUint8(6),
    samplesPerFrame: dv.getUint16(7, true),
    mapping: buf.slice(9, 9 + 8),
  };
}

// Build an OpusHead packet (RFC 7845) so the AudioDecoder understands
// multistream Opus. Without this, Chrome only supports standard (mono/stereo)
// Opus, and moonlight always uses multistream regardless of channel count.
function buildOpusHead(cfg: OpusConfig): ArrayBuffer {
  const mapBytes = cfg.channels; // 1 byte per output channel
  const total = 21 + mapBytes;   // magic(8) + version(1) + channels(1) +
                                 // preSkip(2) + sampleRate(4) + gain(2) +
                                 // mappingFamily(1) + streamCount(1) +
                                 // coupledCount(1) + mapping(mapBytes)
  const buf = new ArrayBuffer(total);
  const v = new DataView(buf);
  const encoder = new TextEncoder();
  const magic = encoder.encode('OpusHead');
  for (let i = 0; i < 8; i++) v.setUint8(i, magic[i]);
  v.setUint8(8, 1);                   // version
  v.setUint8(9, cfg.channels);        // output channel count
  v.setUint16(10, 3840, true);        // pre-skip (80 ms at 48 kHz)
  v.setUint32(12, cfg.sampleRate, true);
  v.setUint16(16, 0, true);           // output gain (0 dB)
  v.setUint8(18, 1);                  // mapping family: 1 = multistream
  v.setUint8(19, cfg.streams);        // stream count
  v.setUint8(20, cfg.coupledStreams); // coupled stream count
  // channel mapping: which stream/channel pair maps to each output channel.
  for (let i = 0; i < mapBytes; i++) {
    v.setUint8(21 + i, cfg.mapping[i]);
  }
  return buf;
}
