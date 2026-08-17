// 配对由 Go 串流代理驱动（与上游 host-proxy 相同职责）。
//
// 浏览器无法：
//   - 干净地自签 X.509 证书（WebCrypto 能生成密钥，但构造 ASN.1 证书
//     需要引入大库）
//   - 直达 RFC1918 地址上的 Sunshine/GFE 主机（Private Network Access
//     与宿主自签 TLS 证书双重障碍）
//
// 因此浏览器只负责：生成 PIN → POST /api/pair → 等待结果。Go 代理
// 执行五步 NvHTTP 配对握手，把主机证书缓存在代理侧
// （dataDir/hosts/<address>.cert.pem），并把 serverCert 回给前端
// 做展示/固定（客户端私钥留在代理内，浏览器无需保管）。
//
// 代理契约（请求）：
//   POST /api/pair
//   Content-Type: application/json
//   { "address": "192.168.1.42", "pin": "1234", "deviceName": "..." }
//
// 代理契约（响应，单次 JSON，无 SSE 流）：
//   { "paired": true,  "serverCert": "-----BEGIN CERTIFICATE-----\n..." }
//   { "paired": false, "error": "human-readable reason" }

import type { Host } from './host-store';
import { getDeviceName } from './settings';
import { streamApiUrl } from './api-base';

export interface PairOptions {
  signal?: AbortSignal;
}

export interface PairResult {
  paired: boolean;
  serverCert?: string;
  error?: string;
}

export async function pairHost(host: Host, pin: string, opts: PairOptions = {}): Promise<Host> {
  const res = await fetch(streamApiUrl('/api/pair'), {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      address: host.address,
      pin,
      deviceName: getDeviceName(),
    }),
    signal: opts.signal,
  });

  if (!res.ok) {
    throw new Error(`proxy /api/pair returned ${res.status}: ${await safeText(res)}`);
  }

  const result = (await res.json()) as PairResult;

  if (!result.paired) {
    throw new Error(result.error ?? 'host rejected the pairing PIN');
  }

  // serverCert 由后端缓存到 dataDir/hosts/<address>.cert.pem（paired 动态
  // 判断的依据），前端无需也不应保管，只更新本地展示字段。
  return {
    ...host,
    paired: true,
    lastSeen: Date.now(),
  };
}

async function safeText(res: Response): Promise<string> {
  try { return (await res.text()).slice(0, 200); } catch { return ''; }
}
