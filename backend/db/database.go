package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	logger "Kibo/backend/kibo_utils"

	_ "github.com/mattn/go-sqlite3"
)

// NewDB creates a new SQLite database connection and ensures tables exist
func NewDB(dbPath string) (*sql.DB, error) {
	// Ensure the /data directory exists (relative to the executable)
	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		return nil, err
	}

	// Open the database file. It will be created if it doesn't exist.
	db, err := sql.Open("sqlite3", dbPath)
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
	logger.Info("[database.go/createTables]:\ttables created or verified successfully✅")

	return nil
}
