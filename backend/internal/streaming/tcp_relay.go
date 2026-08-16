package streaming

import (
	"context"
	"io"
	"log"
	"net"
	"time"
)

const tcpConnectTimeout = 5 * time.Second

// tcpRelay 是单个 TCP 通道的中继。连接目标 host:port 后双向镜像：
// 读侧把 TCP 数据帧转发回客户端，写侧把客户端数据写入 TCP。
// 远端 EOF 时立即发 CLOSED 帧（RTSP 代码把空包当 EOF）。
type tcpRelay struct {
	channel byte
	conn    net.Conn
	out     func(frame []byte)
	closed  func(reason byte)
	writeCh chan []byte
	cancel  context.CancelFunc
}

// startTCPRelay 启动 TCP 中继，返回客户端数据写入通道与 relay 退出通知。
func startTCPRelay(
	ctx context.Context,
	channel byte,
	host string,
	port uint16,
	out func(frame []byte),
	closed func(reason byte),
) (chan []byte, <-chan struct{}) {
	writeCh := make(chan []byte, 128)
	done := make(chan struct{})

	conn, err := net.DialTimeout("tcp", net.JoinHostPort(host, portStr(port)), tcpConnectTimeout)
	if err != nil {
		log.Printf("[stream] tcp ch=%d connect %s:%d failed: %v", channel, host, port, err)
		closed(closeReasonConnectFail)
		close(done)
		return writeCh, done
	}
	if tcpConn, ok := conn.(*net.TCPConn); ok {
		_ = tcpConn.SetNoDelay(true)
	}

	relayCtx, cancel := context.WithCancel(ctx)
	relay := &tcpRelay{
		channel: channel,
		conn:    conn,
		out:     out,
		closed:  closed,
		writeCh: writeCh,
		cancel:  cancel,
	}

	go func() {
		defer close(done)
		defer cancel()
		defer func() { _ = conn.Close() }()
		// readLoop 与 writeLoop 必须并发：readLoop 阻塞在 conn.Read 等远端
		// 数据时，writeLoop 仍要转发客户端数据出去（否则首包即卡死）。
		go relay.readLoop(relayCtx)
		relay.writeLoop(relayCtx)
		relay.closed(closeReasonNormal)
	}()
	return writeCh, done
}

// readLoop 转发 TCP 数据到客户端。EOF 或错误时取消写循环并退出，
// 触发 CLOSED 帧通知（RTSP 代码把空包当 EOF，需要远端关闭及时上送）。
func (r *tcpRelay) readLoop(ctx context.Context) {
	defer r.cancel()
	buf := make([]byte, 65536)
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		n, err := r.conn.Read(buf)
		if err != nil {
			if err != io.EOF {
				log.Printf("[stream] tcp ch=%d read error: %v", r.channel, err)
			}
			return
		}
		frame := make([]byte, 4+n)
		frame[0] = opData
		frame[1] = r.channel
		copy(frame[4:], buf[:n])
		r.out(frame)
	}
}

// writeLoop 转发客户端数据到 TCP。写入通道关闭或上下文取消即退出。
func (r *tcpRelay) writeLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case data, ok := <-r.writeCh:
			if !ok {
				return
			}
			if _, err := r.conn.Write(data); err != nil {
				log.Printf("[stream] tcp ch=%d write error: %v", r.channel, err)
				return
			}
		}
	}
}
