package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	logger "Kibo/backend/kibo_utils"

	// pure-Go SQLite driver — no CGO / C compiler needed, so the app
	// builds on any machine with just Go (Windows included)
	_ "modernc.org/sqlite"
)

// NewDB creates a new SQLite database connection and ensures tables exist
func NewDB(dbPath string) (*sql.DB, error) {
	// Ensure the /data directory exists (relative to the executable)
	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		return nil, err
	}

	// Open the database file. It will be created if it doesn't exist.
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		logger.Error("[database.go/NewDB]:\tfailed to open database: %v", err)
		return nil, err
	}

	// Check the connection
	if err = db.Ping(); err != nil {
		logger.Error("[database.go/NewDB]:\tfailed to ping database: %v", err)
		return nil, err
	}

	// Create tables if they don't exist
	if err = createTables(db); err != nil {
		logger.Error("[database.go/NewDB]:\tfailed to create tables: %v", err)
		return nil, err
	}

	logger.Info("[database.go/NewDB]:\tdatabase initialized successfully✅")
	return db, nil
}

// -------------------------------
// Create Tables
// -------------------------------
func createTables(db *sql.DB) error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			username TEXT UNIQUE NOT NULL,
			password_hash TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);`,
		`CREATE TABLE IF NOT EXISTS body_records (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			record_type TEXT NOT NULL,
			value REAL NOT NULL,
			unit TEXT NOT NULL,
			timestamp DATETIME NOT NULL,
			FOREIGN KEY (user_id) REFERENCES users(id)
		);`,
		`CREATE TABLE IF NOT EXISTS diet_records (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			food_name TEXT NOT NULL,
			calories INTEGER NOT NULL,
			protein REAL,
			carbs REAL,
			fat REAL,
			timestamp DATETIME NOT NULL,
			FOREIGN KEY (user_id) REFERENCES users(id)
		);`,
		`CREATE TABLE IF NOT EXISTS chats (
    		id INTEGER PRIMARY KEY AUTOINCREMENT,
    		user_id INTEGER NOT NULL,
    		title TEXT NOT NULL DEFAULT 'New Chat',
    		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);`,
		`CREATE TABLE IF NOT EXISTS chat_history (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			chat_id INTEGER NOT NULL,
			user_id INTEGER,
			role TEXT NOT NULL,
			message TEXT NOT NULL,
			timestamp DATETIME NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS health_log (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			date TEXT NOT NULL,
			title TEXT NOT NULL,
			severity TEXT DEFAULT '',
			notes TEXT DEFAULT '',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES users(id)
		);`,
		`CREATE TABLE IF NOT EXISTS user_settings (
			user_id INTEGER PRIMARY KEY,
			theme TEXT DEFAULT 'light',
			language TEXT DEFAULT 'en',
			units TEXT DEFAULT 'metric',
			FOREIGN KEY (user_id) REFERENCES users(id)
		);`,
	}

	for _, q := range queries {
		if _, err := db.Exec(q); err != nil {
			return fmt.Errorf("failed to exec query: %w", err)
		}
	}

	// Seed the default profile on first run. Id 2 matters: data created
	// before profiles existed belongs to user 2, so it stays visible.
	if _, err := db.Exec(
		`INSERT INTO users(id, username, password_hash)
		 SELECT 2, 'Family', '' WHERE NOT EXISTS (SELECT 1 FROM users)`,
	); err != nil {
		return fmt.Errorf("failed to seed default profile: %w", err)
	}

	logger.Info("[database.go/createTables]:\ttables created or verified successfully✅")

	return nil
}
