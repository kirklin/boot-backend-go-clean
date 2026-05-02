package entity

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUser_Validate(t *testing.T) {
	tests := []struct {
		name    string
		user    User
		wantErr string
	}{
		{
			name:    "valid user",
			user:    User{Username: "kirk", Email: "kirk@example.com", Password: "securepass"},
			wantErr: "",
		},
		{
			name:    "empty username",
			user:    User{Username: "", Email: "kirk@example.com", Password: "securepass"},
			wantErr: "username cannot be empty",
		},
		{
			name:    "empty email",
			user:    User{Username: "kirk", Email: "", Password: "securepass"},
			wantErr: "email cannot be empty",
		},
		{
			name:    "invalid email format",
			user:    User{Username: "kirk", Email: "notanemail", Password: "securepass"},
			wantErr: "invalid email format",
		},
		{
			name:    "short password",
			user:    User{Username: "kirk", Email: "kirk@example.com", Password: "short"},
			wantErr: "password must be at least 8 characters long",
		},
		{
			name:    "password exactly 8 chars",
			user:    User{Username: "kirk", Email: "kirk@example.com", Password: "12345678"},
			wantErr: "",
		},
		{
			name:    "password exceeds 72 bytes",
			user:    User{Username: "kirk", Email: "kirk@example.com", Password: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"},
			wantErr: "password must not exceed 72 bytes",
		},
		{
			name:    "password exactly 72 bytes",
			user:    User{Username: "kirk", Email: "kirk@example.com", Password: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"},
			wantErr: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.user.Validate()
			if tt.wantErr == "" {
				assert.NoError(t, err)
			} else {
				assert.EqualError(t, err, tt.wantErr)
			}
		})
	}
}

func TestIsValidEmail(t *testing.T) {
	tests := []struct {
		email string
		valid bool
	}{
		{"user@example.com", true},
		{"user.name@example.co", true},
		{"user+tag@example.org", true},
		{"user@sub.domain.com", true},
		{"", false},
		{"notanemail", false},
		{"@example.com", false},
		{"user@", false},
		{"user@.com", false},
	}

	for _, tt := range tests {
		t.Run(tt.email, func(t *testing.T) {
			assert.Equal(t, tt.valid, isValidEmail(tt.email))
		})
	}
}
