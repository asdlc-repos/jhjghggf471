package models

import "time"

type User struct {
	ID           string    `json:"id"`
	Email        string    `json:"email"`
	PasswordHash []byte    `json:"-"`
	CreatedAt    time.Time `json:"createdAt"`
}

type Session struct {
	Token     string
	UserID    string
	ExpiresAt time.Time
}

type Task struct {
	ID          string     `json:"id"`
	UserID      string     `json:"userId"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	DueDate     string     `json:"dueDate,omitempty"`
	CategoryID  string     `json:"categoryId,omitempty"`
	Completed   bool       `json:"completed"`
	CreatedAt   time.Time  `json:"createdAt"`
	CompletedAt *time.Time `json:"completedAt,omitempty"`
}

type Category struct {
	ID        string    `json:"id"`
	UserID    string    `json:"userId"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"createdAt"`
}

type LoginAttempt struct {
	Count       int
	LockedUntil time.Time
}

type PasswordResetToken struct {
	Token     string
	UserID    string
	ExpiresAt time.Time
}
