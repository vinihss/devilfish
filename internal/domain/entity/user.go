package entity

import (
	"errors"
	"time"
)

// UserPreference holds user preferences.
type UserPreference struct {
	Theme         string `json:"theme"`         // UI theme preference
	Language      string `json:"language"`      // Preferred language
	Notifications bool   `json:"notifications"` // Enable notifications
}

// ErrInvalidUser is returned when user data is invalid.
var ErrInvalidUser = errors.New("invalid user")

// User represents a user entity in the domain.
type User struct {
	ID        string          `json:"id"`         // Unique identifier
	Name      string          `json:"name"`       // Display name
	Locale    string          `json:"locale"`     // User locale (e.g., "en", "pt")
	Prefs     *UserPreference `json:"prefs"`      // User preferences
	CreatedAt time.Time       `json:"created_at"` // When user was created
	UpdatedAt time.Time       `json:"updated_at"` // Last update time
}

// NewUser creates a new User entity.
func NewUser(id, name, locale string) (*User, error) {
	if id == "" {
		return nil, errors.New("user id is required")
	}
	if name == "" {
		return nil, errors.New("user name is required")
	}
	if locale == "" {
		locale = "en" // Default to English
	}

	now := time.Now()
	return &User{
		ID:        id,
		Name:      name,
		Locale:    locale,
		Prefs:     defaultPreferences(),
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

// defaultPreferences creates default user preferences.
func defaultPreferences() *UserPreference {
	return &UserPreference{
		Theme:         "light",
		Language:      "en",
		Notifications: true,
	}
}

// UpdateName updates the user's display name.
func (u *User) UpdateName(name string) error {
	if name == "" {
		return ErrInvalidUser
	}
	u.Name = name
	u.UpdatedAt = time.Now()
	return nil
}

// SetLocale updates the user's locale.
func (u *User) SetLocale(locale string) {
	if locale == "" {
		return
	}
	u.Locale = locale
	u.UpdatedAt = time.Now()
}

// UpdatePreferences updates the user preferences.
func (u *User) UpdatePreferences(prefs *UserPreference) error {
	if prefs == nil {
		return ErrInvalidUser
	}
	u.Prefs = prefs
	u.UpdatedAt = time.Now()
	return nil
}
