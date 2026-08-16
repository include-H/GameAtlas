package streaming

import (
	"context"
	"log"
	"net"
	"time"
)

// udpRelay 是单个 UDP 通道的中继。绑定一个临时 UDP socket，把客户端经
// WebSocket 发来的数据 send_to 到目标主机，同时把收到的数据帧转发回客户端。
//
// 刻意不 connect()：Windows 上已连接 UDP socket 会把 ICMP unreachable
// 变成 WSAECONNRESET 杀掉 recv 循环（上游注释，保留同款行为）。
type udpRelay struct {
	channel byte
	target  *net.UDPAddr
	socket  *net.UDPConn
	out     func(frame []byte) // 帧回发客户端
	closed  func(reason byte)
	writeCh chan []byte
	cancel  context.CancelFunc
}

// startUDPRelay 启动 UDP 中继，返回客户端数据写入通道与 socket 关闭通知。
func startUDPRelay(
	ctx context.Context,
	channel byte,
	host string,
	port uint16,
	out func(frame []byte),
	closed func(reason byte),
) (chan []byte, <-chan struct{}) {
	writeCh := make(chan []byte, 128)
	done := make(chan struct{})

	target, err := net.ResolveUDPAddr("udp", net.JoinHostPort(host, portStr(port)))
	if err != nil {
		log.Printf("[stream] udp ch=%d resolve %s:%d failed: %v", channel, host, port, err)
		closed(closeReasonConnectFail)
		close(done)
		return writeCh, done
	}

	socket, err := net.ListenUDP("udp", &net.UDPAddr{})
	if err != nil {
		log.Printf("[stream] udp ch=%d bind failed: %v", channel, err)
		closed(closeReasonBindFail)
		close(done)
		return writeCh, done
	}
	_ = socket.SetReadBuffer(256 * 1024)
	_ = socket.SetWriteBuffer(256 * 1024)

	relayCtx, cancel := context.WithCancel(ctx)
	relay := &udpRelay{
		channel: channel,
		target:  target,
		socket:  socket,
		out:     out,
		closed:  closed,
		writeCh: writeCh,
		cancel:  cancel,
	}

	go relay.readLoop(relayCtx)
	go func() {
		defer close(done)
		defer cancel()
		defer func() { _ = socket.Close() }()
		relay.writeLoop(relayCtx)
		relay.closed(closeReasonNormal)
	}()
	return writeCh, done
}

// readLoop 读取 UDP 数据并帧转发回客户端。错误不拆通道（Linux 上
// ECONNREFUSED 等瞬时错误常见），由上下文取消作为唯一退出途径。
func (r *udpRelay) readLoop(ctx context.Context) {
	buf := make([]byte, 65535)
	for {
		n, _, err := r.socket.ReadFromUDP(buf)
		if err != nil {
			select {
			case <-ctx.Done():
				return
			default:
			}
			log.Printf("[stream] udp ch=%d recv error: %v (continuing)", r.channel, err)
			// ICMP unreachable 可能让下一次读立刻失败，稍歇避免忙循环。
			time.Sleep(10 * time.Millisecond)
			continue
		}
		frame := make([]byte, 4+n)
		frame[0] = opData
		frame[1] = r.channel
		copy(frame[4:], buf[:n])
		r.out(frame)
	}
}

// writeLoop 转发客户端数据到 UDP socket。写入通道关闭或上下文取消即退出。
func (r *udpRelay) writeLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case data, ok := <-r.writeCh:
			if !ok {
				return
			}
			if _, err := r.socket.WriteToUDP(data, r.target); err != nil {
				log.Printf("[stream] udp ch=%d send error: %v", r.channel, err)
				return
			}
		}
	}
}
