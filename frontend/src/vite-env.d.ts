/// <reference types="vite/client" />

interface ImportMetaEnv {
  readonly VITE_USERNAME?: string
}

interface ImportMeta {
  readonly env: ImportMetaEnv
}

declare module '*.vue' {
  import type { DefineComponent } from 'vue'
  const component: DefineComponent<{}, {}, Record<string, never>>
  export default component
}
