package model

import (
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type User struct {
	ID             int64
	Login          string
	HashedPassword string
	IsActive       bool
}

type UserSession struct {
	UUID      uuid.UUID
	UserID    int64
	ExpiresAt time.Time
}

func HashUserPassword(pass string) ([]byte, error) {
	return bcrypt.GenerateFromPassword([]byte(pass), bcrypt.DefaultCost)
}

func CompareUserPassword(hashedPass []byte, pass string) error {
	return bcrypt.CompareHashAndPassword(hashedPass, []byte(pass))
}

type UserValidationFunc func(*User) error
