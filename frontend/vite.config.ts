import { defineConfig } from 'vitest/config'
import vue from '@vitejs/plugin-vue'
import Components from 'unplugin-vue-components/vite'
import { ArcoResolver } from 'unplugin-vue-components/resolvers'
import path from 'node:path'
import { spawnSync } from 'node:child_process'
import type { Plugin } from 'vite'
import type { IncomingMessage, ServerResponse } from 'node:http'

// SharedArrayBuffer + Emscripten pthreads 需要跨源隔离。
// 只在串流页（/streaming.html，独立文档）的响应上加头：COOP/COEP 由顶层
// 文档决定，子资源同源加载不受影响；主站（index.html）不加头，避免影响
// 主站加载的跨源图片等资源。
const crossOriginIsolationHeaders: Record<string, string> = {
  'Cross-Origin-Opener-Policy': 'same-origin',
  'Cross-Origin-Embedder-Policy': 'require-corp',
  'Cross-Origin-Resource-Policy': 'same-origin',
}

/** dev/preview 下给 /streaming* 响应注入 COOP/COEP/CORP 头。 */
function streamingIsolationPlugin(): Plugin {
  const middleware = (
    req: IncomingMessage,
    res: ServerResponse,
    next: () => void,
  ) => {
    if (req.url?.startsWith('/streaming')) {
      for (const [k, v] of Object.entries(crossOriginIsolationHeaders)) {
        res.setHeader(k, v)
      }
    }
    next()
  }
  return {
    name: 'streaming-cross-origin-isolation',
    configureServer(server) {
      server.middlewares.use(middleware)
    },
    configurePreviewServer(server) {
      server.middlewares.use(middleware)
    },
  }
}

/**
 * 构建收尾：组装串流页 WWW 根目录（dist/streaming-www）。
 * npm run build 默认清空 dist，streaming-www 若不在此重建，就会被
 * start-dev.sh / build-release.sh 的每次构建抹掉（后端托管 404）。
 */
function assembleStreamingWwwPlugin(): Plugin {
  return {
    name: 'assemble-streaming-www',
    closeBundle() {
      const script = path.resolve(__dirname, '..', 'scripts', 'prepare-streaming-www.mjs')
      const result = spawnSync(process.execPath, [script], { stdio: 'inherit' })
      if (result.status !== 0) {
        throw new Error(`prepare-streaming-www failed with status ${result.status}`)
      }
    },
  }
}

export default defineConfig({
  plugins: [
    vue(),
    Components({
      dts: 'src/components.d.ts',
      resolvers: [
        ArcoResolver({
          importStyle: false,
          sideEffect: false,
        }),
      ],
    }),
    streamingIsolationPlugin(),
    assembleStreamingWwwPlugin(),
  ],
  envDir: path.resolve(__dirname, '../backend'),
  worker: {
    // 串流 wasm worker 是 ES module worker（new Worker(url, {type:'module'})）
    format: 'es',
  },
  build: {
    assetsDir: 'ui',
    // 多页：主站 + 云串流独立文档（输出 dist/streaming.html）
    rollupOptions: {
      input: {
        main: path.resolve(__dirname, 'index.html'),
        streaming: path.resolve(__dirname, 'streaming.html'),
      },
    },
  },
  resolve: {
    alias: {
      '@': path.resolve(__dirname, 'src'),
    },
  },
  server: {
    port: 5173,
    // 允许通过 VITE_API_TARGET 指向远程后端做联调（默认本地 3000）
    proxy: buildProxy(),
  },
  test: {
    environment: 'jsdom',
    include: ['src/**/*.test.ts'],
  },
})

function buildProxy() {
  const target = process.env.VITE_API_TARGET || 'http://127.0.0.1:3000'
  return {
    '/api': {
      target,
      changeOrigin: true,
    },
    '/assets': {
      target,
      changeOrigin: true,
    },
  }
}
