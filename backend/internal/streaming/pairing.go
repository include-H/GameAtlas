package streaming

import (
	"crypto"
	"crypto/aes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"io"
	"log"
	"math/big"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	deviceNameDefault = "GameAtlas"
	nvhttpPort        = 47989
	pairTimeout       = 180 * time.Second
	hashBytes         = 32
)

// pairIdentity 是持久化的客户端身份：X.509 自签证书 + RSA 私钥 + 唯一 ID。
// 多浏览器共享同一身份（Sunshine 的 Paired Clients 里只显示一个设备）。
type pairIdentity struct {
	uniqueID  string
	certPEM   string
	keyPEM    string
	certBytes []byte // DER，用于取签名
}

// identityDir 返回身份与主机证书缓存目录。
func identityDir(dataDir string) string {
	return filepath.Join(dataDir, "pairing")
}

// loadOrCreateIdentity 加载或生成客户端身份（幂等）。
func loadOrCreateIdentity(dataDir string) (*pairIdentity, error) {
	dir := identityDir(dataDir)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("create pairing dir: %w", err)
	}

	certPath := filepath.Join(dir, "client_cert.pem")
	keyPath := filepath.Join(dir, "client_key.pem")
	idPath := filepath.Join(dir, "unique_id.txt")

	_, certErr := os.Stat(certPath)
	_, keyErr := os.Stat(keyPath)
	if os.IsNotExist(certErr) || os.IsNotExist(keyErr) {
		if err := generateClientIdentity(certPath, keyPath); err != nil {
			return nil, err
		}
	}

	uniqueID := ""
	if raw, err := os.ReadFile(idPath); err == nil {
		uniqueID = strings.TrimSpace(string(raw))
	}
	if uniqueID == "" {
		raw := make([]byte, 8)
		if _, err := io.ReadFull(rand.Reader, raw); err != nil {
			return nil, err
		}
		uniqueID = hex.EncodeToString(raw)
		if err := os.WriteFile(idPath, []byte(uniqueID), 0o644); err != nil {
			return nil, err
		}
	}

	certPEM, err := os.ReadFile(certPath)
	if err != nil {
		return nil, err
	}
	keyPEM, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, err
	}

	block, _ := pem.Decode(certPEM)
	if block == nil {
		return nil, fmt.Errorf("decode client cert: empty PEM block")
	}

	return &pairIdentity{
		uniqueID:  uniqueID,
		certPEM:   string(certPEM),
		keyPEM:    string(keyPEM),
		certBytes: block.Bytes,
	}, nil
}

// generateClientIdentity 生成 RSA-2048 自签证书（CN=NVIDIA GameStream
// Client，SHA-256，20 年），与 moonlight 全系客户端同一身份格式。
func generateClientIdentity(certPath, keyPath string) error {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return err
	}

	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 159))
	if err != nil {
		return err
	}

	template := x509.Certificate{
		SerialNumber: serial,
		Subject:      pkix.Name{CommonName: "NVIDIA GameStream Client"},
		Issuer:       pkix.Name{CommonName: "NVIDIA GameStream Client"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(20 * 365 * 24 * time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
	}
	der, err := x509.CreateCertificate(rand.Reader, &template, &template, &key.PublicKey, key)
	if err != nil {
		return err
	}

	certOut, err := os.Create(certPath)
	if err != nil {
		return err
	}
	defer func() { _ = certOut.Close() }()
	if err := pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: der}); err != nil {
		return err
	}

	keyOut, err := os.Create(keyPath)
	if err != nil {
		return err
	}
	defer func() { _ = keyOut.Close() }()
	if err := pem.Encode(keyOut, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)}); err != nil {
		return err
	}
	return nil
}

// pairRequest 是 POST /api/pair 的请求体。
type pairRequest struct {
	Address    string `json:"address"`
	Pin        string `json:"pin"`
	DeviceName string `json:"deviceName,omitempty"`
}

// pairResponse 是 /api/pair 的响应体。
type pairResponse struct {
	Paired     bool   `json:"paired"`
	ServerCert string `json:"serverCert,omitempty"`
	Error      string `json:"error,omitempty"`
}

// runPairing 执行 5 步配对握手。成功后把主机证书缓存到
// dataDir/hosts/<address>.cert.pem，供后续 NvHTTPS（mTLS）调用。
func runPairing(dataDir string, id *pairIdentity, address, pin, deviceName string) (string, error) {
	serverCertPEM, err := doPair(id, address, pin, deviceName)

	if err != nil {
		// 参考行为：任何失败先 /unpair 清掉主机上的半配对状态，下次干净重来。
		unpairURL := fmt.Sprintf("http://%s:%d/unpair?uniqueid=%s", address, nvhttpPort, url.QueryEscape(id.uniqueID))
		client := &http.Client{Timeout: pairTimeout}
		if resp, uErr := client.Get(unpairURL); uErr == nil {
			_ = resp.Body.Close()
		}
		return "", err
	}

	hostDir := filepath.Join(dataDir, "hosts")
	if err := os.MkdirAll(hostDir, 0o755); err == nil {
		_ = os.WriteFile(filepath.Join(hostDir, sanitizeHost(address)+".cert.pem"), []byte(serverCertPEM), 0o600)
	}
	return serverCertPEM, nil
}

// cachedServerCert 读取配对时缓存的主机证书。
func cachedServerCert(dataDir, address string) (string, error) {
	raw, err := os.ReadFile(filepath.Join(dataDir, "hosts", sanitizeHost(address)+".cert.pem"))
	if err != nil {
		return "", fmt.Errorf("no cached server cert for %s: %w", address, err)
	}
	return string(raw), nil
}

// doPair 逐字节对照 moonlight-chrome/libgamestream/pairing.c:gs_pair()。
// 四步全部是 47989 端口的明文 HTTP，无 uuid 参数（参考实现用 uniqueid 作会话键）。
func doPair(id *pairIdentity, address, pin, deviceName string) (string, error) {
	salt := randBytes(16)
	aesKey := deriveAESKey(salt, pin)

	// Sunshine 的 HTTP server 对 keep-alive 跨步骤连接会出错：每步强制新 TCP 连接。
	client := &http.Client{
		Timeout:   pairTimeout,
		Transport: &http.Transport{DisableKeepAlives: true},
	}

	base := fmt.Sprintf("http://%s:%d/pair", address, nvhttpPort)
	common := func() string {
		return fmt.Sprintf("uniqueid=%s&devicename=%s&updateState=1",
			url.QueryEscape(id.uniqueID), url.QueryEscape(deviceName))
	}

	// ---- Step 1: getservercert ----
	step1URL := fmt.Sprintf("%s?%s&phrase=getservercert&salt=%s&clientcert=%s",
		base, common(), hex.EncodeToString(salt), hex.EncodeToString([]byte(id.certPEM)))
	resp1, err := client.Get(step1URL)
	if err != nil {
		return "", fmt.Errorf("step 1 (getservercert): %w", err)
	}
	body1, err := io.ReadAll(resp1.Body)
	_ = resp1.Body.Close()
	if err != nil {
		return "", fmt.Errorf("step 1 (getservercert): read body: %w", err)
	}
	log.Printf("[stream] pair step1 body_len=%d", len(body1))
	if err := checkPaired(body1, "step 1 (getservercert)"); err != nil {
		return "", err
	}
	plainCertHex, err := extractXML(string(body1), "plaincert")
	if err != nil {
		return "", err
	}
	serverCertPEM, err := hex.DecodeString(plainCertHex)
	if err != nil {
		return "", fmt.Errorf("step 1: decode plaincert: %w", err)
	}
	serverCert, err := x509.ParseCertificate(mustPEMBlock(serverCertPEM))
	if err != nil {
		return "", fmt.Errorf("step 1: parse server cert: %w", err)
	}
	serverCertSig := serverCert.Signature

	// ---- Step 2: clientchallenge ----
	randomChallenge := randBytes(16)
	encChallenge, err := aesECBEncrypt(aesKey, randomChallenge)
	if err != nil {
		return "", err
	}
	step2URL := fmt.Sprintf("%s?%s&clientchallenge=%s", base, common(), hex.EncodeToString(encChallenge))
	resp2, err := client.Get(step2URL)
	if err != nil {
		return "", fmt.Errorf("step 2 (clientchallenge): %w", err)
	}
	body2, err := io.ReadAll(resp2.Body)
	_ = resp2.Body.Close()
	if err != nil {
		return "", fmt.Errorf("step 2 (clientchallenge): read body: %w", err)
	}
	log.Printf("[stream] pair step2 body_len=%d", len(body2))
	if err := checkPaired(body2, "step 2 (clientchallenge)"); err != nil {
		return "", err
	}
	encRespHex, err := extractXML(string(body2), "challengeresponse")
	if err != nil {
		return "", err
	}
	encResp, err := hex.DecodeString(encRespHex)
	if err != nil {
		return "", fmt.Errorf("step 2: decode challengeresponse: %w", err)
	}
	decrypted, err := aesECBDecrypt(aesKey, encResp)
	if err != nil {
		return "", err
	}
	if len(decrypted) < hashBytes+16 {
		return "", fmt.Errorf("step 2: server response too short (%d)", len(decrypted))
	}
	serverResponse := decrypted[:hashBytes]
	serverChallenge := decrypted[hashBytes : hashBytes+16]

	// ---- Step 3: serverchallengeresp ----
	clientSecret := randBytes(16)
	clientResponseHash := sha256.New()
	clientResponseHash.Write(serverChallenge)
	clientResponseHash.Write(serverCertSig)
	clientResponseHash.Write(clientSecret)
	clientResponse := clientResponseHash.Sum(nil)

	encClientResp, err := aesECBEncrypt(aesKey, clientResponse)
	if err != nil {
		return "", err
	}
	step3URL := fmt.Sprintf("%s?%s&serverchallengeresp=%s", base, common(), hex.EncodeToString(encClientResp))
	resp3, err := client.Get(step3URL)
	if err != nil {
		return "", fmt.Errorf("step 3 (serverchallengeresp): %w", err)
	}
	body3, err := io.ReadAll(resp3.Body)
	_ = resp3.Body.Close()
	if err != nil {
		return "", fmt.Errorf("step 3 (serverchallengeresp): read body: %w", err)
	}
	log.Printf("[stream] pair step3 body_len=%d", len(body3))
	if err := checkPaired(body3, "step 3 (serverchallengeresp)"); err != nil {
		return "", err
	}
	pairingSecretHex, err := extractXML(string(body3), "pairingsecret")
	if err != nil {
		return "", err
	}
	pairingSecret, err := hex.DecodeString(pairingSecretHex)
	if err != nil {
		return "", fmt.Errorf("step 3: decode pairingsecret: %w", err)
	}
	if len(pairingSecret) < 16+256 {
		return "", fmt.Errorf("step 3: pairingsecret too short (%d)", len(pairingSecret))
	}
	serverSecret := pairingSecret[:16]
	serverSecretSig := pairingSecret[16:]

	if err := rsa.VerifyPKCS1v15(serverCert.PublicKey.(*rsa.PublicKey), crypto.SHA256, hashOf(serverSecret), serverSecretSig); err != nil {
		return "", fmt.Errorf("step 3: server secret signature invalid - host's identity changed?")
	}

	expectedHash := sha256.New()
	expectedHash.Write(randomChallenge)
	expectedHash.Write(serverCertSig)
	expectedHash.Write(serverSecret)
	expected := expectedHash.Sum(nil)
	if string(expected) != string(serverResponse) {
		return "", fmt.Errorf("step 3: server challenge mismatch - wrong PIN entered on the host?")
	}

	// ---- Step 4: clientpairingsecret ----
	key, err := parseRSAPrivateKey(id.keyPEM)
	if err != nil {
		return "", err
	}
	clientSecretSig, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, hashOf(clientSecret))
	if err != nil {
		return "", err
	}
	clientPairSecret := append(append([]byte{}, clientSecret...), clientSecretSig...)

	step4URL := fmt.Sprintf("%s?%s&clientpairingsecret=%s", base, common(), hex.EncodeToString(clientPairSecret))
	resp4, err := client.Get(step4URL)
	if err != nil {
		return "", fmt.Errorf("step 4 (clientpairingsecret): %w", err)
	}
	body4, err := io.ReadAll(resp4.Body)
	_ = resp4.Body.Close()
	if err != nil {
		return "", fmt.Errorf("step 4 (clientpairingsecret): read body: %w", err)
	}
	log.Printf("[stream] pair step4 body_len=%d", len(body4))
	if err := checkPaired(body4, "step 4 (clientpairingsecret)"); err != nil {
		return "", err
	}

	return string(serverCertPEM), nil
}

func deriveAESKey(salt []byte, pin string) []byte {
	h := sha256.New()
	h.Write(salt)
	h.Write([]byte(pin))
	return h.Sum(nil)[:16]
}

// aesECBEncrypt 实现 AES-128-ECB 加密（Go 标准库无 ECB，手工逐块）。
func aesECBEncrypt(key, data []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	out := make([]byte, len(data))
	for i := 0; i < len(data); i += block.BlockSize() {
		block.Encrypt(out[i:], data[i:])
	}
	return out, nil
}

func aesECBDecrypt(key, data []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	out := make([]byte, len(data))
	for i := 0; i < len(data); i += block.BlockSize() {
		block.Decrypt(out[i:], data[i:])
	}
	return out, nil
}

func randBytes(n int) []byte {
	b := make([]byte, n)
	if _, err := io.ReadFull(rand.Reader, b); err != nil {
		panic(err)
	}
	return b
}

func hashOf(data []byte) []byte {
	h := sha256.Sum256(data)
	return h[:]
}

// extractXML 提取 <tag>...</tag> 内的文本。
func extractXML(body, tag string) (string, error) {
	open := "<" + tag + ">"
	closeTag := "</" + tag + ">"
	s := strings.Index(body, open)
	if s < 0 {
		return "", fmt.Errorf("response missing <%s>: %s", tag, truncate(body, 200))
	}
	s += len(open)
	e := strings.Index(body[s:], closeTag)
	if e < 0 {
		return "", fmt.Errorf("response missing </%s>: %s", tag, truncate(body, 200))
	}
	return strings.TrimSpace(body[s : s+e]), nil
}

func checkPaired(body []byte, step string) error {
	paired, err := extractXML(string(body), "paired")
	if err != nil || paired != "1" {
		return fmt.Errorf("%s: host responded paired=%s", step, paired)
	}
	return nil
}

func sanitizeHost(address string) string {
	var b strings.Builder
	for _, c := range address {
		switch {
		case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z', c >= '0' && c <= '9', c == '.', c == '-':
			b.WriteRune(c)
		default:
			b.WriteByte('_')
		}
	}
	return b.String()
}

// sanitizeDeviceName 收敛用户设备名（对齐上游 Rust 实现）：ASCII 字母数字
// 原样，'-' '_' '.' 保留，空格换 '+'，其余一律 '_'，截断 64 字符。
func sanitizeDeviceName(name string) string {
	var b strings.Builder
	count := 0
	for _, c := range name {
		if count >= 64 {
			break
		}
		switch {
		case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z', c >= '0' && c <= '9', c == '-', c == '_', c == '.':
			b.WriteRune(c)
		case c == ' ':
			b.WriteByte('+')
		default:
			b.WriteByte('_')
		}
		count++
	}
	return b.String()
}

// mustPEMBlock 解析 PEM（兼容 Rust 侧直接 hex 解码的 PEM 字符串）。
func mustPEMBlock(raw []byte) []byte {
	block, _ := pem.Decode(raw)
	if block == nil {
		return raw // 非 PEM（畸形数据），交给 ParseCertificate 报错
	}
	return block.Bytes
}

func parseRSAPrivateKey(pemData string) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode([]byte(pemData))
	if block == nil {
		return nil, fmt.Errorf("decode client key: empty PEM block")
	}
	return x509.ParsePKCS1PrivateKey(block.Bytes)
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
