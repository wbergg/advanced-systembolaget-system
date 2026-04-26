package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

type DB struct {
	conn *sql.DB
}

func Open(path string) (*DB, error) {
	if dir := filepath.Dir(path); dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("create db directory: %w", err)
		}
	}

	conn, err := sql.Open("sqlite", path+"?_pragma=journal_mode(wal)&_pragma=foreign_keys(on)")
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}

	if err := initSchema(conn); err != nil {
		conn.Close()
		return nil, fmt.Errorf("init schema: %w", err)
	}

	return &DB{conn: conn}, nil
}

func (db *DB) Close() error {
	return db.conn.Close()
}

func initSchema(conn *sql.DB) error {
	_, err := conn.Exec(`
		CREATE TABLE IF NOT EXISTS users (
			id         INTEGER PRIMARY KEY AUTOINCREMENT,
			username   TEXT NOT NULL UNIQUE,
			password   TEXT NOT NULL,
			role       TEXT NOT NULL DEFAULT 'user' CHECK(role IN ('admin','user')),
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);

		CREATE TABLE IF NOT EXISTS products (
			product_id       TEXT PRIMARY KEY,
			product_number   TEXT,
			name_bold        TEXT NOT NULL,
			name_thin        TEXT,
			producer_name    TEXT,
			price            REAL,
			volume           REAL,
			volume_text      TEXT,
			alcohol_pct      REAL,
			country          TEXT,
			category_level1  TEXT,
			category_level2  TEXT,
			assortment_text  TEXT,
			taste            TEXT,
			usage            TEXT,
			is_organic       INTEGER,
			is_news          INTEGER,
			packaging_level1 TEXT,
			assortment       TEXT DEFAULT '',
			product_launch_date TEXT DEFAULT '',
			restricted_parcel_qty INTEGER DEFAULT 0,
			vintage          TEXT,
			image_url        TEXT,
			synced_at        DATETIME DEFAULT CURRENT_TIMESTAMP
		);

		CREATE TABLE IF NOT EXISTS notes (
			product_id  TEXT PRIMARY KEY REFERENCES products(product_id),
			note        TEXT NOT NULL,
			updated_at  DATETIME DEFAULT CURRENT_TIMESTAMP
		);

		CREATE INDEX IF NOT EXISTS idx_products_category ON products(category_level1);
		CREATE INDEX IF NOT EXISTS idx_products_price ON products(price);

CREATE TABLE IF NOT EXISTS audit_log (
			id         INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id    INTEGER NOT NULL REFERENCES users(id),
			action     TEXT NOT NULL,
			detail     TEXT NOT NULL DEFAULT '',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);

		CREATE INDEX IF NOT EXISTS idx_audit_log_user_id ON audit_log(user_id);
		CREATE INDEX IF NOT EXISTS idx_audit_log_created_at ON audit_log(created_at);

		CREATE TABLE IF NOT EXISTS comments (
			id         INTEGER PRIMARY KEY AUTOINCREMENT,
			product_id TEXT NOT NULL REFERENCES products(product_id),
			user_id    INTEGER NOT NULL REFERENCES users(id),
			comment    TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);

		CREATE INDEX IF NOT EXISTS idx_comments_product_id ON comments(product_id);

		CREATE TABLE IF NOT EXISTS events (
			id          INTEGER PRIMARY KEY AUTOINCREMENT,
			name        TEXT NOT NULL,
			description TEXT NOT NULL DEFAULT '',
			event_date  TEXT NOT NULL DEFAULT '',
			user_id     INTEGER NOT NULL REFERENCES users(id),
			locked      INTEGER NOT NULL DEFAULT 0,
			type        TEXT NOT NULL DEFAULT 'tasting',
			basket_id   INTEGER,
			hidden      INTEGER NOT NULL DEFAULT 0,
			created_at  DATETIME DEFAULT CURRENT_TIMESTAMP
		);

		CREATE TABLE IF NOT EXISTS event_attendees (
			event_id   INTEGER NOT NULL REFERENCES events(id) ON DELETE CASCADE,
			user_id    INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (event_id, user_id)
		);

		CREATE TABLE IF NOT EXISTS event_beers (
			id         INTEGER PRIMARY KEY AUTOINCREMENT,
			event_id   INTEGER NOT NULL REFERENCES events(id) ON DELETE CASCADE,
			product_id TEXT NOT NULL REFERENCES products(product_id),
			added_at   DATETIME DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(event_id, product_id)
		);

		CREATE TABLE IF NOT EXISTS event_scores (
			event_beer_id INTEGER NOT NULL REFERENCES event_beers(id) ON DELETE CASCADE,
			user_id       INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			score         INTEGER NOT NULL CHECK(score >= 0 AND score <= 10),
			updated_at    DATETIME DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (event_beer_id, user_id)
		);

		CREATE TABLE IF NOT EXISTS roll_pool (
			id          INTEGER PRIMARY KEY AUTOINCREMENT,
			event_id    INTEGER NOT NULL REFERENCES events(id) ON DELETE CASCADE,
			product_id  TEXT NOT NULL REFERENCES products(product_id),
			consumed    INTEGER NOT NULL DEFAULT 0,
			consumed_by INTEGER REFERENCES users(id),
			consumed_at DATETIME,
			vetoed      INTEGER NOT NULL DEFAULT 0,
			created_at  DATETIME DEFAULT CURRENT_TIMESTAMP
		);
		CREATE INDEX IF NOT EXISTS idx_roll_pool_event ON roll_pool(event_id);

		CREATE TABLE IF NOT EXISTS roll_turns (
			id          INTEGER PRIMARY KEY AUTOINCREMENT,
			event_id    INTEGER NOT NULL REFERENCES events(id) ON DELETE CASCADE,
			pool_id     INTEGER NOT NULL REFERENCES roll_pool(id),
			user_id     INTEGER NOT NULL REFERENCES users(id),
			status      TEXT NOT NULL DEFAULT 'pending' CHECK(status IN ('pending','accepted','vetoed')),
			created_at  DATETIME DEFAULT CURRENT_TIMESTAMP,
			resolved_at DATETIME
		);
		CREATE INDEX IF NOT EXISTS idx_roll_turns_event ON roll_turns(event_id);

		CREATE TABLE IF NOT EXISTS shared_lists (
			id         INTEGER PRIMARY KEY AUTOINCREMENT,
			uuid       TEXT NOT NULL UNIQUE,
			name       TEXT NOT NULL,
			user_id    INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			locked     INTEGER NOT NULL DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);

		CREATE TABLE IF NOT EXISTS shared_list_items (
			list_id    INTEGER NOT NULL REFERENCES shared_lists(id) ON DELETE CASCADE,
			product_id TEXT NOT NULL REFERENCES products(product_id),
			quantity   INTEGER NOT NULL DEFAULT 1,
			added_by   INTEGER REFERENCES users(id),
			added_at   DATETIME DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (list_id, product_id)
		);

		CREATE TABLE IF NOT EXISTS shared_list_shares (
			list_id    INTEGER NOT NULL REFERENCES shared_lists(id) ON DELETE CASCADE,
			user_id    INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (list_id, user_id)
		);
	`)
	if err != nil {
		return err
	}

	// Migration: add public_token to events (legacy)
	conn.Exec("ALTER TABLE events ADD COLUMN public_token TEXT")
	conn.Exec("CREATE UNIQUE INDEX IF NOT EXISTS idx_events_public_token ON events(public_token)")

	// Migration: add public flag to events (replaces public_token)
	conn.Exec("ALTER TABLE events ADD COLUMN public INTEGER NOT NULL DEFAULT 0")
	conn.Exec("UPDATE events SET public = 1 WHERE public_token IS NOT NULL AND public_token != ''")

	// Migration: persisted accept/veto duration in seconds (1-decimal precision).
	conn.Exec("ALTER TABLE roll_turns ADD COLUMN decision_seconds REAL")

	return nil
}
