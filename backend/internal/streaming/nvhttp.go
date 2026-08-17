package streaming

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"encoding/xml"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

const nvHTTPSPort = 47984

var nvHTTPTimeout = 15 * time.Second

// appEntry 是 /api/applist 返回的单个应用。
type appEntry struct {
	ID           uint32 `json:"id"`
	Title        string `json:"title"`
	HdrSupported *bool  `json:"hdrSupported,omitempty"`
}

type appListResponse struct {
	Apps []appEntry `json:"apps"`
}

type launchRequest struct {
	Host        string `json:"host"`
	AppID       uint32 `json:"appId"`
	Width       uint32 `json:"width"`
	Height      uint32 `json:"height"`
	FPS         uint32 `json:"fps"`
	Bitrate     uint32 `json:"bitrate"`
	AudioConfig string `json:"audioConfig"`
	RIKeyHex    string `json:"riKeyHex"`
	RIKeyID     int32  `json:"riKeyId"`
	Resume      bool   `json:"resume"`
}

type launchResponse struct {
	RTSPURL     string `json:"rtspSessionUrl"`
	GameSession string `json:"gameSession"`
	AppVersion  string `json:"appVersion"`
	GfeVersion  string `json:"gfeVersion"`
}

// mtlsClient 构造带客户端身份 + 主机证书固定的 HTTPS 客户端。
// 主机证书是配对时缓存的自签证书，宿主无 SAN，因此关闭主机名校验。
func mtlsClient(dataDir string, id *pairIdentity, address string) (*http.Client, error) {
	serverCertPEM, err := cachedServerCert(dataDir, address)
	if err != nil {
		return nil, err
	}

	clientBlock, _ := pem.Decode([]byte(id.certPEM))
	keyBlock, _ := pem.Decode([]byte(id.keyPEM))
	serverBlock, _ := pem.Decode([]byte(serverCertPEM))
	if clientBlock == nil || keyBlock == nil || serverBlock == nil {
		return nil, fmt.Errorf("decode pairing material for %s", address)
	}

	serverCerts, err := x509.ParseCertificates(serverBlock.Bytes)
	if err != nil || len(serverCerts) == 0 {
		return nil, fmt.Errorf("parse server cert for %s: %w", address, err)
	}

	clientCert, err := tls.X509KeyPair([]byte(id.certPEM), []byte(id.keyPEM))
	if err != nil {
		return nil, fmt.Errorf("load client identity: %w", err)
	}

	pool := x509.NewCertPool()
	pool.AddCert(serverCerts[0])

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{clientCert},
		RootCAs:      pool,
		// 自签证书无 SAN，无法做主机名校验（上游同款行为）。
		InsecureSkipVerify: true, //nolint:gosec // 证书已被配对时 pin 到根池，见上方 pool
		MinVersion:         tls.VersionTLS12,
	}

	return &http.Client{
		Timeout:   nvHTTPTimeout,
		Transport: &http.Transport{TLSClientConfig: tlsConfig, DisableKeepAlives: true},
	}, nil
}

// fetchAppList 拉取主机应用列表。
func fetchAppList(dataDir string, id *pairIdentity, host string) ([]appEntry, error) {
	client, err := mtlsClient(dataDir, id, host)
	if err != nil {
		return nil, err
	}
	url := fmt.Sprintf("https://%s:%d/applist?uniqueid=%s&uuid=%s",
		host, nvHTTPSPort, id.uniqueID, newUUID())
	body, err := clientGetBody(client, url)
	if err != nil {
		return nil, err
	}
	return parseAppList(body)
}

type nvXMLRoot struct {
	StatusCode  int     `xml:"status_code,attr"`
	StatusMsg   string  `xml:"status_message,attr"`
	AppList     []nvApp `xml:"App"` // Sunshine 返回 <root><App>... 平铺，无 AppList 容器
	SessionURL0 string  `xml:"sessionUrl0"`
	GameSession string  `xml:"gamesession"`
}

type nvApp struct {
	ID             uint32 `xml:"ID"`
	AppTitle       string `xml:"AppTitle"`
	IsHdrSupported string `xml:"IsHdrSupported"`
}

func parseAppList(body []byte) ([]appEntry, error) {
	var root nvXMLRoot
	if err := xml.Unmarshal(body, &root); err != nil {
		return nil, fmt.Errorf("parse applist: %w", err)
	}
	out := make([]appEntry, 0, len(root.AppList))
	for _, app := range root.AppList {
		if app.ID == 0 {
			continue
		}
		hdr := app.IsHdrSupported == "1"
		entry := appEntry{ID: app.ID, Title: app.AppTitle}
		if app.IsHdrSupported != "" {
			entry.HdrSupported = &hdr
		}
		out = append(out, entry)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("applist response had no App entries: %s", truncate(string(body), 200))
	}
	return out, nil
}

// doLaunch 请求 /launch，Sunshine 返回 400（应用已在运行）时重试 /resume。
func doLaunch(dataDir string, id *pairIdentity, req *launchRequest) (*launchResponse, error) {
	client, err := mtlsClient(dataDir, id, req.Host)
	if err != nil {
		return nil, err
	}

	appVersion, gfeVersion := fetchServerInfoVersions(client, id, req.Host)

	surround := "196610" // stereo：(3<<16)|2
	switch req.AudioConfig {
	case "surround51":
		surround = "65543" // (5<<16)|7
	case "surround71":
		surround = "65799" // (7<<16)|7
	}

	query := fmt.Sprintf("uniqueid=%s&uuid=%s&appid=%d&mode=%dx%dx%d&additionalStates=1&sops=1"+
		"&rikey=%s&rikeyid=%d&localAudioPlayMode=0&surroundAudioInfo=%s&remoteControllersBitmap=0&gcmap=0&gcpersist=0",
		id.uniqueID, newUUID(), req.AppID, req.Width, req.Height, req.FPS,
		req.RIKeyHex, req.RIKeyID, surround)

	endpoint := "launch"
	if req.Resume {
		endpoint = "resume"
	}
	url := fmt.Sprintf("https://%s:%d/%s?%s", req.Host, nvHTTPSPort, endpoint, query)

	// GameStream 惯例：HTTP 状态恒为 200，业务状态在 XML status_code 属性里。
	// 必须用 XML 值判断，否则 400（应用已在运行）会漏掉 resume 重试。
	body, _, err := clientGet(client, url)
	if err != nil {
		return nil, err
	}
	status := extractStatusCode(body)

	if !req.Resume && status == 400 {
		// Sunshine 在别处拥有应用时会拒 launch；对 Apollo 等 fork 尝试 resume 加入会话。
		log.Printf("[stream] /launch rejected (400), retrying as /resume")
		resumeURL := strings.Replace(url, "/launch?", "/resume?", 1)
		body, _, err = clientGet(client, resumeURL)
		if err != nil {
			return nil, err
		}
		status = extractStatusCode(body)
		if status == 200 {
			log.Printf("[stream] /resume returned 200; vanilla Sunshine may drop the RTSP connection if another client owns the session")
		}
	}

	if status != 200 {
		return nil, fmt.Errorf("host rejected %s: status_code=%d", endpoint, status)
	}

	var root nvXMLRoot
	if err := xml.Unmarshal(body, &root); err != nil {
		return nil, fmt.Errorf("parse launch response: %w", err)
	}
	if root.SessionURL0 == "" {
		return nil, fmt.Errorf("launch response missing sessionUrl0: %s", truncate(string(body), 200))
	}

	return &launchResponse{
		RTSPURL:     root.SessionURL0,
		GameSession: root.GameSession,
		AppVersion:  appVersion,
		GfeVersion:  gfeVersion,
	}, nil
}

// fetchServerInfoVersions 取 appversion / GfeVersion 供 LiStartConnection
// 使用；失败不阻塞 launch（空值继续）。
func fetchServerInfoVersions(client *http.Client, id *pairIdentity, host string) (string, string) {
	url := fmt.Sprintf("https://%s:%d/serverinfo?uniqueid=%s&uuid=%s",
		host, nvHTTPSPort, id.uniqueID, newUUID())
	body, err := clientGetBody(client, url)
	if err != nil {
		log.Printf("[stream] serverinfo failed (continuing): %v", err)
		return "", ""
	}
	appVersion, errA := extractXML(string(body), "appversion")
	gfeVersion, errG := extractXML(string(body), "GfeVersion")
	if errA != nil || errG != nil {
		log.Printf("[stream] serverinfo missing version fields (continuing)")
		return "", ""
	}
	return strings.TrimSpace(appVersion), strings.TrimSpace(gfeVersion)
}

func clientGet(client *http.Client, url string) ([]byte, int, error) {
	resp, err := client.Get(url)
	if err != nil {
		return nil, 0, err
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, 0, err
	}
	return body, resp.StatusCode, nil
}

// extractStatusCode 从 GameStream XML 响应解析 status_code 属性
// （HTTP 状态恒为 200，业务状态在 XML 里）。
func extractStatusCode(body []byte) int {
	var root nvXMLRoot
	if err := xml.Unmarshal(body, &root); err != nil {
		return 200
	}
	return root.StatusCode
}

func clientGetBody(client *http.Client, url string) ([]byte, error) {
	body, _, err := clientGet(client, url)
	return body, err
}

func newUUID() string {
	b := randBytes(16)
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
