package streaming

import (
	"context"
	"log"
	"strconv"
	"sync"

	"github.com/gorilla/websocket"
)

// session 是一个 WebSocket 连接上的多路复用会话。每个通道对应一个
// UDP/TCP relay；客户端帧被分发到通道，relay 回发帧经 writer goroutine
// 写回 WebSocket。
type session struct {
	ws         *websocket.Conn
	writeMu    sync.Mutex
	outFrames  chan []byte
	outMu      sync.Mutex
	outClosed  bool
	maxChannel byte

	mu       sync.Mutex
	channels map[byte]*channelEntry
}

type channelEntry struct {
	writeCh chan []byte
	done    <-chan struct{}
}

// runSession 处理一个 WebSocket 连接直到关闭，随后清理全部通道。
func runSession(ctx context.Context, ws *websocket.Conn, maxChannels byte) {
	s := &session{
		ws:         ws,
		outFrames:  make(chan []byte, 256),
		maxChannel: maxChannels,
		channels:   make(map[byte]*channelEntry),
	}

	var writerDone sync.WaitGroup
	writerDone.Add(1)
	go func() {
		defer writerDone.Done()
		for frame := range s.outFrames {
			if err := ws.WriteMessage(websocket.BinaryMessage, frame); err != nil {
				break
			}
		}
	}()

	readErr := s.readLoop(ctx)

	// 会话结束：关闭全部通道，再关闭 outFrames 通知 writer 退出。
	s.closeAll()
	s.closeOutFrames()
	writerDone.Wait()
	_ = ws.Close()

	if readErr != nil && !websocket.IsCloseError(readErr, websocket.CloseNormalClosure) {
		log.Printf("[stream] ws session error: %v", readErr)
	}
}

// readLoop 逐条读取二进制帧并分发。每条 WS 消息即一帧。
func (s *session) readLoop(ctx context.Context) error {
	for {
		msgType, data, err := s.ws.ReadMessage()
		if err != nil {
			return err
		}
		if msgType != websocket.BinaryMessage {
			continue
		}
		if len(data) < 4 {
			log.Printf("[stream] ws frame too short: %d bytes", len(data))
			continue
		}
		op := data[0]
		channel := data[1]
		payload := data[4:]
		switch op {
		case opOpen:
			s.handleOpen(channel, payload)
		case opClose:
			s.handleClose(channel)
		case opData:
			s.handleData(channel, payload)
		default:
			log.Printf("[stream] unknown op %d on ch=%d", op, channel)
		}
	}
}

func (s *session) handleOpen(channel byte, payload []byte) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if channel == channelControl {
		log.Printf("[stream] client tried to open reserved channel 0")
		s.sendFrame(encodeClosed(channel, closeReasonBadProto))
		return
	}
	if len(s.channels) >= int(s.maxChannel) {
		log.Printf("[stream] channel limit %d reached", s.maxChannel)
		s.sendFrame(encodeClosed(channel, closeReasonLimit))
		return
	}
	if _, exists := s.channels[channel]; exists {
		log.Printf("[stream] duplicate open for ch=%d", channel)
		s.sendFrame(encodeClosed(channel, closeReasonBadProto))
		return
	}

	open, err := parseOpenPayload(payload)
	if err != nil {
		log.Printf("[stream] malformed OPEN payload on ch=%d: %v", channel, err)
		s.sendFrame(encodeClosed(channel, closeReasonBadProto))
		return
	}

	log.Printf("[stream] open ch=%d proto=%d host=%s:%d", channel, open.proto, open.host, open.port)

	var writeCh chan []byte
	var done <-chan struct{}
	closed := func(reason byte) { s.relayClosed(channel, reason) }
	switch open.proto {
	case protoUDP:
		writeCh, done = startUDPRelay(context.Background(), channel, open.host, open.port, s.sendFrame, closed)
	case protoTCP:
		writeCh, done = startTCPRelay(context.Background(), channel, open.host, open.port, s.sendFrame, closed)
	default:
		log.Printf("[stream] unknown proto %d on ch=%d", open.proto, channel)
		s.sendFrame(encodeClosed(channel, closeReasonBadProto))
		return
	}
	s.channels[channel] = &channelEntry{writeCh: writeCh, done: done}
}

func (s *session) handleClose(channel byte) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if entry, ok := s.channels[channel]; ok {
		delete(s.channels, channel)
		close(entry.writeCh)
		log.Printf("[stream] close ch=%d (initiated by client)", channel)
	}
}

func (s *session) handleData(channel byte, payload []byte) {
	s.mu.Lock()
	entry, ok := s.channels[channel]
	s.mu.Unlock()
	if !ok {
		log.Printf("[stream] data for unknown ch=%d, dropping", channel)
		return
	}
	select {
	case entry.writeCh <- payload:
	default:
		// 写满说明 relay 或对端慢：丢帧等价于 UDP 丢包，比阻塞读循环好。
		log.Printf("[stream] data for ch=%d dropped (relay busy)", channel)
	}
}

// relayClosed 由 relay 在退出时调用（sendFrame 线程安全，此处不持锁）。
func (s *session) relayClosed(channel byte, reason byte) {
	s.mu.Lock()
	if entry, ok := s.channels[channel]; ok {
		delete(s.channels, channel)
		close(entry.writeCh)
	}
	s.mu.Unlock()
	s.sendFrame(encodeClosed(channel, reason))
}

func (s *session) sendFrame(frame []byte) {
	// 与 closeOutFrames 互斥：outFrames 关闭后不再发送。
	s.outMu.Lock()
	defer s.outMu.Unlock()
	if s.outClosed {
		return
	}
	select {
	case s.outFrames <- frame:
	default:
		// writer 积压：会话正在下行或已死，丢弃。
		log.Printf("[stream] outFrames full, dropping frame")
	}
}

// closeOutFrames 幂等关闭发送队列。
func (s *session) closeOutFrames() {
	s.outMu.Lock()
	defer s.outMu.Unlock()
	if s.outClosed {
		return
	}
	s.outClosed = true
	close(s.outFrames)
}

func (s *session) closeAll() {
	s.mu.Lock()
	defer s.mu.Unlock()
	for channel, entry := range s.channels {
		delete(s.channels, channel)
		close(entry.writeCh)
		log.Printf("[stream] force-closing ch=%d", channel)
	}
}

func portStr(port uint16) string {
	return strconv.Itoa(int(port))
}
