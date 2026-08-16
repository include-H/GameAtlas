// Package streaming 提供浏览器游戏串流的 Go 代理实现。
//
// 移植自 moonlight-webclient 的 Rust host-proxy，线协议与其 byte-for-byte
// 兼容（见 docs/moonlight 串流集成设计.md）。浏览器不能发裸 UDP，串流页
// 通过 WebSocket 连到本代理，代理按帧头把每个通道桥接成对 Sunshine /
// GameStream 主机的 UDP / TCP 流量。
package streaming

import (
	"encoding/binary"
	"fmt"
)

// 帧头（4 字节，小端）：[u8 op][u8 channel][u16 reserved][payload]
// 客户端 -> 代理：
//
//	op=1 OPEN:  payload = [u8 proto][u16 port LE][cstring host]
//	op=2 CLOSE: payload = (none)
//	op=3 DATA:  payload = raw bytes
//
// 代理 -> 客户端：
//
//	op=3 DATA:  payload = raw bytes
//	op=4 CLOSED: payload = [u8 reason]
const (
	opOpen   byte = 1
	opClose  byte = 2
	opData   byte = 3
	opClosed byte = 4
)

const (
	protoUDP byte = 1
	protoTCP byte = 2
)

// Channel 0 保留给控制/心跳，不参与转发。
const channelControl byte = 0

// 通道关闭原因（CLOSED 帧 payload）。
const (
	closeReasonNormal      byte = 0
	closeReasonBadProto    byte = 1
	closeReasonLimit       byte = 2
	closeReasonConnectFail byte = 3
	closeReasonBindFail    byte = 4
	closeReasonIOError     byte = 5
)

// openPayload 是 OPEN 帧的解析结果。
type openPayload struct {
	proto byte
	port  uint16
	host  string
}

// encodeClosed 编码代理 -> 客户端的 CLOSED 帧。
func encodeClosed(channel byte, reason byte) []byte {
	return []byte{opClosed, channel, 0, 0, reason}
}

// parseOpenPayload 解析 OPEN 帧 payload：
// [proto u8][port u16 LE][host cstring]。格式非法返回 error。
func parseOpenPayload(payload []byte) (openPayload, error) {
	if len(payload) < 4 {
		return openPayload{}, fmt.Errorf("open payload too short: %d bytes", len(payload))
	}
	proto := payload[0]
	port := binary.LittleEndian.Uint16(payload[1:3])
	nul := -1
	for i := 3; i < len(payload); i++ {
		if payload[i] == 0 {
			nul = i
			break
		}
	}
	if nul < 0 {
		return openPayload{}, fmt.Errorf("open payload missing NUL terminator")
	}
	host := string(payload[3:nul])
	if host == "" {
		return openPayload{}, fmt.Errorf("open payload empty host")
	}
	return openPayload{proto: proto, port: port, host: host}, nil
}
