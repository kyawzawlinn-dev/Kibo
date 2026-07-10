package bodyrecord

import (
	"context"
	"database/sql"
	"time"
)

type Repository struct {
	DB *sql.DB
}

// Constructor
func NewRepository(db *sql.DB) *Repository {
	return &Repository{DB: db}
}

// -------------------------------
// User CRUD
// -------------------------------
func (r *Repository) CreateUser(ctx context.Context, username, passwordHash string) (int64, error) {
	res, err := r.DB.ExecContext(ctx, "INSERT INTO users(username, password_hash) VALUES(?, ?)", username, passwordHash)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (r *Repository) GetUserByUsername(ctx context.Context, username string) (*User, error) {
	row := r.DB.QueryRowContext(ctx, "SELECT id, username, password_hash, created_at FROM users WHERE username = ?", username)
	var u User
	if err := row.Scan(&u.ID, &u.Username, &u.PasswordHash, &u.CreatedAt); err != nil {
		return nil, err
	}
	return &u, nil
}

// -------------------------------
// BodyRecord CRUD
// -------------------------------

func (r *Repository) AddBodyRecord(ctx context.Context, br BodyRecord) (int64, error) {
	res, err := r.DB.ExecContext(ctx, "INSERT INTO body_records(user_id, record_type, value, unit, timestamp) VALUES(?,?,?,?,?)",
		br.UserID, br.RecordType, br.Value, br.Unit, br.Timestamp)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (r *Repository) GetBodyRecords(ctx context.Context, userID int64) ([]BodyRecord, error) {
	rows, err := r.DB.QueryContext(ctx, "SELECT id, user_id, record_type, value, unit, timestamp FROM body_records WHERE user_id = ? ORDER BY timestamp DESC", userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []BodyRecord
	for rows.Next() {
		var br BodyRecord
		if err := rows.Scan(&br.ID, &br.UserID, &br.RecordType, &br.Value, &br.Unit, &br.Timestamp); err != nil {
			return nil, err
		}
		result = append(result, br)
	}
	return result, nil
}

// -------------------------------
// DietRecord CRUD
// -------------------------------
func (r *Repository) AddDietRecord(ctx context.Context, dr DietRecord) (int64, error) {
	res, err := r.DB.ExecContext(ctx, "INSERT INTO diet_records(user_id, food_name, calories, protein, carbs, fat, timestamp) VALUES(?,?,?,?,?,?,?)",
		dr.UserID, dr.FoodName, dr.Calories, dr.Protein, dr.Carbs, dr.Fat, dr.Timestamp)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (r *Repository) GetDietRecords(ctx context.Context, userID int64) ([]DietRecord, error) {
	rows, err := r.DB.QueryContext(ctx, "SELECT id, user_id, food_name, calories, protein, carbs, fat, timestamp FROM diet_records WHERE user_id = ? ORDER BY timestamp DESC", userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []DietRecord
	for rows.Next() {
		var dr DietRecord
		if err := rows.Scan(&dr.ID, &dr.UserID, &dr.FoodName, &dr.Calories, &dr.Protein, &dr.Carbs, &dr.Fat, &dr.Timestamp); err != nil {
			return nil, err
		}
		result = append(result, dr)
	}
	return result, nil
}

// -------------------------------
// Chat CRUD
// -------------------------------

// CreateChat creates a new chat room for a user
func (r *Repository) CreateChat(ctx context.Context, userID int64) (int64, error) {
	res, err := r.DB.ExecContext(
		ctx,
		"INSERT INTO chats(user_id, title) VALUES (?, 'New Chat')",
		userID,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// GetChats returns all chats for a user
func (r *Repository) GetChats(ctx context.Context, userID int64) ([]Chat, error) {
	rows, err := r.DB.QueryContext(ctx,
		"SELECT id, user_id, title, created_at, updated_at FROM chats WHERE user_id = ? ORDER BY updated_at DESC",
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []Chat
	for rows.Next() {
		var c Chat
		if err := rows.Scan(&c.ID, &c.UserID, &c.Title, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		result = append(result, c)
	}
	return result, nil
}

// -------------------------------
// Chat History CRUD
// -------------------------------

func (r *Repository) AddChatHistory(ctx context.Context, chatID int64, userID *int64, role, message string) (int64, error) {
	var uid interface{}
	if userID != nil {
		uid = *userID
	} else {
		uid = nil
	}

	res, err := r.DB.ExecContext(ctx,
		"INSERT INTO chat_history(chat_id, user_id, role, message, timestamp) VALUES(?,?,?,?,?)",
		chatID, uid, role, message, time.Now(),
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (r *Repository) GetChatHistory(ctx context.Context, chatID int64, limit int) ([]ChatHistory, error) {
	// Order by id: messages within a chat can share a timestamp, and
	// ids are strictly insertion-ordered.
	rows, err := r.DB.QueryContext(
		ctx,
		"SELECT id, chat_id, user_id, role, message, timestamp FROM chat_history WHERE chat_id = ? ORDER BY id ASC LIMIT ?",
		chatID, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanChatHistory(rows)
}

// GetRecentChatHistory returns the LAST `limit` messages of a chat in
// chronological order — the conversation window the agent feeds to the
// LLM. (GetChatHistory with a limit returns the oldest messages instead.)
func (r *Repository) GetRecentChatHistory(ctx context.Context, chatID int64, limit int) ([]ChatHistory, error) {
	rows, err := r.DB.QueryContext(
		ctx,
		"SELECT id, chat_id, user_id, role, message, timestamp FROM chat_history WHERE chat_id = ? ORDER BY id DESC LIMIT ?",
		chatID, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	history, err := scanChatHistory(rows)
	if err != nil {
		return nil, err
	}

	// reverse newest-first into chronological order
	for i, j := 0, len(history)-1; i < j; i, j = i+1, j-1 {
		history[i], history[j] = history[j], history[i]
	}
	return history, nil
}

func scanChatHistory(rows *sql.Rows) ([]ChatHistory, error) {
	var history []ChatHistory
	for rows.Next() {
		var m ChatHistory
		if err := rows.Scan(&m.ID, &m.ChatID, &m.UserID, &m.Role, &m.Message, &m.Timestamp); err != nil {
			return nil, err
		}
		history = append(history, m)
	}
	return history, rows.Err()
}

// ListChatsByUser returns chats owned by a user (id + title + updated_at)
func (r *Repository) ListChatsByUser(ctx context.Context, userID int64) ([]Chat, error) {
	rows, err := r.DB.QueryContext(ctx, "SELECT id, user_id, title, created_at, updated_at FROM chats WHERE user_id = ? ORDER BY updated_at DESC", userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Chat
	for rows.Next() {
		var c Chat
		if err := rows.Scan(&c.ID, &c.UserID, &c.Title, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, nil
}

// DeleteChat deletes messages and the chat row inside a transaction
func (r *Repository) DeleteChat(ctx context.Context, chatID int64) error {
	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, "DELETE FROM chat_history WHERE chat_id = ?", chatID); err != nil {
		tx.Rollback()
		return err
	}
	if _, err := tx.ExecContext(ctx, "DELETE FROM chats WHERE id = ?", chatID); err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit()
}

// ChatBelongsToUser checks ownership
func (r *Repository) ChatBelongsToUser(ctx context.Context, chatID int64, userID int64) (bool, error) {
	var count int
	err := r.DB.QueryRowContext(ctx, "SELECT COUNT(*) FROM chats WHERE id = ? AND user_id = ?", chatID, userID).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// -------------------------------
// Chat Utilities
// -------------------------------

// IsFirstMessage checks if this is the first user message in this chat
func (r *Repository) IsFirstMessage(chatID int64) (bool, error) {
	count := 0
	err := r.DB.QueryRow("SELECT COUNT(*) FROM chat_history WHERE chat_id = ?", chatID).Scan(&count)
	if err != nil {
		return false, err
	}
	return count == 1, nil
}

func (r *Repository) UpdateChatTitle(chatID int64, title string) error {
	_, err := r.DB.Exec("UPDATE chats SET title = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?", title, chatID)
	return err
}

func (r *Repository) RetrieveChatTitle(chatID int64) (string, error) {
	var title string
	err := r.DB.QueryRow("SELECT title FROM chats WHERE id = ?", chatID).Scan(&title)
	if err != nil {
		return "", err
	}
	return title, nil
}
