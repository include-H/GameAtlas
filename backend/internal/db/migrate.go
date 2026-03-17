package db

import (
	"database/sql"
	"fmt"
	"sort"
	"strings"

	"github.com/jmoiron/sqlx"

	embeddedmigrations "github.com/hao/game/migrations"
)

func RunMigrations(db *sqlx.DB) error {
	if err := ensureMigrationTable(db); err != nil {
		return err
	}

	files, err := embeddedmigrations.Files.ReadDir(".")
	if err != nil {
		return fmt.Errorf("read embedded migrations: %w", err)
	}

	names := make([]string, 0, len(files))
	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".sql") {
			continue
		}
		names = append(names, file.Name())
	}
	sort.Strings(names)

	for _, name := range names {
		applied, err := hasMigration(db, name)
		if err != nil {
			return err
		}
		if applied {
			continue
		}

		content, err := embeddedmigrations.Files.ReadFile(name)
		if err != nil {
			return fmt.Errorf("read migration %s: %w", name, err)
		}

		tx, err := db.Beginx()
		if err != nil {
			return fmt.Errorf("begin migration %s: %w", name, err)
		}

		if _, err := tx.Exec(string(content)); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("execute migration %s: %w", name, err)
		}

		if _, err := tx.Exec("INSERT INTO schema_migrations (name) VALUES (?)", name); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("record migration %s: %w", name, err)
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("commit migration %s: %w", name, err)
		}
	}

	return nil
}

func ensureMigrationTable(db *sqlx.DB) error {
	const query = `
	CREATE TABLE IF NOT EXISTS schema_migrations (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL UNIQUE,
		applied_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
	);`

	if _, err := db.Exec(query); err != nil {
		return fmt.Errorf("ensure schema_migrations table: %w", err)
	}

	return nil
}

func hasMigration(db *sqlx.DB, name string) (bool, error) {
	const query = `SELECT 1 FROM schema_migrations WHERE name = ? LIMIT 1`

	var exists int
	err := db.Get(&exists, query, name)
	if err == nil {
		return true, nil
	}
	if err == sql.ErrNoRows {
		return false, nil
	}

	return false, fmt.Errorf("check migration %s: %w", name, err)
}
