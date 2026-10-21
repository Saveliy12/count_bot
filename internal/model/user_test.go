package model

import (
	"strings"
	"testing"
)

func TestValidateUser(t *testing.T) {
	valid := User{
		TelegramID:  42,
		FamilyID:    1,
		DisplayName: "Савелий",
		Username:    "saveliy",
	}

	tests := []struct {
		name      string
		mutate    func(user *User)
		wantField string
	}{
		{name: "valid"},
		{
			name: "missing telegram ID",
			mutate: func(user *User) {
				user.TelegramID = 0
			},
			wantField: "telegram_id",
		},
		{
			name: "missing family ID",
			mutate: func(user *User) {
				user.FamilyID = 0
			},
			wantField: "family_id",
		},
		{
			name: "empty display name",
			mutate: func(user *User) {
				user.DisplayName = "   "
			},
			wantField: "display_name",
		},
		{
			name: "display name is too long",
			mutate: func(user *User) {
				user.DisplayName = strings.Repeat("я", maxDisplayNameLength+1)
			},
			wantField: "display_name",
		},
		{
			name: "username is too long",
			mutate: func(user *User) {
				user.Username = strings.Repeat("a", maxUsernameLength+1)
			},
			wantField: "username",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			user := valid
			if test.mutate != nil {
				test.mutate(&user)
			}

			err := ValidateUser(user)
			if test.wantField == "" {
				if err != nil {
					t.Fatalf("validate user: %v", err)
				}
				return
			}

			assertValidationField(t, err, test.wantField)
		})
	}
}
