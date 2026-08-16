// Thin wrapper around the emscripten-generated moonlight.js module.
//
// The wasm is loaded lazily from /public/wasm/moonlight.js by the worker
// (see worker.ts). The main thread never touches the wasm directly - all
// communication goes through worker messages.

export interface TransportImpl {
  open(channel: number, host: string, port: number, proto: number): void;
  send(channel: number, data: Uint8Array): number;
  close(channel: number): void;
}

export interface ModuleEvents {
  onStage?: (stage: number, state: 'starting' | 'failed', err?: number) => void;
  onConnected?: () => void;
  onTerminated?: (err: number) => void;
  onLog?: (msg: string) => void;
  onRumble?: (controller: number, low: number, high: number) => void;
  onHttp?: (id: number, status: number, body: Uint8Array) => void;
  onVideoFormat?: (format: number) => void;
}

export type EmscriptenType = 'number' | 'string' | 'boolean' | 'array';
export type EmscriptenReturnType = EmscriptenType | null;

export interface MoonlightModule {
  HEAPU8: Uint8Array;
  _malloc(size: number): number;
  _free(ptr: number): void;
  ccall(name: string, returnType: EmscriptenReturnType, argTypes: EmscriptenType[], args: unknown[]): unknown;
  cwrap(name: string, returnType: EmscriptenReturnType, argTypes: EmscriptenType[]): (...args: unknown[]) => unknown;

  // Custom slots populated by the wrapper before mlw_init is called.
  transport?: TransportImpl;
  videoSink?: (data: Uint8Array, ptsUs: number, flags: number) => void;
  audioSink?: (data: Uint8Array) => void;
  events?: ModuleEvents;
}

let modulePromise: Promise<MoonlightModule> | null = null;

const WASM_ENTRY = '/wasm/moonlight.js';

export async function loadMoonlightModule(): Promise<MoonlightModule> {
  if (modulePromise) return modulePromise;
  modulePromise = (async () => {
    const factoryMod = await import(/* @vite-ignore */ WASM_ENTRY);
    const factory: ((opts?: unknown) => Promise<MoonlightModule>) | undefined =
      factoryMod.default ?? factoryMod.createMoonlightModule;
    if (!factory) {
      throw new Error('moonlight.js did not export a module factory');
    }

    // Fetch moonlight.js text to supply as mainScriptUrlOrBlob so that
    // pthread workers are created via a blob URL. This avoids any issues
    // with `new URL("moonlight.js", import.meta.url)` resolution inside
    // bundled workers and ensures module-worker creation is reliable.
    const moonlightJsUrl = new URL(WASM_ENTRY, location.href).href;
    const resp = await fetch(moonlightJsUrl, { credentials: 'same-origin' });
    if (!resp.ok) throw new Error(`failed to fetch moonlight.js: ${resp.status}`);
    const moonlightJsText = await resp.text();
    const workerBlob = new Blob([moonlightJsText], { type: 'text/javascript' });

    const m = await factory({
      locateFile(path: string) {
        if (path.endsWith('.wasm')) return '/wasm/moonlight.wasm';
        return '/wasm/' + path;
      },
      mainScriptUrlOrBlob: workerBlob,
    });
    m.ccall('mlw_init', 'number', [], []);
    return m;
  })();
  return modulePromise;
}
