// AudioWorklet that plays interleaved f32 PCM from a ring buffer.
// Opus decoding happens in the main thread via WebCodecs AudioDecoder;
// this worklet only handles low-latency PCM playback.
//
// When AudioDecoder is unavailable, raw Opus frames arrive as `opus`
// messages — we generate silence for the expected frame size so the
// pipeline doesn't stall.

class OpusWorkletProcessor extends AudioWorkletProcessor {
  constructor(options) {
    super();
    const channels = options?.processorOptions?.channels ?? 2;
    // 500 ms at 48 kHz — large enough to absorb AudioDecoder output bursts
    this.bufferSamples = options?.processorOptions?.bufferSamples ?? 24000;
    this.channels = channels;
    this.ring = new Float32Array(this.bufferSamples * channels);
    this.readIdx = 0;
    this.writeIdx = 0;
    this.config = null;
    this.queuedFrames = 0;
    this.lastQueueReport = currentTime;
    this.port.onmessage = (e) => this.handleMessage(e.data);
  }

  handleMessage(msg) {
    if (msg.type === 'config') {
      this.config = msg.config;
    } else if (msg.type === 'pcm') {
      this.enqueue(msg.data, msg.channels);
    } else if (msg.type === 'opus') {
      // WebCodecs AudioDecoder unavailable — stub: generate silence.
      const samplesPerChannel = this.config?.samplesPerFrame ?? 240;
      const channels = this.config?.channels ?? 2;
      const pcm = new Float32Array(samplesPerChannel * channels);
      this.enqueue(pcm, channels);
    }
  }

  enqueue(pcm, channels) {
    const cap = this.ring.length;
    for (let i = 0; i < pcm.length; i++) {
      this.ring[this.writeIdx] = pcm[i];
      this.writeIdx = (this.writeIdx + 1) % cap;
      if (this.writeIdx === this.readIdx) {
        // Overflow — drop one complete frame (all channels) so
        // L/R interleaving stays aligned. Dropping a single sample
        // would swap the channels, producing metallic/underwater audio.
        this.readIdx = (this.readIdx + channels) % cap;
        this.queuedFrames = Math.max(0, this.queuedFrames - 1);
      }
    }
    this.queuedFrames += pcm.length / channels;
  }

  process(_inputs, outputs) {
    const out = outputs[0];
    const outChannels = out.length;
    const frames = out[0].length;
    let underrun = false;

    for (let f = 0; f < frames; f++) {
      if (this.readIdx === this.writeIdx) {
        for (let c = 0; c < outChannels; c++) out[c][f] = 0;
        underrun = true;
        continue;
      }
      for (let c = 0; c < outChannels; c++) {
        out[c][f] = this.ring[this.readIdx] ?? 0;
        this.readIdx = (this.readIdx + 1) % this.ring.length;
      }
      this.queuedFrames = Math.max(0, this.queuedFrames - 1);
    }

    if (underrun) this.port.postMessage({ type: 'underrun' });
    if (currentTime - this.lastQueueReport > 0.2) {
      this.port.postMessage({ type: 'queue', frames: this.queuedFrames });
      this.lastQueueReport = currentTime;
    }
    return true;
  }
}

registerProcessor('opus-worklet', OpusWorkletProcessor);
