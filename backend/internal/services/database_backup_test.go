package services

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/hao/game/internal/config"
	dbpkg "github.com/hao/game/internal/db"
)

func TestDatabaseBackupServiceBackupNowCreatesReadableSQLiteCopy(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "app.db")
	backupDir := filepath.Join(tempDir, "backups")

	db, err := dbpkg.OpenSQLite(dbPath)
	if err != nil {
		t.Fatalf("OpenSQLite returned error: %v", err)
	}
	defer func() { _ = db.Close() }()

	if _, err := db.Exec(`CREATE TABLE sample (id INTEGER PRIMARY KEY, name TEXT NOT NULL)`); err != nil {
		t.Fatalf("create sample table: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO sample (name) VALUES (?)`, "Need for Speed"); err != nil {
		t.Fatalf("insert sample row: %v", err)
	}

	service := NewDatabaseBackupService(config.Config{
		DBPath:          dbPath,
		DBBackupEnabled: true,
		DBBackupDir:     backupDir,
	}, db)

	backupPath, err := service.BackupNow("startup")
	if err != nil {
		t.Fatalf("BackupNow returned error: %v", err)
	}
	assertFileExists(t, backupPath)

	backupDB, err := dbpkg.OpenSQLite(backupPath)
	if err != nil {
		t.Fatalf("open backup sqlite: %v", err)
	}
	defer func() { _ = backupDB.Close() }()

	var name string
	if err := backupDB.Get(&name, `SELECT name FROM sample WHERE id = 1`); err != nil {
		t.Fatalf("query backup sample row: %v", err)
	}
	if name != "Need for Speed" {
		t.Fatalf("backup name = %q, want Need for Speed", name)
	}
}

func TestDatabaseBackupServiceCleanupOldBackupsKeepsConfiguredNewestFiles(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "app.db")
	backupDir := filepath.Join(tempDir, "backups")
	if err := os.MkdirAll(backupDir, 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}

	oldManaged := filepath.Join(backupDir, "app-startup-20260101-000000-000000000.db")
	middleManaged := filepath.Join(backupDir, "app-startup-20260701-000000-000000000.db")
	newManaged := filepath.Join(backupDir, "app-startup-20260801-000000-000000000.db")
	unmanaged := filepath.Join(backupDir, "manual-copy.db")
	for _, path := range []string{oldManaged, middleManaged, newManaged, unmanaged} {
		if err := os.WriteFile(path, []byte("backup"), 0o644); err != nil {
			t.Fatalf("WriteFile(%s) returned error: %v", path, err)
		}
	}

	now := time.Date(2026, 8, 3, 12, 0, 0, 0, time.UTC)
	oldTime := now.Add(-48 * time.Hour)
	middleTime := now.Add(-24 * time.Hour)
	newTime := now.Add(-2 * time.Hour)
	if err := os.Chtimes(oldManaged, oldTime, oldTime); err != nil {
		t.Fatalf("Chtimes oldManaged returned error: %v", err)
	}
	if err := os.Chtimes(middleManaged, middleTime, middleTime); err != nil {
		t.Fatalf("Chtimes middleManaged returned error: %v", err)
	}
	if err := os.Chtimes(newManaged, newTime, newTime); err != nil {
		t.Fatalf("Chtimes newManaged returned error: %v", err)
	}
	if err := os.Chtimes(unmanaged, oldTime, oldTime); err != nil {
		t.Fatalf("Chtimes unmanaged returned error: %v", err)
	}

	service := NewDatabaseBackupService(config.Config{
		DBPath:                 dbPath,
		DBBackupEnabled:        true,
		DBBackupDir:            backupDir,
		DBBackupRetentionCount: 2,
	}, nil)

	removed, err := service.CleanupOldBackups()
	if err != nil {
		t.Fatalf("CleanupOldBackups returned error: %v", err)
	}
	if removed != 1 {
		t.Fatalf("removed = %d, want 1", removed)
	}
	assertFileMissing(t, oldManaged)
	assertFileExists(t, middleManaged)
	assertFileExists(t, newManaged)
	assertFileExists(t, unmanaged)
}
