package services

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"

	"github.com/hao/game/internal/config"
)

type DatabaseBackupService struct {
	db             *sqlx.DB
	dbPath         string
	backupDir      string
	enabled        bool
	interval       time.Duration
	retentionCount int
	now            func() time.Time
	logf           func(string, ...any)
}

func NewDatabaseBackupService(cfg config.Config, db *sqlx.DB) *DatabaseBackupService {
	return &DatabaseBackupService{
		db:             db,
		dbPath:         cfg.DBPath,
		backupDir:      cfg.DBBackupDir,
		enabled:        cfg.DBBackupEnabled,
		interval:       cfg.DBBackupInterval,
		retentionCount: cfg.DBBackupRetentionCount,
		now:            time.Now,
		logf:           log.Printf,
	}
}

func (s *DatabaseBackupService) BackupNow(reason string) (string, error) {
	if !s.enabled {
		return "", nil
	}
	if strings.TrimSpace(s.backupDir) == "" {
		return "", fmt.Errorf("database backup dir is empty")
	}
	if err := os.MkdirAll(s.backupDir, 0o755); err != nil {
		return "", fmt.Errorf("create database backup directory: %w", err)
	}

	target := s.backupPath(reason)
	if _, err := s.db.Exec(`VACUUM main INTO ?`, target); err != nil {
		return "", fmt.Errorf("backup sqlite database: %w", err)
	}

	return target, nil
}

func (s *DatabaseBackupService) CleanupOldBackups() (int, error) {
	if !s.enabled || s.retentionCount <= 0 {
		return 0, nil
	}

	info, err := os.Stat(s.backupDir)
	if os.IsNotExist(err) {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("stat database backup directory: %w", err)
	}
	if !info.IsDir() {
		return 0, fmt.Errorf("database backup path is not a directory: %s", s.backupDir)
	}

	prefix := backupFilenamePrefix(s.dbPath)
	backups := make([]databaseBackupFile, 0)
	err = filepath.WalkDir(s.backupDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if !strings.HasPrefix(d.Name(), prefix+"-") || filepath.Ext(d.Name()) != ".db" {
			return nil
		}
		info, statErr := d.Info()
		if statErr != nil {
			return statErr
		}
		backups = append(backups, databaseBackupFile{path: path, modifiedAt: info.ModTime()})
		return nil
	})
	if err != nil {
		return 0, fmt.Errorf("list database backups: %w", err)
	}

	sort.Slice(backups, func(i, j int) bool {
		if backups[i].modifiedAt.Equal(backups[j].modifiedAt) {
			return backups[i].path > backups[j].path
		}
		return backups[i].modifiedAt.After(backups[j].modifiedAt)
	})
	if len(backups) <= s.retentionCount {
		return 0, nil
	}

	removed := 0
	for _, backup := range backups[s.retentionCount:] {
		if err := os.Remove(backup.path); err != nil {
			return removed, fmt.Errorf("remove database backup: %w", err)
		}
		removed++
	}
	return removed, nil
}

type databaseBackupFile struct {
	path       string
	modifiedAt time.Time
}

func (s *DatabaseBackupService) StartPeriodic(ctx context.Context) {
	if !s.enabled || s.interval <= 0 {
		return
	}

	go func() {
		ticker := time.NewTicker(s.interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				path, err := s.BackupNow("scheduled")
				if err != nil {
					s.logf("database scheduled backup failed: %v", err)
					continue
				}
				if path != "" {
					s.logf("database scheduled backup created: %s", path)
				}
				if removed, err := s.CleanupOldBackups(); err != nil {
					s.logf("database backup retention cleanup failed: %v", err)
				} else if removed > 0 {
					s.logf("database backup retention removed %d file(s)", removed)
				}
			case <-ctx.Done():
				return
			}
		}
	}()
}

func (s *DatabaseBackupService) backupPath(reason string) string {
	name := backupFilenamePrefix(s.dbPath)
	reason = sanitizeBackupReason(reason)
	if reason != "" {
		name += "-" + reason
	}
	name += "-" + s.now().UTC().Format("20060102-150405-000000000") + ".db"
	return filepath.Join(s.backupDir, name)
}

func backupFilenamePrefix(dbPath string) string {
	base := filepath.Base(strings.TrimSpace(dbPath))
	if base == "" || base == "." || base == string(filepath.Separator) {
		base = "database"
	}
	ext := filepath.Ext(base)
	base = strings.TrimSuffix(base, ext)
	if base == "" {
		return "database"
	}
	return sanitizeBackupReason(base)
}

func sanitizeBackupReason(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	var builder strings.Builder
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z':
			builder.WriteRune(r)
		case r >= '0' && r <= '9':
			builder.WriteRune(r)
		case r == '-' || r == '_':
			builder.WriteRune(r)
		case builder.Len() > 0:
			builder.WriteByte('-')
		}
	}
	return strings.Trim(builder.String(), "-_")
}
