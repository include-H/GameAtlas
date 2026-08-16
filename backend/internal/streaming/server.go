package streaming

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"log"
	"math/big"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// Options 是串流代理的启动配置。
type Options struct {
	Bind        string // 监听地址
	Port        int    // 监听端口
	DataDir     string // 身份/证书/主机缓存目录
	WWWRoot     string // 串流前端 dist 目录（空 = 不托管静态页）
	MaxChannels byte   // 每会话最大通道数
}

// Server 是浏览器游戏串流代理服务：TLS 监听 + 静态托管 + 配对/启动 API +
// WebSocket 多路复用转发。
type Server struct {
	opts       Options
	identity   *pairIdentity
	server     *http.Server
	ln         net.Listener
	wsUpgrader websocket.Upgrader

	mu      sync.Mutex
	started bool
}

// New 构造串流代理。生成/加载 TLS 证书与配对身份，不监听端口。
func New(opts Options) (*Server, error) {
	if opts.MaxChannels == 0 {
		opts.MaxChannels = 64
	}
	if err := os.MkdirAll(opts.DataDir, 0o755); err != nil {
		return nil, fmt.Errorf("create streaming data dir: %w", err)
	}

	identity, err := loadOrCreateIdentity(opts.DataDir)
	if err != nil {
		return nil, err
	}

	s := &Server{
		opts:     opts,
		identity: identity,
		wsUpgrader: websocket.Upgrader{
			ReadBufferSize:  65536,
			WriteBufferSize: 65536,
			// 同源串流页访问，无需跨源校验。
			CheckOrigin: func(*http.Request) bool { return true },
		},
	}
	return s, nil
}

// Start 生成 TLS 证书并开始监听。失败返回错误（由调用方决定降级）。
func (s *Server) Start() error {
	cert, err := s.loadOrGenerateTLS()
	if err != nil {
		return err
	}

	addr := net.JoinHostPort(s.opts.Bind, fmt.Sprintf("%d", s.opts.Port))
	ln, err := tls.Listen("tcp", addr, &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS12,
		NextProtos:   []string{"http/1.1"},
	})
	if err != nil {
		return fmt.Errorf("streaming listen %s: %w", addr, err)
	}
	s.ln = ln

	mux := http.NewServeMux()
	mux.HandleFunc("/health", s.handleHealth)
	mux.HandleFunc("/api/pair", s.handlePair)
	mux.HandleFunc("/api/applist", s.handleAppList)
	mux.HandleFunc("/api/launch", s.handleLaunch)
	mux.HandleFunc("/proxy", s.handleProxy)

	var handler http.Handler = mux
	if s.opts.WWWRoot != "" {
		handler = s.staticHandler(mux)
	}

	s.server = &http.Server{
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
	}

	s.mu.Lock()
	s.started = true
	s.mu.Unlock()

	log.Printf("[stream] serving on https://%s (wwwRoot=%s)", ln.Addr(), s.opts.WWWRoot)
	return nil
}

// Run 阻塞服务直到关闭。
func (s *Server) Run() error {
	if s.server == nil {
		return fmt.Errorf("streaming server not started")
	}
	err := s.server.Serve(s.ln)
	if err == http.ErrServerClosed {
		return nil
	}
	return err
}

// Shutdown 优雅关闭。
func (s *Server) Shutdown(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.started {
		return nil
	}
	if s.server == nil {
		return nil
	}
	return s.server.Shutdown(ctx)
}

// ---------- 路由 ----------

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "version": "1.0.0"})
}

func (s *Server) handlePair(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	var req pairRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, pairResponse{Error: "invalid json body"})
		return
	}
	if strings.TrimSpace(req.Address) == "" || strings.TrimSpace(req.Pin) == "" {
		writeJSON(w, http.StatusBadRequest, pairResponse{Error: "address and pin are required"})
		return
	}
	deviceName := sanitizeDeviceName(req.DeviceName)
	if deviceName == "" {
		deviceName = deviceNameDefault
	}

	ctx, cancel := context.WithTimeout(r.Context(), pairTimeout)
	defer cancel()

	type result struct {
		cert string
		err  error
	}
	done := make(chan result, 1)
	go func() {
		cert, err := runPairing(s.opts.DataDir, s.identity, strings.TrimSpace(req.Address), strings.TrimSpace(req.Pin), deviceName)
		done <- result{cert: cert, err: err}
	}()

	select {
	case res := <-done:
		if res.err != nil {
			log.Printf("[stream] pair %s failed: %v", req.Address, res.err)
			writeJSON(w, http.StatusOK, pairResponse{Paired: false, Error: res.err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, pairResponse{Paired: true, ServerCert: res.cert})
	case <-ctx.Done():
		log.Printf("[stream] pair %s timed out", req.Address)
		writeJSON(w, http.StatusOK, pairResponse{Paired: false, Error: "timed out"})
	}
}

func (s *Server) handleAppList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	host := strings.TrimSpace(r.URL.Query().Get("host"))
	if host == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "host param is required"})
		return
	}
	apps, err := fetchAppList(s.opts.DataDir, s.identity, host)
	if err != nil {
		log.Printf("[stream] applist %s failed: %v", host, err)
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, appListResponse{Apps: apps})
}

func (s *Server) handleLaunch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	var req launchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json body"})
		return
	}
	if strings.TrimSpace(req.Host) == "" || req.RIKeyHex == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "host and riKeyHex are required"})
		return
	}
	resp, err := doLaunch(s.opts.DataDir, s.identity, &req)
	if err != nil {
		log.Printf("[stream] launch %s app=%d failed: %v", req.Host, req.AppID, err)
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleProxy(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	ws, err := s.wsUpgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("[stream] ws upgrade failed: %v", err)
		return
	}
	log.Printf("[stream] ws session started from %s", r.RemoteAddr)
	runSession(r.Context(), ws, s.opts.MaxChannels)
}

// ---------- 静态托管（COOP/COEP/CORP + SPA fallback） ----------

func (s *Server) staticHandler(api http.Handler) http.Handler {
	root := s.opts.WWWRoot
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cross-Origin-Opener-Policy", "same-origin")
		w.Header().Set("Cross-Origin-Embedder-Policy", "require-corp")
		w.Header().Set("Cross-Origin-Resource-Policy", "same-origin")

		if apiPath(r.URL.Path) {
			api.ServeHTTP(w, r)
			return
		}

		path := strings.TrimPrefix(r.URL.Path, "/")
		if path == "" {
			path = "index.html"
		}
		fullPath := filepath.Join(root, filepath.FromSlash(path))

		// SPA fallback：不存在则回 index.html（history 路由）。
		info, err := os.Stat(fullPath)
		if err != nil || info.IsDir() {
			fullPath = filepath.Join(root, "index.html")
		}

		data, err := os.ReadFile(fullPath)
		if err != nil {
			http.Error(w, "streaming bundle missing (build frontend streaming entry first)", http.StatusNotFound)
			return
		}

		contentType := mimeTypeByPath(fullPath)
		if servedPathIsIndex(fullPath, root) {
			w.Header().Set("Cache-Control", "no-store")
		} else {
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		}
		w.Header().Set("Content-Type", contentType)
		_, _ = w.Write(data)
	})
}

func apiPath(path string) bool {
	return path == "/proxy" ||
		strings.HasPrefix(path, "/api/") ||
		strings.HasPrefix(path, "/health")
}

func servedPathIsIndex(fullPath, root string) bool {
	return filepath.Clean(fullPath) == filepath.Join(root, "index.html")
}

func mimeTypeByPath(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".html":
		return "text/html; charset=utf-8"
	case ".js":
		return "application/javascript; charset=utf-8"
	case ".mjs":
		return "application/javascript; charset=utf-8"
	case ".css":
		return "text/css; charset=utf-8"
	case ".json":
		return "application/json"
	case ".wasm":
		return "application/wasm"
	case ".svg":
		return "image/svg+xml"
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".webp":
		return "image/webp"
	case ".ico":
		return "image/x-icon"
	case ".woff", ".woff2":
		return "font/woff2"
	case ".webmanifest":
		return "application/manifest+json"
	default:
		return "application/octet-stream"
	}
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

// ---------- TLS 自签证书 ----------

const certLifeYears = 10

func (s *Server) loadOrGenerateTLS() (tls.Certificate, error) {
	certPath := filepath.Join(s.opts.DataDir, "server.crt")
	keyPath := filepath.Join(s.opts.DataDir, "server.key")

	if cert, err := tls.LoadX509KeyPair(certPath, keyPath); err == nil {
		return cert, nil
	}

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return tls.Certificate{}, err
	}

	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 159))
	if err != nil {
		return tls.Certificate{}, err
	}

	template := x509.Certificate{
		SerialNumber: serial,
		Subject:      pkix.Name{CommonName: "GameAtlas Streaming"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(certLifeYears * 365 * 24 * time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		DNSNames:     []string{"localhost"},
		IPAddresses:  []net.IP{net.ParseIP("127.0.0.1"), net.ParseIP("::1")},
	}
	der, err := x509.CreateCertificate(rand.Reader, &template, &template, &key.PublicKey, key)
	if err != nil {
		return tls.Certificate{}, err
	}

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	keyDER, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		return tls.Certificate{}, err
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})

	if err := os.WriteFile(certPath, certPEM, 0o644); err != nil {
		return tls.Certificate{}, err
	}
	if err := os.WriteFile(keyPath, keyPEM, 0o600); err != nil {
		return tls.Certificate{}, err
	}

	log.Printf("[stream] self-signed TLS certificate generated in %s", s.opts.DataDir)
	return tls.X509KeyPair(certPEM, keyPEM)
}
