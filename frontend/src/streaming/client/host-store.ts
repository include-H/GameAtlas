// Backend-persisted host registry. Hosts live in the Go streaming proxy's
// <dataDir>/hosts.json so they survive browser storage clears and device
// switches. Pairing state (paired) is computed by the backend from the
// cached host cert (<dataDir>/hosts/<address>.cert.pem); pairing material
// (client cert/key, server cert) never leaves the server, so the frontend
// no longer stores them.

import { streamApiUrl } from './api-base';

export interface Host {
  id: string;
  name: string;
  address: string;
  /** Port for the HTTP control endpoint. Defaults to 47989 (GFE) / 47984 (TLS). */
  httpPort?: number;
  httpsPort?: number;
  paired: boolean;
  lastSeen: number;
  lastAppId?: number;
}

/** Backend wire shape of a host entry (paired is server-computed). */
interface HostDTO {
  id: string;
  name: string;
  address: string;
  lastSeen: number;
  paired: boolean;
}

function toHost(dto: HostDTO): Host {
  return {
    id: dto.id,
    name: dto.name,
    address: dto.address,
    paired: dto.paired,
    lastSeen: dto.lastSeen,
  };
}

async function parseError(res: Response): Promise<string> {
  try {
    const data = (await res.json()) as { error?: string };
    return data.error ?? `HTTP ${res.status}`;
  } catch {
    return `HTTP ${res.status}`;
  }
}

/** GET /api/hosts → full host list (paired computed by the backend). */
export async function loadHosts(): Promise<Host[]> {
  const res = await fetch(streamApiUrl('/api/hosts'), { headers: { Accept: 'application/json' } });
  if (!res.ok) {
    throw new Error(`加载主机列表失败：${await parseError(res)}`);
  }
  const data = (await res.json()) as { hosts: HostDTO[] };
  return data.hosts.map(toHost);
}

/**
 * 逐台 POST /api/hosts（按 id upsert），返回最后一次响应里的完整列表。
 * 新增主机可传 id: ''，由后端生成。
 */
export async function saveHosts(hosts: Host[]): Promise<Host[]> {
  let fresh: Host[] = [];
  for (const host of hosts) {
    const res = await fetch(streamApiUrl('/api/hosts'), {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        id: host.id,
        name: host.name,
        address: host.address,
        lastSeen: host.lastSeen,
      }),
    });
    if (!res.ok) {
      throw new Error(`保存主机失败：${await parseError(res)}`);
    }
    const data = (await res.json()) as { hosts: HostDTO[] };
    fresh = data.hosts.map(toHost);
  }
  return fresh;
}

/** DELETE /api/hosts?id=<id> → 删除后的完整列表。 */
export async function removeHost(id: string): Promise<Host[]> {
  const res = await fetch(streamApiUrl('/api/hosts', new URLSearchParams({ id })), {
    method: 'DELETE',
  });
  if (!res.ok) {
    throw new Error(`移除主机失败：${await parseError(res)}`);
  }
  const data = (await res.json()) as { hosts: HostDTO[] };
  return data.hosts.map(toHost);
}