import { defineConfig } from 'vitest/config'
import vue from '@vitejs/plugin-vue'
import Components from 'unplugin-vue-components/vite'
import { ArcoResolver } from 'unplugin-vue-components/resolvers'
import path from 'node:path'

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
  ],
  envDir: path.resolve(__dirname, '../backend'),
  build: {
    assetsDir: 'ui',
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
