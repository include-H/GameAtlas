// Device-wide PWA settings persisted to localStorage.

const DEVICE_NAME_KEY = 'moonlight.deviceName.v1';

function defaultDeviceName(): string {
  // Best effort: ChromeOS / FydeOS don't expose the hostname to JS, so we
  // fall back to a generic label. The user can edit this in the header.
  const ua = navigator.userAgent.toLowerCase();
  if (ua.includes('cros'))    return 'Chromebook';
  if (ua.includes('android')) return 'Android Tablet';
  if (ua.includes('mac'))     return 'Mac';
  if (ua.includes('windows')) return 'Windows PC';
  if (ua.includes('linux'))   return 'Linux PC';
  return 'Moonlight PWA';
}

export function getDeviceName(): string {
  return localStorage.getItem(DEVICE_NAME_KEY) ?? defaultDeviceName();
}

export function setDeviceName(name: string): void {
  const cleaned = name.trim().slice(0, 64);
  if (cleaned) localStorage.setItem(DEVICE_NAME_KEY, cleaned);
  else         localStorage.removeItem(DEVICE_NAME_KEY);
}
