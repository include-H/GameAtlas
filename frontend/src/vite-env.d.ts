/// <reference types="vite/client" />

interface ImportMetaEnv {
  readonly VITE_USERNAME?: string
  /** 串流 API/WS 源（Go 串流代理），如 https://127.0.0.1:47999；留空 = 同源 */
  readonly VITE_STREAM_API_ORIGIN?: string
}

interface ImportMeta {
  readonly env: ImportMetaEnv
}

declare module '*.vue' {
  import type { DefineComponent } from 'vue'
  const component: DefineComponent<{}, {}, Record<string, never>>
  export default component
}
