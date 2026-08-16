// 串流 API 源（/api/* 与 /proxy WebSocket 所在服务）。
//
// 生产：串流页由 Go 串流代理（https://NAS:47999）自身托管，所有请求
// 同源，无需配置。
//
// 开发限制（重要）：dev server 上的串流页带有 COEP require-corp，而
// Go 代理的 /api、/proxy 响应带 `Cross-Origin-Resource-Policy:
// same-origin` 且无 CORS 头——因此即使把 VITE_STREAM_API_ORIGIN 指向
// Go 代理，跨源请求也会被 COEP 拦截。完整的串流调试请直接打开
// https://127.0.0.1:47999/（同源路径，与生产一致）。若未来要在 dev
// server 上全链路串流，需要给 Go 代理加 CORS（ACAO + preflight），
// 或让 vite 中间件按 Referer 把 /api、/proxy 转发到 :47999。

const envOrigin = (import.meta.env.VITE_STREAM_API_ORIGIN as string | undefined)?.trim() ?? '';

/** /api/* 的 HTTP 源（空串 = 同源）。 */
export const STREAM_API_ORIGIN: string = envOrigin;

/** /proxy WebSocket 的完整 URL。 */
export function streamProxyWsUrl(): string {
  if (envOrigin) {
    const wsScheme = envOrigin.startsWith('https:') ? 'wss:' : 'ws:';
    return `${wsScheme}//${envOrigin.slice(envOrigin.indexOf('//') + 2)}/proxy`;
  }
  const scheme = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
  return `${scheme}//${window.location.host}/proxy`;
}

/** 拼出 /api/* 的完整 URL（含 query）。 */
export function streamApiUrl(path: string, searchParams?: URLSearchParams): string {
  const url = new URL(path, envOrigin || window.location.origin);
  searchParams?.forEach((v, k) => url.searchParams.set(k, v));
  return url.toString();
}
