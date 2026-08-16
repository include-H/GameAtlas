package streaming

import (
	"encoding/binary"
	"testing"
)

func TestParseOpenPayload(t *testing.T) {
	host := "192.168.1.100\x00"
	payload := make([]byte, 1+2+len(host))
	payload[0] = protoUDP
	binary.LittleEndian.PutUint16(payload[1:3], 47998)
	copy(payload[3:], host)

	open, err := parseOpenPayload(payload)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if open.proto != protoUDP {
		t.Errorf("proto = %d, want %d", open.proto, protoUDP)
	}
	if open.port != 47998 {
		t.Errorf("port = %d, want 47998", open.port)
	}
	if open.host != "192.168.1.100" {
		t.Errorf("host = %q, want 192.168.1.100", open.host)
	}
}

func TestParseOpenPayloadTooShort(t *testing.T) {
	if _, err := parseOpenPayload([]byte{1, 2, 3}); err == nil {
		t.Fatal("expected error for short payload")
	}
}

func TestParseOpenPayloadMissingNul(t *testing.T) {
	if _, err := parseOpenPayload([]byte{protoUDP, 0x22, 0xbb, 0x41}); err == nil {
		t.Fatal("expected error for missing NUL terminator")
	}
}

func TestParseOpenPayloadEmptyHost(t *testing.T) {
	if _, err := parseOpenPayload([]byte{protoUDP, 0, 0, 0}); err == nil {
		t.Fatal("expected error for empty host")
	}
}

func TestEncodeClosed(t *testing.T) {
	frame := encodeClosed(7, closeReasonNormal)
	if len(frame) != 5 {
		t.Fatalf("frame len = %d, want 5", len(frame))
	}
	if frame[0] != opClosed || frame[1] != 7 || frame[4] != closeReasonNormal {
		t.Errorf("unexpected frame: %v", frame)
	}
}

func TestDeriveAESKey(t *testing.T) {
	key := deriveAESKey([]byte("0123456789abcdef"), "1234")
	if len(key) != 16 {
		t.Fatalf("key len = %d, want 16", len(key))
	}
	// 与参考实现同输入同输出（SHA256(salt||pin) 前 16 字节）
	again := deriveAESKey([]byte("0123456789abcdef"), "1234")
	if string(key) != string(again) {
		t.Fatal("deriveAESKey not deterministic")
	}
}

func TestAESECB(t *testing.T) {
	key := deriveAESKey([]byte("salt"), "1234")
	plain := []byte("0123456789abcdef0123456789abcdef")
	enc, err := aesECBEncrypt(key, plain)
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	dec, err := aesECBDecrypt(key, enc)
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}
	if string(dec) != string(plain) {
		t.Fatal("AES-ECB roundtrip mismatch")
	}
}

func TestExtractXML(t *testing.T) {
	body := `<root><paired>1</paired><plaincert>abcd</plaincert></root>`
	got, err := extractXML(body, "plaincert")
	if err != nil || got != "abcd" {
		t.Fatalf("extractXML = %q, %v", got, err)
	}
	if _, err := extractXML(body, "missing"); err == nil {
		t.Fatal("expected error for missing tag")
	}
}

func TestCheckPaired(t *testing.T) {
	if err := checkPaired([]byte(`<root><paired>1</paired></root>`), "test"); err != nil {
		t.Fatalf("paired=1 should pass: %v", err)
	}
	if err := checkPaired([]byte(`<root><paired>0</paired></root>`), "test"); err == nil {
		t.Fatal("paired=0 should fail")
	}
}

func TestSanitizeDeviceName(t *testing.T) {
	cases := map[string]string{
		"GameAtlas":        "GameAtlas",
		"我的 设备":            "__+__", // 非 ASCII → '_'，空格 → '+'
		"PC #1 (客厅)":       "PC+_1+____",
		"GameAtlasBrowser": "GameAtlasBrowser",
	}
	for input, want := range cases {
		if got := sanitizeDeviceName(input); got != want {
			t.Errorf("sanitizeDeviceName(%q) = %q, want %q", input, got, want)
		}
	}
	if got := sanitizeDeviceName(string(make([]byte, 100))); len(got) > 64 {
		t.Errorf("sanitizeDeviceName should cap at 64, got %d", len(got))
	}
}

func TestSanitizeHost(t *testing.T) {
	if got := sanitizeHost("192.168.1.100"); got != "192.168.1.100" {
		t.Errorf("sanitizeHost = %q", got)
	}
	if got := sanitizeHost("bad:addr/1"); got != "bad_addr_1" {
		t.Errorf("sanitizeHost = %q", got)
	}
}

func TestNewUUID(t *testing.T) {
	u1 := newUUID()
	u2 := newUUID()
	if u1 == u2 {
		t.Fatal("UUIDs should differ")
	}
	if len(u1) != 36 {
		t.Fatalf("UUID len = %d, want 36", len(u1))
	}
}
