package streaming

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sync"
)

// settingsStore 是串流设置的 JSON 文件存储（<dataDir>/stream-settings.json）。
// 内容不校验结构：前端全量保存、原样读回。
type settingsStore struct {
	mu   sync.Mutex
	path string
}

func newSettingsStore(dataDir string) *settingsStore {
	return &settingsStore{path: filepath.Join(dataDir, "stream-settings.json")}
}

// load 读回设置；文件不存在返回空对象（首次启动）。
func (s *settingsStore) load() (map[string]any, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	raw, err := os.ReadFile(s.path)
	if os.IsNotExist(err) {
		return map[string]any{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read stream-settings file: %w", err)
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil, fmt.Errorf("parse stream-settings file: %w", err)
	}
	if m == nil {
		m = map[string]any{}
	}
	return m, nil
}

// save 原子写设置：临时文件 + rename（调用方持锁）。
func (s *settingsStore) save(m map[string]any) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	raw, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o644); err != nil {
		return fmt.Errorf("write stream-settings tmp file: %w", err)
	}
	if err := os.Rename(tmp, s.path); err != nil {
		return fmt.Errorf("rename stream-settings file: %w", err)
	}
	return nil
}

// handleStreamSettings 串流设置 API：
//
//	GET /api/stream-settings             → {"settings":{...}}（文件缺失时空对象）
//	PUT /api/stream-settings body:{"settings":{...}} → 原样存，返回存储结果
func (s *Server) handleStreamSettings(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		m, err := s.settings.load()
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"settings": m})

	case http.MethodPut:
		var req struct {
			Settings map[string]any `json:"settings"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json body"})
			return
		}
		if req.Settings == nil {
			req.Settings = map[string]any{}
		}
		if err := s.settings.save(req.Settings); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"settings": req.Settings})

	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}
