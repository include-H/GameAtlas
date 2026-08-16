// Map browser KeyboardEvent.code / KeyboardEvent.key to Win32 VK codes
// (which is the wire format moonlight-common-c forwards to the host).
//
// We prefer `event.code` because it is layout-independent (USB usage
// page semantics), then fall back to `event.key` for a handful of
// codes that don't have a Win32 equivalent.

export const MODIFIER_SHIFT = 0x01;
export const MODIFIER_CTRL  = 0x02;
export const MODIFIER_ALT   = 0x04;
export const MODIFIER_META  = 0x08;

// The high byte is the moonlight key prefix (0x80) - applied by callers.
export const KEY_PREFIX = 0x80;

const TABLE: Record<string, number> = {
  // Letters
  KeyA: 0x41, KeyB: 0x42, KeyC: 0x43, KeyD: 0x44, KeyE: 0x45, KeyF: 0x46,
  KeyG: 0x47, KeyH: 0x48, KeyI: 0x49, KeyJ: 0x4A, KeyK: 0x4B, KeyL: 0x4C,
  KeyM: 0x4D, KeyN: 0x4E, KeyO: 0x4F, KeyP: 0x50, KeyQ: 0x51, KeyR: 0x52,
  KeyS: 0x53, KeyT: 0x54, KeyU: 0x55, KeyV: 0x56, KeyW: 0x57, KeyX: 0x58,
  KeyY: 0x59, KeyZ: 0x5A,

  // Top-row digits
  Digit0: 0x30, Digit1: 0x31, Digit2: 0x32, Digit3: 0x33, Digit4: 0x34,
  Digit5: 0x35, Digit6: 0x36, Digit7: 0x37, Digit8: 0x38, Digit9: 0x39,

  // Function keys
  F1: 0x70, F2: 0x71, F3: 0x72, F4: 0x73, F5: 0x74, F6: 0x75,
  F7: 0x76, F8: 0x77, F9: 0x78, F10: 0x79, F11: 0x7A, F12: 0x7B,
  F13: 0x7C, F14: 0x7D, F15: 0x7E, F16: 0x7F, F17: 0x80, F18: 0x81,
  F19: 0x82, F20: 0x83,

  // Navigation
  Escape: 0x1B,
  Tab: 0x09,
  CapsLock: 0x14,
  Backspace: 0x08,
  Enter: 0x0D,
  Space: 0x20,
  ArrowUp: 0x26, ArrowDown: 0x28, ArrowLeft: 0x25, ArrowRight: 0x27,
  Insert: 0x2D, Delete: 0x2E, Home: 0x24, End: 0x23,
  PageUp: 0x21, PageDown: 0x22,
  PrintScreen: 0x2C, ScrollLock: 0x91, Pause: 0x13,

  // Modifiers (left/right specific - moonlight cares about the side)
  ShiftLeft: 0xA0, ShiftRight: 0xA1,
  ControlLeft: 0xA2, ControlRight: 0xA3,
  AltLeft: 0xA4, AltRight: 0xA5,
  MetaLeft: 0x5B, MetaRight: 0x5C,
  ContextMenu: 0x5D,

  // Symbol keys (US layout codes - moonlight passes through to OS)
  Backquote: 0xC0, Minus: 0xBD, Equal: 0xBB,
  BracketLeft: 0xDB, BracketRight: 0xDD, Backslash: 0xDC,
  Semicolon: 0xBA, Quote: 0xDE,
  Comma: 0xBC, Period: 0xBE, Slash: 0xBF,
  IntlBackslash: 0xE2,

  // Numpad
  NumLock: 0x90,
  Numpad0: 0x60, Numpad1: 0x61, Numpad2: 0x62, Numpad3: 0x63, Numpad4: 0x64,
  Numpad5: 0x65, Numpad6: 0x66, Numpad7: 0x67, Numpad8: 0x68, Numpad9: 0x69,
  NumpadMultiply: 0x6A, NumpadAdd: 0x6B, NumpadSubtract: 0x6D,
  NumpadDecimal: 0x6E, NumpadDivide: 0x6F, NumpadEnter: 0x0D,
};

export function vkFromEvent(e: KeyboardEvent): number | null {
  return TABLE[e.code] ?? null;
}

export function modifiersFromEvent(e: KeyboardEvent): number {
  let m = 0;
  if (e.shiftKey) m |= MODIFIER_SHIFT;
  if (e.ctrlKey)  m |= MODIFIER_CTRL;
  if (e.altKey)   m |= MODIFIER_ALT;
  if (e.metaKey)  m |= MODIFIER_META;
  return m;
}
