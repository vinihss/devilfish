package entity_test

import (
	"testing"

	"devilfish/internal/domain/entity"
)

func TestNewUser_WithValidInput_ReturnsUser(t *testing.T) {
	user, err := entity.NewUser("user-001", "John Doe", "en")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if user == nil {
		t.Fatal("expected user, got nil")
	}
	if user.ID != "user-001" {
		t.Errorf("expected ID user-001, got %s", user.ID)
	}
	if user.Name != "John Doe" {
		t.Errorf("expected Name John Doe, got %s", user.Name)
	}
	if user.Locale != "en" {
		t.Errorf("expected Locale en, got %s", user.Locale)
	}
	if user.Prefs == nil {
		t.Error("expected Prefs to be initialized")
	}
	if user.Prefs.Theme != "light" {
		t.Errorf("expected Theme light, got %s", user.Prefs.Theme)
	}
	if !user.Prefs.Notifications {
		t.Error("expected Notifications to be true")
	}
}

func TestNewUser_WithEmptyID_ReturnsError(t *testing.T) {
	user, err := entity.NewUser("", "John Doe", "en")

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if user != nil {
		t.Errorf("expected nil user, got %v", user)
	}
	if err.Error() != "user id is required" {
		t.Errorf("expected 'user id is required', got %s", err.Error())
	}
}

func TestNewUser_WithEmptyName_ReturnsError(t *testing.T) {
	user, err := entity.NewUser("user-001", "", "en")

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if user != nil {
		t.Errorf("expected nil user, got %v", user)
	}
	if err.Error() != "user name is required" {
		t.Errorf("expected 'user name is required', got %s", err.Error())
	}
}

func TestNewUser_WithEmptyLocale_DefaultsToEnglish(t *testing.T) {
	user, err := entity.NewUser("user-001", "John Doe", "")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if user == nil {
		t.Fatal("expected user, got nil")
	}
	if user.Locale != "en" {
		t.Errorf("expected Locale en, got %s", user.Locale)
	}
}

func TestNewUser_WithPortugueseLocale_ReturnsUser(t *testing.T) {
	user, err := entity.NewUser("user-001", "João Silva", "pt")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if user == nil {
		t.Fatal("expected user, got nil")
	}
	if user.Locale != "pt" {
		t.Errorf("expected Locale pt, got %s", user.Locale)
	}
}

func TestUser_UpdateName_WithValidName_UpdatesName(t *testing.T) {
	user, _ := entity.NewUser("user-001", "John Doe", "en")

	err := user.UpdateName("Jane Doe")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if user.Name != "Jane Doe" {
		t.Errorf("expected Name Jane Doe, got %s", user.Name)
	}
}

func TestUser_UpdateName_WithEmptyName_ReturnsError(t *testing.T) {
	user, _ := entity.NewUser("user-001", "John Doe", "en")

	err := user.UpdateName("")

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err != entity.ErrInvalidUser {
		t.Errorf("expected ErrInvalidUser, got %v", err)
	}
}

func TestUser_SetLocale_WithValidLocale_UpdatesLocale(t *testing.T) {
	user, _ := entity.NewUser("user-001", "John Doe", "en")

	user.SetLocale("pt")

	if user.Locale != "pt" {
		t.Errorf("expected Locale pt, got %s", user.Locale)
	}
}

func TestUser_SetLocale_WithEmptyLocale_NoChange(t *testing.T) {
	user, _ := entity.NewUser("user-001", "John Doe", "en")

	user.SetLocale("")

	if user.Locale != "en" {
		t.Errorf("expected Locale en, got %s", user.Locale)
	}
}

func TestUser_UpdatePreferences_WithValidPrefs_UpdatesPrefs(t *testing.T) {
	user, _ := entity.NewUser("user-001", "John Doe", "en")
	prefs := &entity.UserPreference{
		Theme:         "dark",
		Language:      "pt",
		Notifications: false,
	}

	err := user.UpdatePreferences(prefs)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if user.Prefs.Theme != "dark" {
		t.Errorf("expected Theme dark, got %s", user.Prefs.Theme)
	}
	if user.Prefs.Language != "pt" {
		t.Errorf("expected Language pt, got %s", user.Prefs.Language)
	}
	if user.Prefs.Notifications {
		t.Error("expected Notifications to be false")
	}
}

func TestUser_UpdatePreferences_WithNilPrefs_ReturnsError(t *testing.T) {
	user, _ := entity.NewUser("user-001", "John Doe", "en")

	err := user.UpdatePreferences(nil)

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err != entity.ErrInvalidUser {
		t.Errorf("expected ErrInvalidUser, got %v", err)
	}
}