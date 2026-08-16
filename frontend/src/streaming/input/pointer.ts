// Pointer input — unified for mouse, touchpad, touchscreen, pen.
//
// Strategy:
//   - pointerType === 'mouse' (covers desktop mice AND touchpads on ChromeOS):
//       request pointer lock with `unadjustedMovement: true` so OS-level
//       cursor acceleration is bypassed (touchpad motion becomes linear),
//       send relative deltas via sendMouseMove.
//   - pointerType === 'touch' or 'pen': no pointer lock, send absolute
//       positions via sendMousePosition.
//
// We listen to the unified PointerEvent API rather than MouseEvent so
// trackpad taps and touchscreen events all surface as pointerdown/up,
// which mousedown/up does not (notably, ChromeOS often does not synthesize
// mousedown/up for a trackpad tap when the pointer is locked).

import type { MoonlightClient } from '../client/moonlight-client';

const POLL_MS = 5;

const BUTTON_LEFT = 1;
const BUTTON_MIDDLE = 2;
const BUTTON_RIGHT = 3;
const BUTTON_X1 = 4;
const BUTTON_X2 = 5;

export class PointerInput {
  private locked = false;
  private dx = 0;
  private dy = 0;
  private wheelTicks = 0;
  private pollHandle?: number;

  /** While a finger is on the touchscreen, this is its pointerId. Extra
   *  fingers are ignored — moonlight has no native multi-touch protocol. */
  private touchPointerId: number | null = null;

  private onPointerDown = (e: PointerEvent) => { this.trace('pointerdown', e); this.handleDown(e); };
  private onPointerUp = (e: PointerEvent) => { this.trace('pointerup', e); this.handleUp(e); };
  private onPointerMove = (e: PointerEvent) => this.handleMove(e);
  private onPointerCancel = (e: PointerEvent) => this.handleCancel(e);
  private onTouchStart = (e: TouchEvent) => this.handleTouchStart(e);
  private onTouchMove = (e: TouchEvent) => this.handleTouchMove(e);
  private onTouchEnd = (e: TouchEvent) => this.handleTouchEnd(e);
  private onWheel = (e: WheelEvent) => this.handleWheel(e);
  private onLockChange = () => this.handleLockChange();
  private onContextMenu = (e: Event) => { e.preventDefault(); };
  private onClick = (e: MouseEvent) => { this.trace('click', e); this.handleClick(e); };
  private onMouseDown = (e: MouseEvent) => { this.trace('mousedown', e); this.handleMouseDownFallback(e); };
  private onMouseUp = (e: MouseEvent) => { this.trace('mouseup', e); this.handleMouseUpFallback(e); };
  private onAuxClick = (e: MouseEvent) => this.trace('auxclick', e);

  /** Diagnostic: log every "press-like" event for the first 30 s so we
   *  can see what ChromeOS is actually emitting from a trackpad tap.
   *  Logs only when the pointer is locked (we only care about tap events
   *  during streaming, not the lock-acquiring first click). */
  private traceStartedAt = 0;
  private trace(kind: string, e: PointerEvent | MouseEvent) {
    if (!this.traceStartedAt) this.traceStartedAt = performance.now();
    if (performance.now() - this.traceStartedAt > 30000) return;
    const pe = e as PointerEvent;
    console.info(
      '[trace] %s button=%d buttons=%d pointerType=%s locked=%s ts=%dms',
      kind, e.button, e.buttons, pe.pointerType ?? '?', this.locked,
      Math.round(performance.now() - this.traceStartedAt),
    );
  }

  /** Identifier of the touch we're tracking (legacy TouchEvent path).
   *  Multi-touch is ignored — Moonlight has no multi-touch protocol. */
  private touchTrackedId: number | null = null;

  /** Set to true on first touchpad tap that surfaces as a click event so we
   *  only emit synthetic button events from clicks when pointerdown/up were
   *  absent. */
  private suppressNextClick = false;
  private suppressNextMouseDown = false;
  private suppressNextMouseUp = false;
  private firstTouchLogged = false;

  constructor(private element: HTMLElement, private client: MoonlightClient) {}

  attach() {
    // Mouse / touchpad / pen → pointer events (gives us movementX/Y under lock).
    this.element.addEventListener('pointerdown', this.onPointerDown);
    this.element.addEventListener('pointerup', this.onPointerUp);
    this.element.addEventListener('pointermove', this.onPointerMove);
    this.element.addEventListener('pointercancel', this.onPointerCancel);
    // Touchscreen → legacy touch events. On ChromeOS, pointer events with
    // pointerType='touch' stop firing under pointer lock after the first
    // tap — only a synthetic `click` is emitted with clientX/Y reporting
    // the locked-cursor position (center of the viewport). TouchEvents are
    // emitted regardless of pointer-lock state and carry the real
    // hardware coordinates.
    this.element.addEventListener('touchstart', this.onTouchStart, { passive: false });
    this.element.addEventListener('touchmove', this.onTouchMove, { passive: false });
    this.element.addEventListener('touchend', this.onTouchEnd, { passive: false });
    this.element.addEventListener('touchcancel', this.onTouchEnd, { passive: false });
    this.element.addEventListener('wheel', this.onWheel, { passive: false });
    this.element.addEventListener('contextmenu', this.onContextMenu);
    // `click` synthesised by ChromeOS for trackpad taps when pointer is
    // locked — pointerdown/pointerup are not always emitted in that case.
    this.element.addEventListener('click', this.onClick);
    // Defence in depth: some Chromebooks emit only mousedown/mouseup for
    // trackpad taps under pointer lock — no pointer events, no click. The
    // mouse-event handlers below are no-ops when pointer events already
    // fired (suppressNextMouseDown/Up gate).
    this.element.addEventListener('mousedown', this.onMouseDown);
    this.element.addEventListener('mouseup', this.onMouseUp);
    this.element.addEventListener('auxclick', this.onAuxClick);
    document.addEventListener('pointerlockchange', this.onLockChange);
    // Make sure touch gestures don't get hijacked by the browser as scroll /
    // pinch-zoom. We want every touch routed to us.
    (this.element.style as CSSStyleDeclaration).touchAction = 'none';
    this.pollHandle = window.setInterval(() => this.flush(), POLL_MS);
  }

  detach() {
    this.element.removeEventListener('pointerdown', this.onPointerDown);
    this.element.removeEventListener('pointerup', this.onPointerUp);
    this.element.removeEventListener('pointermove', this.onPointerMove);
    this.element.removeEventListener('pointercancel', this.onPointerCancel);
    this.element.removeEventListener('touchstart', this.onTouchStart);
    this.element.removeEventListener('touchmove', this.onTouchMove);
    this.element.removeEventListener('touchend', this.onTouchEnd);
    this.element.removeEventListener('touchcancel', this.onTouchEnd);
    this.element.removeEventListener('wheel', this.onWheel);
    this.element.removeEventListener('contextmenu', this.onContextMenu);
    this.element.removeEventListener('click', this.onClick);
    this.element.removeEventListener('mousedown', this.onMouseDown);
    this.element.removeEventListener('mouseup', this.onMouseUp);
    this.element.removeEventListener('auxclick', this.onAuxClick);
    document.removeEventListener('pointerlockchange', this.onLockChange);
    if (this.pollHandle) clearInterval(this.pollHandle);
    if (document.pointerLockElement) document.exitPointerLock();
  }

  // ---- pen: absolute positioning via pointer events (no TouchEvent emitted) ----

  private penDown(e: PointerEvent) {
    e.preventDefault();
    if (this.touchPointerId !== null) return;
    this.touchPointerId = e.pointerId;
    this.sendAbsolutePointer(e);
    this.client.sendMouseButton('press', BUTTON_LEFT);
  }

  private penMove(e: PointerEvent) {
    if (e.pointerId !== this.touchPointerId) return;
    e.preventDefault();
    this.sendAbsolutePointer(e);
  }

  private penUp(e: PointerEvent) {
    if (e.pointerId !== this.touchPointerId) return;
    e.preventDefault();
    this.touchPointerId = null;
    this.client.sendMouseButton('release', BUTTON_LEFT);
  }

  private sendAbsolutePointer(e: PointerEvent) {
    this.sendAbsolute(e.clientX, e.clientY);
  }

  // ---- touchscreen: legacy TouchEvent path ----

  private handleTouchStart(e: TouchEvent) {
    e.preventDefault();
    if (this.touchTrackedId !== null) return;
    if (e.changedTouches.length === 0) return;
    const t = e.changedTouches[0];
    this.touchTrackedId = t.identifier;
    if (!this.firstTouchLogged) {
      this.firstTouchLogged = true;
      const r = this.element.getBoundingClientRect();
      console.info(
        '[pointer] first touchstart: clientX=%d clientY=%d r.left=%d r.top=%d r.w=%d r.h=%d touches=%d',
        t.clientX, t.clientY, r.left, r.top, r.width, r.height, e.touches.length,
      );
    }
    this.sendTouchPosition(t);
    this.client.sendMouseButton('press', BUTTON_LEFT);
  }

  private handleTouchMove(e: TouchEvent) {
    e.preventDefault();
    for (let i = 0; i < e.changedTouches.length; i++) {
      const t = e.changedTouches[i];
      if (t.identifier === this.touchTrackedId) {
        this.sendTouchPosition(t);
        return;
      }
    }
  }

  private handleTouchEnd(e: TouchEvent) {
    e.preventDefault();
    for (let i = 0; i < e.changedTouches.length; i++) {
      const t = e.changedTouches[i];
      if (t.identifier === this.touchTrackedId) {
        this.touchTrackedId = null;
        this.client.sendMouseButton('release', BUTTON_LEFT);
        return;
      }
    }
  }

  private sendTouchPosition(t: Touch) {
    this.sendAbsolute(t.clientX, t.clientY);
  }

  /** Convert a viewport-relative (clientX, clientY) into a position inside
   *  the actual video frame area, accounting for `object-fit: contain`
   *  letterboxing. Sends in the video's natural coordinate space so the
   *  host gets pixel-accurate mapping regardless of client aspect ratio. */
  private sendAbsolute(clientX: number, clientY: number) {
    const r = this.element.getBoundingClientRect();
    // Media size: actual pixel dimensions of the source (video frame or canvas).
    let mediaW = 0;
    let mediaH = 0;
    if (this.element instanceof HTMLVideoElement) {
      mediaW = this.element.videoWidth;
      mediaH = this.element.videoHeight;
    } else if (this.element instanceof HTMLCanvasElement) {
      mediaW = this.element.width;
      mediaH = this.element.height;
    }
    if (!mediaW || !mediaH || !r.width || !r.height) {
      // No media size yet — fall back to element-relative.
      const x = Math.max(0, Math.min(r.width, clientX - r.left));
      const y = Math.max(0, Math.min(r.height, clientY - r.top));
      this.client.sendMousePosition(x, y, r.width, r.height);
      return;
    }

    // Compute the rendered video area inside the element (object-fit: contain).
    const elemAspect = r.width / r.height;
    const mediaAspect = mediaW / mediaH;
    let displayW: number, displayH: number, xOffset: number, yOffset: number;
    if (elemAspect > mediaAspect) {
      // Pillarbox (bars on the sides).
      displayH = r.height;
      displayW = r.height * mediaAspect;
      xOffset = (r.width - displayW) / 2;
      yOffset = 0;
    } else {
      // Letterbox (bars on top/bottom).
      displayW = r.width;
      displayH = r.width / mediaAspect;
      xOffset = 0;
      yOffset = (r.height - displayH) / 2;
    }

    // Position relative to the visible video area, then scale into the
    // media's native pixel space so the host sees a pixel-aligned coord.
    const px = ((clientX - r.left - xOffset) / displayW) * mediaW;
    const py = ((clientY - r.top - yOffset) / displayH) * mediaH;
    const x = Math.max(0, Math.min(mediaW, px));
    const y = Math.max(0, Math.min(mediaH, py));
    this.client.sendMousePosition(x, y, mediaW, mediaH);
  }

  // ---- mouse / touchpad: pointer lock + relative deltas ----

  private handleDown(e: PointerEvent) {
    // touch is handled by the TouchEvent path (more reliable under lock).
    if (e.pointerType === 'touch') return;
    if (e.pointerType === 'pen') { this.penDown(e); return; }
    if (!this.locked) {
      // unadjustedMovement: true asks Chrome to skip OS pointer-acceleration
      // curves. The trackpad becomes truly linear; without it, ChromeOS
      // applies its accel curve and `movementX/Y` is non-linear. Falls
      // back gracefully if the option is unsupported.
      this.tryLock();
      e.preventDefault();
      // The lock-acquisition click should not also be forwarded as a press
      // via the mouse-event fallback below.
      this.suppressNextMouseDown = true;
      this.suppressNextMouseUp = true;
      this.suppressNextClick = true;
      return;
    }
    e.preventDefault();
    // We're servicing pointerdown — make sure the click-fallback and the
    // compatibility-mouse-event fallback below don't re-emit press/release
    // for the same gesture.
    this.suppressNextClick = true;
    this.suppressNextMouseDown = true;
    this.client.sendMouseButton('press', mapButton(e.button));
  }

  private handleUp(e: PointerEvent) {
    if (e.pointerType === 'touch') return;
    if (e.pointerType === 'pen') { this.penUp(e); return; }
    if (!this.locked) return;
    e.preventDefault();
    this.suppressNextClick = true;
    this.suppressNextMouseUp = true;
    this.client.sendMouseButton('release', mapButton(e.button));
  }

  private handleMouseDownFallback(e: MouseEvent) {
    if (this.suppressNextMouseDown) {
      this.suppressNextMouseDown = false;
      return;
    }
    if (!this.locked) return;
    e.preventDefault();
    console.info('[pointer] fallback mousedown → press (button=%d)', e.button);
    this.client.sendMouseButton('press', mapButton(e.button));
    // The matching click will fire after mouseup; pre-suppress it.
    this.suppressNextClick = true;
  }

  private handleMouseUpFallback(e: MouseEvent) {
    if (this.suppressNextMouseUp) {
      this.suppressNextMouseUp = false;
      return;
    }
    if (!this.locked) return;
    e.preventDefault();
    console.info('[pointer] fallback mouseup → release (button=%d)', e.button);
    this.client.sendMouseButton('release', mapButton(e.button));
  }

  private handleMove(e: PointerEvent) {
    if (e.pointerType === 'touch') return;
    if (e.pointerType === 'pen') { this.penMove(e); return; }
    if (!this.locked) return;
    this.dx += e.movementX;
    this.dy += e.movementY;
  }

  private handleCancel(e: PointerEvent) {
    if (e.pointerType === 'pen') { this.penUp(e); return; }
  }

  private handleWheel(e: WheelEvent) {
    if (!this.locked && this.touchPointerId === null) return;
    e.preventDefault();
    // deltaMode 0 = pixels, 1 = lines, 2 = pages. Convert to ticks
    // (1 tick = 120 in moonlight's high-res scroll).
    const lines = e.deltaMode === 0 ? -e.deltaY / 100 : -e.deltaY;
    this.wheelTicks += lines;
  }

  private handleLockChange() {
    this.locked = document.pointerLockElement === this.element;
    console.info('[pointer] lock change: locked=%s', this.locked);
  }

  private handleClick(e: MouseEvent) {
    // Touchpad taps under pointer lock often surface ONLY as a `click`
    // event on ChromeOS — no pointerdown/pointerup pair. Synthesise a
    // press+release so the host sees a left click. Skip if we already
    // observed pointerdown/pointerup for this gesture (suppressNextClick),
    // or if a touchscreen interaction is in progress (TouchEvents own that
    // path and a synthesised click here would re-emit at the wrong place).
    if (this.suppressNextClick) {
      this.suppressNextClick = false;
      return;
    }
    if (this.touchTrackedId !== null) return;
    if (!this.locked) return;
    e.preventDefault();
    console.info('[pointer] synthetic click → press/release (button=%d)', e.button);
    this.client.sendMouseButton('press', mapButton(e.button));
    this.client.sendMouseButton('release', mapButton(e.button));
  }

  private tryLock() {
    type LockOpts = { unadjustedMovement?: boolean };
    type LockReturning = HTMLElement & {
      requestPointerLock: (opts?: LockOpts) => Promise<void> | void;
    };
    const el = this.element as LockReturning;
    let p: Promise<void> | void;
    try {
      p = el.requestPointerLock({ unadjustedMovement: true });
    } catch (err) {
      console.warn('[pointer] requestPointerLock({unadjustedMovement:true}) threw:', err);
      p = undefined;
    }
    if (p && typeof (p as Promise<void>).then === 'function') {
      (p as Promise<void>)
        .then(() => console.info('[pointer] pointer lock acquired with unadjustedMovement=true (raw deltas)'))
        .catch((err) => {
          console.warn('[pointer] unadjustedMovement rejected (%s); falling back to standard lock', err?.message ?? err);
          try { this.element.requestPointerLock(); } catch { /* ignore */ }
        });
    } else if (!p) {
      console.info('[pointer] requestPointerLock returned no promise; trying plain lock');
      try { this.element.requestPointerLock(); } catch { /* ignore */ }
    } else {
      console.info('[pointer] requestPointerLock with options accepted (no Promise returned)');
    }
  }

  private flush() {
    if (this.dx !== 0 || this.dy !== 0) {
      this.client.sendMouseMove(this.dx, this.dy);
      this.dx = this.dy = 0;
    }
    if (this.wheelTicks !== 0) {
      this.client.sendScroll(Math.round(this.wheelTicks * 120));
      this.wheelTicks = 0;
    }
  }
}

function mapButton(b: number): 1 | 2 | 3 | 4 | 5 {
  switch (b) {
    case 0: return BUTTON_LEFT;
    case 1: return BUTTON_MIDDLE;
    case 2: return BUTTON_RIGHT;
    case 3: return BUTTON_X1;
    case 4: return BUTTON_X2;
    default: return BUTTON_LEFT;
  }
}
