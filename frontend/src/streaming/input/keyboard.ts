// Keyboard capture, including system keys (Esc, Tab, Super/Meta, Alt+Tab,
// PrintScreen, Caps Lock).
//
// The browser only delivers these to the page when:
//   1. The page is in fullscreen mode, AND
//   2. The page has called `navigator.keyboard.lock([...keys])` for the
//      keys it wants to intercept.
//
// `keyboard.lock()` requires a transient user activation (a recent user
// gesture) and only takes effect while in fullscreen. We call it from the
// click handler that enters fullscreen + starts the stream.
//
// See: https://wicg.github.io/keyboard-lock/

import type { Capabilities } from '../capabilities';
import type { MoonlightClient } from '../client/moonlight-client';
import { vkFromEvent, modifiersFromEvent, KEY_PREFIX, MODIFIER_ALT, MODIFIER_CTRL, MODIFIER_SHIFT } from './vk-codes';

// Codes we want to intercept from the OS. The browser will swallow these
// while keyboard lock is active.
const LOCKED_KEYS = [
  'Escape',
  'Tab',
  'MetaLeft', 'MetaRight',
  'AltLeft', 'AltRight',
  'ContextMenu',
  'F11',
  'PrintScreen',
];

// Keyboard Lock API 不在标准 lib.dom 中，用精确类型描述替代上游的 any。
interface KeyboardLockAPI {
  lock(keyCodes?: string[]): Promise<void>;
  unlock(): void;
}

function keyboardLockApi(): KeyboardLockAPI | undefined {
  return (navigator as unknown as { keyboard?: KeyboardLockAPI }).keyboard;
}

export class KeyboardInput {
  private downHandler = (e: KeyboardEvent) => this.onKey(e, 'down');
  private upHandler = (e: KeyboardEvent) => this.onKey(e, 'up');
  private locked = false;

  constructor(
    private root: HTMLElement,
    private client: MoonlightClient,
    private caps: Capabilities,
  ) {}

  async attach(): Promise<void> {
    // Keyboard Lock requires a user-activation; the caller should have
    // already requested fullscreen as the activating gesture.
    const kb = keyboardLockApi();
    if (this.caps.keyboardLock && kb?.lock) {
      try {
        await kb.lock(LOCKED_KEYS);
        this.locked = true;
      } catch (err) {
        console.warn('[keyboard] lock failed', err);
      }
    } else {
      console.warn('[keyboard] Keyboard Lock API unavailable - system keys will leak to OS.');
    }

    // We attach to window so we keep receiving events even if focus drifts
    // off the canvas. capture=true so we beat the browser to Esc handling.
    window.addEventListener('keydown', this.downHandler, { capture: true });
    window.addEventListener('keyup', this.upHandler, { capture: true });
    this.root.focus();
  }

  detach() {
    window.removeEventListener('keydown', this.downHandler, { capture: true });
    window.removeEventListener('keyup', this.upHandler, { capture: true });
    const kb = keyboardLockApi();
    if (this.locked && kb?.unlock) {
      kb.unlock();
      this.locked = false;
    }
  }

  private onKey(e: KeyboardEvent, action: 'down' | 'up') {
    // Allow the user to exit a stream with Ctrl+Alt+Shift+Q like the NaCl
    // client. We do this BEFORE checking repeat so the chord always wins.
    if (action === 'down' && e.code === 'KeyQ') {
      const mods = modifiersFromEvent(e);
      if ((mods & (MODIFIER_CTRL | MODIFIER_ALT | MODIFIER_SHIFT)) === (MODIFIER_CTRL | MODIFIER_ALT | MODIFIER_SHIFT)) {
        e.preventDefault();
        this.client.disconnect();
        return;
      }
    }

    if (e.repeat) {
      // Always swallow autorepeat - the host generates its own.
      e.preventDefault();
      return;
    }

    const vk = vkFromEvent(e);
    if (vk == null) return;

    e.preventDefault();
    this.client.sendKeyboard(KEY_PREFIX << 8 | vk, action, modifiersFromEvent(e));
  }
}
