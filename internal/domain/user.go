package domain

import "time"

type UserID string

type User struct {
	ID           UserID
	Name         string
	Email        string
	PasswordHash string
	CreatedAt    time.Time
}
