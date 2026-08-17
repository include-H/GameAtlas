package streaming

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// hostRecord 是 hosts.json 中持久化的主机条目。配对状态不入库——
// paired 由 dataDir/hosts/<sanitizeHost(address)>.cert.pem 是否存在
// 动态判断（runPairing 成功时后端已缓存该证书）。
type hostRecord struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Address  string `json:"address"`
	LastSeen int64  `json:"lastSeen"`
}

// hostView 是 /api/hosts 返回的主机视图：持久化字段 + 动态 paired。
type hostView struct {
	hostRecord
	Paired bool `json:"paired"`
}

// hostsFile 是 hosts.json 的磁盘结构。
type hostsFile struct {
	Hosts []hostRecord `json:"hosts"`
}

// hostStore 是主机列表的 JSON 文件存储（<dataDir>/hosts.json）。
// 注意与 <dataDir>/hosts/ 证书目录分离，后者只存配对缓存证书。
type hostStore struct {
	mu       sync.Mutex
	path     string
	hostsDir string // <dataDir>/hosts/，仅用于 paired 判断
}

func newHostStore(dataDir string) *hostStore {
	return &hostStore{
		path:     filepath.Join(dataDir, "hosts.json"),
		hostsDir: filepath.Join(dataDir, "hosts"),
	}
}

// load 读回全部主机；文件不存在返回空列表（首次启动）。
func (s *hostStore) load() ([]hostRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.loadLocked()
}

func (s *hostStore) loadLocked() ([]hostRecord, error) {
	raw, err := os.ReadFile(s.path)
	if os.IsNotExist(err) {
		return []hostRecord{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read hosts file: %w", err)
	}
	var f hostsFile
	if err := json.Unmarshal(raw, &f); err != nil {
		return nil, fmt.Errorf("parse hosts file: %w", err)
	}
	if f.Hosts == nil {
		f.Hosts = []hostRecord{}
	}
	return f.Hosts, nil
}

// upsert 按 id 更新或新增，返回全量列表；id 为空时生成随机 id。
func (s *hostStore) upsert(h hostRecord) ([]hostRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	hosts, err := s.loadLocked()
	if err != nil {
		return nil, err
	}
	if h.ID == "" {
		h.ID = newHostID()
	}
	idx := -1
	for i, cur := range hosts {
		if cur.ID == h.ID {
			idx = i
			break
		}
	}
	if idx >= 0 {
		hosts[idx] = h
	} else {
		hosts = append(hosts, h)
	}
	if err := s.saveLocked(hosts); err != nil {
		return nil, err
	}
	return hosts, nil
}

// remove 按 id 删除，返回全量列表（id 不存在时静默成功）。
func (s *hostStore) remove(id string) ([]hostRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	hosts, err := s.loadLocked()
	if err != nil {
		return nil, err
	}
	filtered := make([]hostRecord, 0, len(hosts))
	for _, h := range hosts {
		if h.ID != id {
			filtered = append(filtered, h)
		}
	}
	if err := s.saveLocked(filtered); err != nil {
		return nil, err
	}
	return filtered, nil
}

// saveLocked 原子写：临时文件 + rename（调用方持锁）。
func (s *hostStore) saveLocked(hosts []hostRecord) error {
	raw, err := json.MarshalIndent(hostsFile{Hosts: hosts}, "", "  ")
	if err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o644); err != nil {
		return fmt.Errorf("write hosts tmp file: %w", err)
	}
	if err := os.Rename(tmp, s.path); err != nil {
		return fmt.Errorf("rename hosts file: %w", err)
	}
	return nil
}

// paired 判断：<dataDir>/hosts/<sanitizeHost(address)>.cert.pem 存在即已配对。
func (s *hostStore) paired(address string) bool {
	_, err := os.Stat(filepath.Join(s.hostsDir, sanitizeHost(address)+".cert.pem"))
	return err == nil
}

// views 把持久化条目转成 API 视图（填充动态 paired）。
func (s *hostStore) views(hosts []hostRecord) []hostView {
	out := make([]hostView, 0, len(hosts))
	for _, h := range hosts {
		out = append(out, hostView{hostRecord: h, Paired: s.paired(h.Address)})
	}
	return out
}

// newHostID 生成 "host-" + 16 位随机 hex（crypto/rand）。
func newHostID() string {
	raw := make([]byte, 8)
	if _, err := rand.Read(raw); err != nil {
		// crypto/rand 失败极罕见；退化为时间戳保证 id 非空。
		return fmt.Sprintf("host-%d", time.Now().UnixNano())
	}
	return "host-" + hex.EncodeToString(raw)
}

// handleHosts 主机列表 API：
//
//	GET    /api/hosts            → {"hosts":[{id,name,address,lastSeen,paired}]}
//	POST   /api/hosts            body: {id?,name,address,lastSeen?} → 全量列表
//	DELETE /api/hosts?id=<id>    → 全量列表
func (s *Server) handleHosts(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		hosts, err := s.hosts.load()
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"hosts": s.hosts.views(hosts)})

	case http.MethodPost:
		var h hostRecord
		if err := json.NewDecoder(r.Body).Decode(&h); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json body"})
			return
		}
		h.Address = strings.TrimSpace(h.Address)
		if h.Address == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "address is required"})
			return
		}
		if strings.TrimSpace(h.Name) == "" {
			h.Name = h.Address
		}
		hosts, err := s.hosts.upsert(h)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"hosts": s.hosts.views(hosts)})

	case http.MethodDelete:
		id := strings.TrimSpace(r.URL.Query().Get("id"))
		if id == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "id param is required"})
			return
		}
		hosts, err := s.hosts.remove(id)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"hosts": s.hosts.views(hosts)})

	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}
