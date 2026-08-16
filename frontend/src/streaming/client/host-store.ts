// Simple localStorage-backed host registry. Pairing material (PPK, cert/key
// pair) lives here too once we wire up pairing.

const STORAGE_KEY = 'moonlight.hosts.v1';

export interface Host {
  id: string;
  name: string;
  address: string;
  /** Port for the HTTP control endpoint. Defaults to 47989 (GFE) / 47984 (TLS). */
  httpPort?: number;
  httpsPort?: number;
  paired: boolean;
  /** Pinned server cert (PEM) returned during pairing. */
  serverCert?: string;
  /** Client cert/key pair (PEM) generated locally during pairing. */
  clientCert?: string;
  clientKey?: string;
  lastSeen: number;
  lastAppId?: number;
}

export function loadHosts(): Host[] {
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    if (!raw) return [];
    return JSON.parse(raw) as Host[];
  } catch {
    return [];
  }
}

export function saveHosts(hosts: Host[]): void {
  localStorage.setItem(STORAGE_KEY, JSON.stringify(hosts));
}

export function upsertHost(host: Host): void {
  const hosts = loadHosts();
  const idx = hosts.findIndex((h) => h.id === host.id);
  if (idx >= 0) hosts[idx] = host;
  else hosts.push(host);
  saveHosts(hosts);
}

export function removeHost(id: string): void {
  saveHosts(loadHosts().filter((h) => h.id !== id));
}
