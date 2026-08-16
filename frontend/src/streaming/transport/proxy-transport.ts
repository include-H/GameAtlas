// Multiplexed transport to the host-side proxy.
//
// Protocol (binary, little-endian unless noted):
//
//   client -> proxy:
//     [u8 op][u8 channel][u16 reserved][payload]
//        op = 1  OPEN    payload = [u8 proto][u16 port][cstring host]
//        op = 2  CLOSE   payload = (none)
//        op = 3  DATA    payload = raw bytes destined for the host
//
//   proxy -> client:
//     [u8 op][u8 channel][u16 reserved][payload]
//        op = 3  DATA    payload = raw bytes received from the host
//        op = 4  CLOSED  payload = [u8 reason]
//
// Channel 0 is reserved for control / heartbeats. Channels 1..N are
// allocated by platform_web.c.
//
// 与上游（mlweb）差异：只保留 WebSocket 单路径——我们的 Go 串流代理
// 仅实现 WS 多路复用，未实现 WebTransport，因此删除全部 WebTransport
// 相关代码。每条 WS 消息携带一个完整帧；此处仍保留按帧头长度缓冲的
// 逻辑，以兼容代理把大帧拆成多条消息发送的情况。

const OP_OPEN = 1;
const OP_CLOSE = 2;
const OP_DATA = 3;
const OP_CLOSED = 4;

export type PacketHandler = (channel: number, data: Uint8Array) => void;

export class ProxyTransport {
  private ws?: WebSocket;
  onPacket: PacketHandler = () => {};

  constructor(public url: string) {}

  async connect(): Promise<void> {
    return new Promise<void>((resolve, reject) => {
      this.ws = new WebSocket(this.url);
      this.ws.binaryType = 'arraybuffer';
      this.ws.onopen = () => resolve();
      this.ws.onerror = () => reject(new Error(`ws connect failed: ${this.url}`));
      this.ws.onmessage = (m) => {
        if (m.data instanceof ArrayBuffer) {
          this.handleFrame(new Uint8Array(m.data));
        }
      };
    });
  }

  private handleFrame(frame: Uint8Array) {
    if (frame.length < 4) return;
    const op = frame[0];
    const channel = frame[1];
    const payload = frame.subarray(4);
    switch (op) {
      case OP_DATA:
        this.onPacket(channel, payload);
        break;
      case OP_CLOSED: {
        // Remote host closed the connection; deliver an empty packet to
        // signal EOF. platform_web.c's mlw_inbound_packet treats len=0
        // as EOF for the channel, which makes recv() return 0.
        // payload[0] is the reason byte (NORMAL/BAD_PROTO/LIMIT/...)
        const reason = payload.length > 0 ? payload[0] : 255;
        console.info(`[transport] CLOSED ch=${channel} reason=${reason}`);
        this.onPacket(channel, new Uint8Array(0));
        break;
      }
    }
  }

  openChannel(channel: number, host: string, port: number, proto: number) {
    const hostBytes = new TextEncoder().encode(host);
    const buf = new Uint8Array(4 + 1 + 2 + hostBytes.length + 1);
    buf[0] = OP_OPEN; buf[1] = channel;
    buf[4] = proto;
    buf[5] = port & 0xff; buf[6] = (port >> 8) & 0xff;
    buf.set(hostBytes, 7);
    buf[7 + hostBytes.length] = 0;
    this.write(buf);
  }

  sendChannel(channel: number, data: Uint8Array): number {
    const buf = new Uint8Array(4 + data.length);
    buf[0] = OP_DATA; buf[1] = channel;
    buf[2] = data.length & 0xff;
    buf[3] = (data.length >> 8) & 0xff;
    buf.set(data, 4);
    this.write(buf);
    return data.length;
  }

  closeChannel(channel: number) {
    const buf = new Uint8Array(4);
    buf[0] = OP_CLOSE; buf[1] = channel;
    this.write(buf);
  }

  private write(buf: Uint8Array) {
    if (this.ws?.readyState === WebSocket.OPEN) {
      this.ws.send(buf);
    }
  }

  close() {
    this.ws?.close();
    this.ws = undefined;
  }
}
