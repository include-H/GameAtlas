// Gamepad polling. The Gamepad API doesn't deliver events, so we poll
// once per animation frame.
//
// We mirror the moonlight controller button mapping from upstream
// gamepad.cpp:
//
//   buttons[0]  A         -> A_FLAG
//   buttons[1]  B         -> B_FLAG
//   buttons[2]  X         -> X_FLAG
//   buttons[3]  Y         -> Y_FLAG
//   buttons[4]  LB        -> LB_FLAG
//   buttons[5]  RB        -> RB_FLAG
//   buttons[6]  LT (axis) -> leftTrigger (0-255)
//   buttons[7]  RT (axis) -> rightTrigger (0-255)
//   buttons[8]  Back      -> BACK_FLAG
//   buttons[9]  Start     -> PLAY_FLAG
//   buttons[10] LSB       -> LS_CLK_FLAG
//   buttons[11] RSB       -> RS_CLK_FLAG
//   buttons[12] DPad Up   -> UP_FLAG
//   buttons[13] DPad Down -> DOWN_FLAG
//   buttons[14] DPad Lft  -> LEFT_FLAG
//   buttons[15] DPad Rgt  -> RIGHT_FLAG
//   buttons[16] Guide     -> SPECIAL_FLAG

import type { MoonlightClient } from '../client/moonlight-client';

const A_FLAG       = 0x1000;
const B_FLAG       = 0x2000;
const X_FLAG       = 0x4000;
const Y_FLAG       = 0x8000;
const LB_FLAG      = 0x0100;
const RB_FLAG      = 0x0200;
const BACK_FLAG    = 0x0020;
const PLAY_FLAG    = 0x0010;
const LS_CLK_FLAG  = 0x0040;
const RS_CLK_FLAG  = 0x0080;
const UP_FLAG      = 0x0001;
const DOWN_FLAG    = 0x0002;
const LEFT_FLAG    = 0x0004;
const RIGHT_FLAG   = 0x0008;
const SPECIAL_FLAG = 0x0400;

const BUTTON_MAP = [
  A_FLAG, B_FLAG, X_FLAG, Y_FLAG,
  LB_FLAG, RB_FLAG,
  0, 0, // triggers handled via axis
  BACK_FLAG, PLAY_FLAG,
  LS_CLK_FLAG, RS_CLK_FLAG,
  UP_FLAG, DOWN_FLAG, LEFT_FLAG, RIGHT_FLAG,
  SPECIAL_FLAG,
];

export class GamepadInput {
  private running = false;
  private lastTimestamps = new Map<number, number>();
  private announced = new Set<number>();

  constructor(private client: MoonlightClient) {}

  start() {
    if (this.running) return;
    this.running = true;
    requestAnimationFrame(this.tick);
    window.addEventListener('gamepadconnected', this.onConnect);
    window.addEventListener('gamepaddisconnected', this.onDisconnect);
  }

  stop() {
    this.running = false;
    window.removeEventListener('gamepadconnected', this.onConnect);
    window.removeEventListener('gamepaddisconnected', this.onDisconnect);
  }

  private onConnect = (e: GamepadEvent) => {
    if (this.announced.has(e.gamepad.index)) return;
    this.announced.add(e.gamepad.index);
    // type=0 unknown, 1 xbox, 2 ps, 3 nintendo - we forward as unknown.
    // supportedButtons mask = 0xFFFF (all), capabilities = rumble (0x01).
    this.client.sendControllerArrival(e.gamepad.index, 0, 0xFFFF, 0x01);
  };

  private onDisconnect = (e: GamepadEvent) => {
    this.announced.delete(e.gamepad.index);
    this.lastTimestamps.delete(e.gamepad.index);
  };

  private tick = () => {
    if (!this.running) return;
    const pads = navigator.getGamepads();
    for (const pad of pads) {
      if (!pad) continue;
      if (this.lastTimestamps.get(pad.index) === pad.timestamp) continue;
      this.lastTimestamps.set(pad.index, pad.timestamp);
      this.send(pad);
    }
    requestAnimationFrame(this.tick);
  };

  private send(pad: Gamepad) {
    let buttons = 0;
    for (let i = 0; i < BUTTON_MAP.length && i < pad.buttons.length; i++) {
      if (pad.buttons[i]?.pressed) buttons |= BUTTON_MAP[i];
    }
    const lt = Math.round((pad.buttons[6]?.value ?? 0) * 255);
    const rt = Math.round((pad.buttons[7]?.value ?? 0) * 255);
    // Axis range: gamepad -1..1 -> int16 -32768..32767. Y inverted.
    const ax = (v: number) => Math.max(-32768, Math.min(32767, Math.round(v * 32767)));
    const lsx = ax(pad.axes[0] ?? 0);
    const lsy = ax(-(pad.axes[1] ?? 0));
    const rsx = ax(pad.axes[2] ?? 0);
    const rsy = ax(-(pad.axes[3] ?? 0));
    this.client.sendController({
      index: pad.index,
      buttons,
      leftTrigger: lt,
      rightTrigger: rt,
      leftStickX: lsx,
      leftStickY: lsy,
      rightStickX: rsx,
      rightStickY: rsy,
    });
  }
}
