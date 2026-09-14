package store

import (
	"database/sql"
	"fmt"
)

func Migrate(db *sql.DB) error {
	statements := []string{
		"PRAGMA journal_mode = WAL",
		"PRAGMA foreign_keys = ON",
		"PRAGMA busy_timeout = 5000",
		`CREATE TABLE IF NOT EXISTS users (
            id TEXT PRIMARY KEY,
            openid TEXT NOT NULL UNIQUE,
            nickname TEXT NOT NULL DEFAULT '',
            created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
        )`,
		`CREATE TABLE IF NOT EXISTS foods (
            id TEXT PRIMARY KEY,
            user_id TEXT NOT NULL REFERENCES users(id),
            family_id TEXT REFERENCES families(id),
            name TEXT NOT NULL,
            barcode TEXT NOT NULL DEFAULT '',
            category TEXT NOT NULL DEFAULT 'other',
            storage_location TEXT NOT NULL DEFAULT 'fridge',
            quantity REAL NOT NULL DEFAULT 1,
            unit TEXT NOT NULL DEFAULT 'item',
            expiry_date DATE NOT NULL,
            status TEXT NOT NULL DEFAULT 'active' CHECK(status IN ('active','consumed','discarded')),
            created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
            updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
        )`,
		`CREATE TABLE IF NOT EXISTS families (
            id TEXT PRIMARY KEY,
            name TEXT NOT NULL,
            invite_code TEXT NOT NULL UNIQUE,
            created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
        )`,
		`CREATE TABLE IF NOT EXISTS family_members (
            family_id TEXT NOT NULL REFERENCES families(id) ON DELETE CASCADE,
            user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
            role TEXT NOT NULL DEFAULT 'member' CHECK(role IN ('owner','member')),
            created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
            PRIMARY KEY (family_id, user_id),
            UNIQUE (user_id)
        )`,
		`CREATE INDEX IF NOT EXISTS idx_family_members_family ON family_members(family_id)`,
		`CREATE INDEX IF NOT EXISTS idx_foods_user_status_expiry ON foods(user_id, status, expiry_date)`,
		`CREATE TABLE IF NOT EXISTS reminder_jobs (
            id TEXT PRIMARY KEY,
            food_id TEXT NOT NULL REFERENCES foods(id),
            user_id TEXT NOT NULL REFERENCES users(id),
            remind_at DATETIME NOT NULL,
            template_id TEXT NOT NULL DEFAULT '',
            status TEXT NOT NULL DEFAULT 'pending' CHECK(status IN ('pending','sent','failed','cancelled')),
            sent_at DATETIME,
            error_message TEXT NOT NULL DEFAULT '',
            created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
        )`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_reminder_unique_pending ON reminder_jobs(food_id) WHERE status = 'pending'`,
		`CREATE INDEX IF NOT EXISTS idx_reminder_due ON reminder_jobs(status, remind_at)`,
	}
	for _, statement := range statements {
		if _, err := db.Exec(statement); err != nil {
			return fmt.Errorf("migrate database: %w", err)
		}
	}
	if err := ensureColumn(db, "users", "avatar_url", "TEXT NOT NULL DEFAULT ''"); err != nil {
		return err
	}
	if err := ensureColumn(db, "foods", "family_id", "TEXT REFERENCES families(id)"); err != nil {
		return err
	}
	if _, err := db.Exec("CREATE INDEX IF NOT EXISTS idx_foods_family_status_expiry ON foods(family_id, status, expiry_date)"); err != nil {
		return fmt.Errorf("create family food index: %w", err)
	}
	return nil
}

func ensureColumn(db *sql.DB, table, column, definition string) error {
	rows, err := db.Query("PRAGMA table_info(" + table + ")")
	if err != nil {
		return fmt.Errorf("inspect %s: %w", table, err)
	}
	defer rows.Close()
	for rows.Next() {
		var cid int
		var name, dataType string
		var notNull int
		var defaultValue any
		var primaryKey int
		if err := rows.Scan(&cid, &name, &dataType, &notNull, &defaultValue, &primaryKey); err != nil {
			return fmt.Errorf("inspect %s columns: %w", table, err)
		}
		if name == column {
			return nil
		}
	}
	if _, err := db.Exec("ALTER TABLE " + table + " ADD COLUMN " + column + " " + definition); err != nil {
		return fmt.Errorf("add %s.%s: %w", table, column, err)
	}
	return nil
}
