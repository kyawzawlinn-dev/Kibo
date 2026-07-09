package bodyrecord

import (
	"database/sql"
	"time"
)

type User struct {
	ID           int64     `json:"id"`
	Username     string    `json:"username"`
	PasswordHash string    `json:"password_hash"`
	CreatedAt    time.Time `json:"created_at"`
}

type BodyRecord struct {
	ID         int64     `json:"id"`
	UserID     int64     `json:"user_id"`
	RecordType string    `json:"record_type"`
	Value      float64   `json:"value"`
	Unit       string    `json:"unit"`
	Timestamp  time.Time `json:"timestamp"`
}

type DietRecord struct {
	ID        int64     `json:"id"`
	UserID    int64     `json:"user_id"`
	FoodName  string    `json:"food_name"`
	Calories  int       `json:"calories"`
	Protein   float64   `json:"protein"`
	Carbs     float64   `json:"carbs"`
	Fat       float64   `json:"fat"`
	Timestamp time.Time `json:"timestamp"`
}

// Chat represents a chat session
type Chat struct {
	ID        int64     `json:"id"`
	UserID    int64     `json:"user_id"`
	Title     string    `json:"title"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ChatHistory represents a message in a chat conversation
type ChatHistory struct {
	ID        int64         `json:"id"`
	ChatID    int64         `json:"chat_id"`
	UserID    sql.NullInt64 `json:"user_id"`
	Role      string        `json:"role"`
	Message   string        `json:"message"`
	Timestamp time.Time     `json:"timestamp"`
}

type UserSettings struct {
	UserID   int64  `json:"user_id"`
	Theme    string `json:"theme"`
	Language string `json:"language"`
	Units    string `json:"units"`
}
