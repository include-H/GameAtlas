// Vite resolves the `?url` suffix to a static asset URL, which is what
// AudioWorklet.addModule() needs.

import url from './opus-worklet.js?url';

export const OpusWorkletProcessorUrl = url;
