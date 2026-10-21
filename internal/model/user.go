package model

import (
	"strings"
	"time"
)

const (
	maxDisplayNameLength = 100
	maxUsernameLength    = 64
)

type Family struct {
	ID        int64
	Name      string
	IsDefault bool
	CreatedAt time.Time
}

type User struct {
	TelegramID  int64
	FamilyID    int64
	DisplayName string
	Username    string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func ValidateUser(user User) error {
	if user.TelegramID <= 0 {
		return NewValidationError("telegram_id", "must be positive")
	}
	if user.FamilyID <= 0 {
		return NewValidationError("family_id", "must be positive")
	}

	displayName := strings.TrimSpace(user.DisplayName)
	if displayName == "" {
		return NewValidationError("display_name", "must not be empty")
	}
	if len([]rune(displayName)) > maxDisplayNameLength {
		return NewValidationError("display_name", "is too long")
	}
	if len([]rune(user.Username)) > maxUsernameLength {
		return NewValidationError("username", "is too long")
	}

	return nil
}
