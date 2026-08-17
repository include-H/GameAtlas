package streaming

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ---- hostStore 单元测试 ----

func TestHostStoreEmptyOnFirstLoad(t *testing.T) {
	s := newHostStore(t.TempDir())
	hosts, err := s.load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(hosts) != 0 {
		t.Fatalf("expected empty list, got %d hosts", len(hosts))
	}
}

func TestHostStoreUpsertAndPersist(t *testing.T) {
	dir := t.TempDir()
	s := newHostStore(dir)

	hosts, err := s.upsert(hostRecord{Name: "PC", Address: "192.168.1.100", LastSeen: 123})
	if err != nil {
		t.Fatalf("upsert: %v", err)
	}
	if len(hosts) != 1 {
		t.Fatalf("expected 1 host, got %d", len(hosts))
	}
	if hosts[0].ID == "" {
		t.Fatal("expected generated id for empty id")
	}
	if !strings.HasPrefix(hosts[0].ID, "host-") {
		t.Fatalf("unexpected id format: %q", hosts[0].ID)
	}

	// 重新打开存储（模拟重启）应读到同一数据。
	s2 := newHostStore(dir)
	hosts, err = s2.load()
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if len(hosts) != 1 || hosts[0].Address != "192.168.1.100" {
		t.Fatalf("reload mismatch: %+v", hosts)
	}

	// 同 id 更新（字段变更），不新增。
	id := hosts[0].ID
	hosts, err = s2.upsert(hostRecord{ID: id, Name: "LivingRoom PC", Address: "192.168.1.100", LastSeen: 456})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if len(hosts) != 1 || hosts[0].Name != "LivingRoom PC" || hosts[0].LastSeen != 456 {
		t.Fatalf("update mismatch: %+v", hosts)
	}

	// 新增第二条，保持原有条目。
	hosts, err = s2.upsert(hostRecord{Name: "Laptop", Address: "10.0.0.7"})
	if err != nil {
		t.Fatalf("second upsert: %v", err)
	}
	if len(hosts) != 2 {
		t.Fatalf("expected 2 hosts, got %d", len(hosts))
	}
}

func TestHostStoreRemove(t *testing.T) {
	s := newHostStore(t.TempDir())
	hosts, _ := s.upsert(hostRecord{Name: "A", Address: "10.0.0.1"})
	hosts, _ = s.upsert(hostRecord{Name: "B", Address: "10.0.0.2"})
	if len(hosts) != 2 {
		t.Fatalf("setup: expected 2 hosts, got %d", len(hosts))
	}

	hosts, err := s.remove(hosts[0].ID)
	if err != nil {
		t.Fatalf("remove: %v", err)
	}
	if len(hosts) != 1 || hosts[0].Name != "B" {
		t.Fatalf("remove mismatch: %+v", hosts)
	}

	// 删除不存在的 id 不报错、不改变列表。
	if _, err := s.remove("nope"); err != nil {
		t.Fatalf("remove missing id: %v", err)
	}
}

func TestHostStorePaired(t *testing.T) {
	dir := t.TempDir()
	s := newHostStore(dir)

	if s.paired("192.168.1.100") {
		t.Fatal("expected unpaired without cached cert")
	}

	// 模拟 runPairing 成功后的证书落盘位置。
	hostsDir := filepath.Join(dir, "hosts")
	if err := os.MkdirAll(hostsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	certPath := filepath.Join(hostsDir, sanitizeHost("192.168.1.100")+".cert.pem")
	if err := os.WriteFile(certPath, []byte("cert"), 0o600); err != nil {
		t.Fatal(err)
	}

	if !s.paired("192.168.1.100") {
		t.Fatal("expected paired after cert exists")
	}
	if s.paired("10.0.0.1") {
		t.Fatal("expected other host unpaired")
	}

	// views 视图应填充动态 paired。
	hosts, _ := s.upsert(hostRecord{Name: "PC", Address: "192.168.1.100"})
	views := s.views(hosts)
	if !views[0].Paired {
		t.Fatalf("expected paired view, got %+v", views[0])
	}
}

func TestHostStoreAtomicWriteLeavesNoTmp(t *testing.T) {
	dir := t.TempDir()
	s := newHostStore(dir)
	if _, err := s.upsert(hostRecord{Name: "A", Address: "10.0.0.1"}); err != nil {
		t.Fatalf("upsert: %v", err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".tmp") {
			t.Fatalf("leftover tmp file: %s", e.Name())
		}
	}
}

// ---- /api/hosts HTTP 测试 ----

func TestHostsHandler(t *testing.T) {
	dir := t.TempDir()
	srv := &Server{hosts: newHostStore(dir), settings: newSettingsStore(dir)}

	// GET 空列表。
	rec := httptest.NewRecorder()
	srv.handleHosts(rec, httptest.NewRequest(http.MethodGet, "/api/hosts", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET empty: status %d", rec.Code)
	}
	var body struct {
		Hosts []hostView `json:"hosts"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("GET empty: parse: %v", err)
	}
	if len(body.Hosts) != 0 {
		t.Fatalf("GET empty: expected 0, got %d", len(body.Hosts))
	}

	// POST 新增（id 留空，由后端生成）。
	rec = httptest.NewRecorder()
	srv.handleHosts(rec, httptest.NewRequest(http.MethodPost, "/api/hosts",
		strings.NewReader(`{"name":"PC","address":"192.168.1.5","lastSeen":0}`)))
	if rec.Code != http.StatusOK {
		t.Fatalf("POST add: status %d: %s", rec.Code, rec.Body.String())
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("POST add: parse: %v", err)
	}
	if len(body.Hosts) != 1 || body.Hosts[0].ID == "" || body.Hosts[0].Paired {
		t.Fatalf("POST add mismatch: %+v", body.Hosts)
	}
	id := body.Hosts[0].ID

	// POST 更新：name 变更 + 存在证书 → paired=true。
	hostsDir := filepath.Join(dir, "hosts")
	if err := os.MkdirAll(hostsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(hostsDir, sanitizeHost("192.168.1.5")+".cert.pem"), []byte("c"), 0o600); err != nil {
		t.Fatal(err)
	}
	rec = httptest.NewRecorder()
	srv.handleHosts(rec, httptest.NewRequest(http.MethodPost, "/api/hosts",
		strings.NewReader(`{"id":"`+id+`","name":"LivingRoom PC","address":"192.168.1.5","lastSeen":99}`)))
	if rec.Code != http.StatusOK {
		t.Fatalf("POST update: status %d", rec.Code)
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("POST update: parse: %v", err)
	}
	if len(body.Hosts) != 1 || body.Hosts[0].Name != "LivingRoom PC" || !body.Hosts[0].Paired || body.Hosts[0].LastSeen != 99 {
		t.Fatalf("POST update mismatch: %+v", body.Hosts)
	}

	// POST 缺 address → 400。
	rec = httptest.NewRecorder()
	srv.handleHosts(rec, httptest.NewRequest(http.MethodPost, "/api/hosts",
		strings.NewReader(`{"name":"x"}`)))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("POST missing address: status %d", rec.Code)
	}

	// DELETE → 列表清空。
	rec = httptest.NewRecorder()
	srv.handleHosts(rec, httptest.NewRequest(http.MethodDelete, "/api/hosts?id="+id, nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("DELETE: status %d", rec.Code)
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("DELETE: parse: %v", err)
	}
	if len(body.Hosts) != 0 {
		t.Fatalf("DELETE: expected empty, got %+v", body.Hosts)
	}

	// DELETE 缺 id → 400。
	rec = httptest.NewRecorder()
	srv.handleHosts(rec, httptest.NewRequest(http.MethodDelete, "/api/hosts", nil))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("DELETE missing id: status %d", rec.Code)
	}

	// 不支持的方法 → 405。
	rec = httptest.NewRecorder()
	srv.handleHosts(rec, httptest.NewRequest(http.MethodPut, "/api/hosts", nil))
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("PUT: status %d", rec.Code)
	}
}

// ---- /api/stream-settings HTTP 测试 ----

func TestStreamSettingsHandler(t *testing.T) {
	dir := t.TempDir()
	srv := &Server{hosts: newHostStore(dir), settings: newSettingsStore(dir)}

	// GET 首次 → 空对象。
	rec := httptest.NewRecorder()
	srv.handleStreamSettings(rec, httptest.NewRequest(http.MethodGet, "/api/stream-settings", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET: status %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `"settings":{}`) {
		t.Fatalf("GET empty: body %s", rec.Body.String())
	}

	// PUT 保存。
	rec = httptest.NewRecorder()
	srv.handleStreamSettings(rec, httptest.NewRequest(http.MethodPut, "/api/stream-settings",
		strings.NewReader(`{"settings":{"width":1920,"height":1080,"fps":60,"codec":"h264","showStats":false}}`)))
	if rec.Code != http.StatusOK {
		t.Fatalf("PUT: status %d: %s", rec.Code, rec.Body.String())
	}

	// GET 读回一致（模拟重启，新实例读文件）。
	srv2 := &Server{hosts: newHostStore(dir), settings: newSettingsStore(dir)}
	rec = httptest.NewRecorder()
	srv2.handleStreamSettings(rec, httptest.NewRequest(http.MethodGet, "/api/stream-settings", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET after PUT: status %d", rec.Code)
	}
	var body struct {
		Settings map[string]any `json:"settings"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("GET after PUT: parse: %v", err)
	}
	if body.Settings["width"] != float64(1920) || body.Settings["codec"] != "h264" || body.Settings["showStats"] != false {
		t.Fatalf("GET after PUT mismatch: %v", body.Settings)
	}

	// PUT 空 settings → 存空对象，不报错。
	rec = httptest.NewRecorder()
	srv.handleStreamSettings(rec, httptest.NewRequest(http.MethodPut, "/api/stream-settings",
		strings.NewReader(`{"settings":{}}`)))
	if rec.Code != http.StatusOK {
		t.Fatalf("PUT empty: status %d", rec.Code)
	}

	// 非法方法 → 405。
	rec = httptest.NewRecorder()
	srv.handleStreamSettings(rec, httptest.NewRequest(http.MethodPost, "/api/stream-settings", nil))
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("POST: status %d", rec.Code)
	}
}