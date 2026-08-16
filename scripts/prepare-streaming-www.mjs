#!/usr/bin/env node
// 组装 Go 串流代理的 WWW 根目录（frontend/dist/streaming-www/）：
//
//   index.html   ← dist/streaming.html（串流页作为根入口，满足
//                  streaming.Server.staticHandler 的 SPA fallback 契约：
//                  index.html 必须位于 WWWRoot 根）
//   ui/          ← dist/ui（vite 构建产物，页面内为绝对路径 /ui/assets/...）
//   wasm/        ← dist/wasm（moonlight.js + moonlight.wasm，由 public/ 复制）
//
// 用法：先 npm run build，再执行本脚本，然后：
//   STREAM_WWW_ROOT=frontend/dist/streaming-www 启动后端，
//   https://<NAS>:47999/ 即串流页（/ui、/wasm、/api、/proxy 全部同源）。
import { cpSync, mkdirSync, rmSync } from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..')
const dist = path.join(root, 'frontend', 'dist')
const out = path.join(dist, 'streaming-www')

const srcHtml = path.join(dist, 'streaming.html')
const srcUi = path.join(dist, 'ui')
const srcWasm = path.join(dist, 'wasm')

rmSync(out, { recursive: true, force: true })
mkdirSync(out, { recursive: true })

cpSync(srcHtml, path.join(out, 'index.html'))
cpSync(srcUi, path.join(out, 'ui'), { recursive: true })
cpSync(srcWasm, path.join(out, 'wasm'), { recursive: true })

console.log(`[prepare-streaming-www] assembled ${out}`)
console.log(`  STREAM_WWW_ROOT=${out}`)
