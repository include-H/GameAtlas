import { defineConfig } from 'vite'
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
    proxy: {
      '/api': {
        target: 'http://127.0.0.1:3000',
        changeOrigin: true,
      },
      '/assets': {
        target: 'http://127.0.0.1:3000',
        changeOrigin: true,
      },
      '/data': {
        target: 'http://127.0.0.1:3000',
        changeOrigin: true,
      },
    },
  },
})
