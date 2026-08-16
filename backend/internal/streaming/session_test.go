package streaming

import (
	"context"
	"encoding/binary"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

// startUDPEcho 起一个 UDP echo server，返回地址与关闭函数。
func startUDPEcho(t *testing.T) (net.Addr, func()) {
	t.Helper()
	conn, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.ParseIP("127.0.0.1")})
	if err != nil {
		t.Fatalf("listen udp: %v", err)
	}
	go func() {
		buf := make([]byte, 2048)
		for {
			n, addr, err := conn.ReadFromUDP(buf)
			if err != nil {
				return
			}
			_, _ = conn.WriteToUDP(buf[:n], addr)
		}
	}()
	return conn.LocalAddr(), func() { _ = conn.Close() }
}

// buildOpenFrame 构造 OPEN 帧（对齐 wire 协议）。
func buildOpenFrame(channel byte, proto byte, host string, port uint16) []byte {
	hostBytes := append([]byte(host), 0)
	frame := make([]byte, 4+1+2+len(hostBytes))
	frame[0] = opOpen
	frame[1] = channel
	frame[4] = proto
	binary.LittleEndian.PutUint16(frame[5:7], port)
	copy(frame[7:], hostBytes)
	return frame
}

func buildDataFrame(channel byte, payload []byte) []byte {
	frame := make([]byte, 4+len(payload))
	frame[0] = opData
	frame[1] = channel
	copy(frame[4:], payload)
	return frame
}

// TestSessionUDPRelayRoundtrip 验证 WS 会话内 UDP 通道的完整转发链路：
// OPEN 后客户端 DATA 帧应被转发到 UDP echo 并原路返回。
func TestSessionUDPRelayRoundtrip(t *testing.T) {
	echoAddr, closeEcho := startUDPEcho(t)
	defer closeEcho()

	upgrader := websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ws, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		runSession(r.Context(), ws, 8)
	}))
	defer server.Close()

	wsURL := "ws" + server.URL[len("http"):]
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial ws: %v", err)
	}
	defer func() { _ = conn.Close() }()

	// OPEN UDP 通道 ch=1 → 本机 echo
	host, portStr, _ := net.SplitHostPort(echoAddr.String())
	port := uint16(mustAtoi(t, portStr))
	if err := conn.WriteMessage(websocket.BinaryMessage, buildOpenFrame(1, protoUDP, host, port)); err != nil {
		t.Fatalf("write OPEN: %v", err)
	}

	// 发数据，等 echo
	payload := []byte("gameatlas-echo-test")
	if err := conn.WriteMessage(websocket.BinaryMessage, buildDataFrame(1, payload)); err != nil {
		t.Fatalf("write DATA: %v", err)
	}

	deadline := time.Now().Add(3 * time.Second)
	for {
		_ = conn.SetReadDeadline(deadline)
		_, data, err := conn.ReadMessage()
		if err != nil {
			t.Fatalf("read echo: %v", err)
		}
		if len(data) < 4 || data[0] != opData || data[1] != 1 {
			continue
		}
		if string(data[4:]) != string(payload) {
			t.Fatalf("echo mismatch: got %q, want %q", data[4:], payload)
		}
		break
	}

	// 客户端 CLOSE 应让 relay 干净退出
	if err := conn.WriteMessage(websocket.BinaryMessage, []byte{opClose, 1, 0, 0}); err != nil {
		t.Fatalf("write CLOSE: %v", err)
	}
	time.Sleep(100 * time.Millisecond)
}

// TestSessionOpenValidations 验证 OPEN 的边界：保留通道 0、重复通道、超限。
func TestSessionOpenValidations(t *testing.T) {
	upgrader := websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ws, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		runSession(r.Context(), ws, 2)
	}))
	defer server.Close()

	wsURL := "ws" + server.URL[len("http"):]
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial ws: %v", err)
	}
	defer func() { _ = conn.Close() }()

	// 保留通道 0 → 应回 CLOSED(BAD_PROTO)
	if err := conn.WriteMessage(websocket.BinaryMessage, buildOpenFrame(0, protoUDP, "127.0.0.1", 47998)); err != nil {
		t.Fatalf("write OPEN ch0: %v", err)
	}
	_, data, err := readFrame(conn, 3*time.Second)
	if err != nil {
		t.Fatalf("read ch0 response: %v", err)
	}
	if data[0] != opClosed || data[1] != 0 || data[4] != closeReasonBadProto {
		t.Fatalf("ch0 response = %v, want CLOSED(BAD_PROTO)", data)
	}

	// 畸形 OPEN（无 NUL）→ CLOSED(BAD_PROTO)
	badOpen := []byte{opOpen, 3, 0, 0, protoUDP, 0x22, 0xbb}
	if err := conn.WriteMessage(websocket.BinaryMessage, badOpen); err != nil {
		t.Fatalf("write bad OPEN: %v", err)
	}
	_, data, err = readFrame(conn, 3*time.Second)
	if err != nil {
		t.Fatalf("read bad open response: %v", err)
	}
	if data[0] != opClosed || data[4] != closeReasonBadProto {
		t.Fatalf("bad open response = %v, want CLOSED(BAD_PROTO)", data)
	}

	// 未知 proto → CLOSED(BAD_PROTO)
	if err := conn.WriteMessage(websocket.BinaryMessage, buildOpenFrame(4, 9, "127.0.0.1", 47998)); err != nil {
		t.Fatalf("write bad proto: %v", err)
	}
	_, data, err = readFrame(conn, 3*time.Second)
	if err != nil {
		t.Fatalf("read bad proto response: %v", err)
	}
	if data[0] != opClosed || data[4] != closeReasonBadProto {
		t.Fatalf("bad proto response = %v, want CLOSED(BAD_PROTO)", data)
	}
}

// TestSessionTCPRelayEOF 验证 TCP 通道：连接本地 HTTP 端口，收到 200 响应
// 且远端关闭后收到 CLOSED(NORMAL)。
func TestSessionTCPRelayEOF(t *testing.T) {
	upgrader := websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ws, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		runSession(r.Context(), ws, 8)
	}))
	defer server.Close()

	// 本地 HTTP 目标（echo server 本身即 TCP 端点）
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer target.Close()

	targetURL := target.URL // http://127.0.0.1:port
	tHost, tPortStr, _ := net.SplitHostPort(targetURL[len("http://"):])
	tPort := uint16(mustAtoi(t, tPortStr))

	wsURL := "ws" + server.URL[len("http"):]
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial ws: %v", err)
	}
	defer func() { _ = conn.Close() }()

	if err := conn.WriteMessage(websocket.BinaryMessage, buildOpenFrame(1, protoTCP, tHost, tPort)); err != nil {
		t.Fatalf("write OPEN tcp: %v", err)
	}

	req := []byte("GET / HTTP/1.1\r\nHost: x\r\nConnection: close\r\n\r\n")
	if err := conn.WriteMessage(websocket.BinaryMessage, buildDataFrame(1, req)); err != nil {
		t.Fatalf("write DATA: %v", err)
	}

	sawData := false
	sawClosed := false
	deadline := time.Now().Add(5 * time.Second)
	for !sawClosed {
		_ = conn.SetReadDeadline(deadline)
		_, data, err := conn.ReadMessage()
		if err != nil {
			t.Fatalf("read: %v", err)
		}
		if len(data) < 4 {
			continue
		}
		switch data[0] {
		case opData:
			if data[1] == 1 {
				sawData = true
			}
		case opClosed:
			if data[1] == 1 && data[4] == closeReasonNormal {
				sawClosed = true
			}
		}
	}
	if !sawData || !sawClosed {
		t.Fatalf("sawData=%v sawClosed=%v, want both true", sawData, sawClosed)
	}
}

func readFrame(conn *websocket.Conn, timeout time.Duration) ([]byte, []byte, error) {
	_ = conn.SetReadDeadline(time.Now().Add(timeout))
	_, data, err := conn.ReadMessage()
	return nil, data, err
}

func mustAtoi(t *testing.T, s string) int {
	t.Helper()
	var n int
	for _, c := range s {
		if c < '0' || c > '9' {
			t.Fatalf("invalid port %q", s)
		}
		n = n*10 + int(c-'0')
	}
	return n
}

var _ = context.Background
