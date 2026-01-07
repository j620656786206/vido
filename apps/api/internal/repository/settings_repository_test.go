package repository

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/vido/api/internal/models"
	_ "modernc.org/sqlite"
)

// setupSettingsTestDB creates an in-memory database with settings table
func setupSettingsTestDB(t *testing.T) *sql.DB {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}

	// Create settings table
	_, err = db.Exec(`
		CREATE TABLE settings (
			key TEXT PRIMARY KEY,
			value TEXT NOT NULL,
			type TEXT NOT NULL,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create settings table: %v", err)
	}

	return db
}

// TestSettingsSet verifies setting creation
func TestSettingsSet(t *testing.T) {
	db := setupSettingsTestDB(t)
	defer db.Close()

	repo := NewSettingsRepository(db)
	ctx := context.Background()

	setting := &models.Setting{
		Key:   "app_name",
		Value: "Vido",
		Type:  string(models.SettingTypeString),
	}

	err := repo.Set(ctx, setting)
	if err != nil {
		t.Fatalf("Failed to set setting: %v", err)
	}

	// Verify timestamps were set
	if setting.UpdatedAt.IsZero() {
		t.Error("Expected UpdatedAt to be set")
	}

	// Verify setting was inserted
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM settings WHERE key = ?", "app_name").Scan(&count)
	if err != nil {
		t.Fatalf("Failed to count settings: %v", err)
	}
	if count != 1 {
		t.Errorf("Expected 1 setting, got %d", count)
	}
}

// TestSettingsSetNil verifies nil setting rejection
func TestSettingsSetNil(t *testing.T) {
	db := setupSettingsTestDB(t)
	defer db.Close()

	repo := NewSettingsRepository(db)
	ctx := context.Background()

	err := repo.Set(ctx, nil)
	if err == nil {
		t.Fatal("Expected error for nil setting, got nil")
	}
}

// TestSettingsSetEmptyKey verifies empty key rejection
func TestSettingsSetEmptyKey(t *testing.T) {
	db := setupSettingsTestDB(t)
	defer db.Close()

	repo := NewSettingsRepository(db)
	ctx := context.Background()

	setting := &models.Setting{
		Key:   "",
		Value: "test",
		Type:  string(models.SettingTypeString),
	}

	err := repo.Set(ctx, setting)
	if err == nil {
		t.Fatal("Expected error for empty key, got nil")
	}
}

// TestSettingsSetUpsert verifies upsert behavior
func TestSettingsSetUpsert(t *testing.T) {
	db := setupSettingsTestDB(t)
	defer db.Close()

	repo := NewSettingsRepository(db)
	ctx := context.Background()

	// Create setting
	setting := &models.Setting{
		Key:   "test_key",
		Value: "original_value",
		Type:  string(models.SettingTypeString),
	}

	err := repo.Set(ctx, setting)
	if err != nil {
		t.Fatalf("Failed to set setting: %v", err)
	}

	// Wait a bit to ensure updated_at changes
	time.Sleep(10 * time.Millisecond)

	// Update setting
	setting.Value = "updated_value"
	err = repo.Set(ctx, setting)
	if err != nil {
		t.Fatalf("Failed to update setting: %v", err)
	}

	// Verify only one record exists
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM settings WHERE key = ?", "test_key").Scan(&count)
	if err != nil {
		t.Fatalf("Failed to count settings: %v", err)
	}
	if count != 1 {
		t.Errorf("Expected 1 setting, got %d", count)
	}

	// Verify value was updated
	found, err := repo.Get(ctx, "test_key")
	if err != nil {
		t.Fatalf("Failed to get setting: %v", err)
	}
	if found.Value != "updated_value" {
		t.Errorf("Expected value 'updated_value', got '%s'", found.Value)
	}
}

// TestSettingsGet verifies getting a setting
func TestSettingsGet(t *testing.T) {
	db := setupSettingsTestDB(t)
	defer db.Close()

	repo := NewSettingsRepository(db)
	ctx := context.Background()

	// Create setting
	setting := &models.Setting{
		Key:   "test_key",
		Value: "test_value",
		Type:  string(models.SettingTypeString),
	}

	err := repo.Set(ctx, setting)
	if err != nil {
		t.Fatalf("Failed to set setting: %v", err)
	}

	// Get setting
	found, err := repo.Get(ctx, "test_key")
	if err != nil {
		t.Fatalf("Failed to get setting: %v", err)
	}

	if found.Key != setting.Key {
		t.Errorf("Expected key %s, got %s", setting.Key, found.Key)
	}
	if found.Value != setting.Value {
		t.Errorf("Expected value %s, got %s", setting.Value, found.Value)
	}
	if found.Type != setting.Type {
		t.Errorf("Expected type %s, got %s", setting.Type, found.Type)
	}
}

// TestSettingsGetNotFound verifies error for non-existent setting
func TestSettingsGetNotFound(t *testing.T) {
	db := setupSettingsTestDB(t)
	defer db.Close()

	repo := NewSettingsRepository(db)
	ctx := context.Background()

	_, err := repo.Get(ctx, "non_existent")
	if err == nil {
		t.Fatal("Expected error for non-existent setting, got nil")
	}
}

// TestSettingsGetEmptyKey verifies empty key rejection
func TestSettingsGetEmptyKey(t *testing.T) {
	db := setupSettingsTestDB(t)
	defer db.Close()

	repo := NewSettingsRepository(db)
	ctx := context.Background()

	_, err := repo.Get(ctx, "")
	if err == nil {
		t.Fatal("Expected error for empty key, got nil")
	}
}

// TestSettingsGetAll verifies getting all settings
func TestSettingsGetAll(t *testing.T) {
	db := setupSettingsTestDB(t)
	defer db.Close()

	repo := NewSettingsRepository(db)
	ctx := context.Background()

	// Create multiple settings
	settings := []*models.Setting{
		{Key: "setting1", Value: "value1", Type: string(models.SettingTypeString)},
		{Key: "setting2", Value: "value2", Type: string(models.SettingTypeString)},
		{Key: "setting3", Value: "value3", Type: string(models.SettingTypeString)},
	}

	for _, setting := range settings {
		err := repo.Set(ctx, setting)
		if err != nil {
			t.Fatalf("Failed to set setting: %v", err)
		}
	}

	// Get all settings
	all, err := repo.GetAll(ctx)
	if err != nil {
		t.Fatalf("Failed to get all settings: %v", err)
	}

	if len(all) != 3 {
		t.Errorf("Expected 3 settings, got %d", len(all))
	}

	// Verify settings are sorted by key
	for i := 0; i < len(all)-1; i++ {
		if all[i].Key > all[i+1].Key {
			t.Error("Expected settings to be sorted by key")
			break
		}
	}
}

// TestSettingsGetAllEmpty verifies empty list when no settings exist
func TestSettingsGetAllEmpty(t *testing.T) {
	db := setupSettingsTestDB(t)
	defer db.Close()

	repo := NewSettingsRepository(db)
	ctx := context.Background()

	all, err := repo.GetAll(ctx)
	if err != nil {
		t.Fatalf("Failed to get all settings: %v", err)
	}

	if len(all) != 0 {
		t.Errorf("Expected 0 settings, got %d", len(all))
	}
}

// TestSettingsDelete verifies setting deletion
func TestSettingsDelete(t *testing.T) {
	db := setupSettingsTestDB(t)
	defer db.Close()

	repo := NewSettingsRepository(db)
	ctx := context.Background()

	// Create setting
	setting := &models.Setting{
		Key:   "to_delete",
		Value: "value",
		Type:  string(models.SettingTypeString),
	}

	err := repo.Set(ctx, setting)
	if err != nil {
		t.Fatalf("Failed to set setting: %v", err)
	}

	// Delete setting
	err = repo.Delete(ctx, "to_delete")
	if err != nil {
		t.Fatalf("Failed to delete setting: %v", err)
	}

	// Verify setting was deleted
	_, err = repo.Get(ctx, "to_delete")
	if err == nil {
		t.Fatal("Expected error for deleted setting, got nil")
	}
}

// TestSettingsDeleteNotFound verifies error for non-existent setting
func TestSettingsDeleteNotFound(t *testing.T) {
	db := setupSettingsTestDB(t)
	defer db.Close()

	repo := NewSettingsRepository(db)
	ctx := context.Background()

	err := repo.Delete(ctx, "non_existent")
	if err == nil {
		t.Fatal("Expected error for non-existent setting, got nil")
	}
}

// TestSettingsDeleteEmptyKey verifies empty key rejection
func TestSettingsDeleteEmptyKey(t *testing.T) {
	db := setupSettingsTestDB(t)
	defer db.Close()

	repo := NewSettingsRepository(db)
	ctx := context.Background()

	err := repo.Delete(ctx, "")
	if err == nil {
		t.Fatal("Expected error for empty key, got nil")
	}
}

// TestSettingsGetString verifies string value retrieval
func TestSettingsGetString(t *testing.T) {
	db := setupSettingsTestDB(t)
	defer db.Close()

	repo := NewSettingsRepository(db)
	ctx := context.Background()

	// Set string value
	err := repo.SetString(ctx, "string_key", "string_value")
	if err != nil {
		t.Fatalf("Failed to set string: %v", err)
	}

	// Get string value
	value, err := repo.GetString(ctx, "string_key")
	if err != nil {
		t.Fatalf("Failed to get string: %v", err)
	}

	if value != "string_value" {
		t.Errorf("Expected value 'string_value', got '%s'", value)
	}
}

// TestSettingsGetStringWrongType verifies error for wrong type
func TestSettingsGetStringWrongType(t *testing.T) {
	db := setupSettingsTestDB(t)
	defer db.Close()

	repo := NewSettingsRepository(db)
	ctx := context.Background()

	// Set int value
	err := repo.SetInt(ctx, "int_key", 42)
	if err != nil {
		t.Fatalf("Failed to set int: %v", err)
	}

	// Try to get as string
	_, err = repo.GetString(ctx, "int_key")
	if err == nil {
		t.Fatal("Expected error for wrong type, got nil")
	}
}

// TestSettingsGetInt verifies int value retrieval
func TestSettingsGetInt(t *testing.T) {
	db := setupSettingsTestDB(t)
	defer db.Close()

	repo := NewSettingsRepository(db)
	ctx := context.Background()

	// Set int value
	err := repo.SetInt(ctx, "int_key", 42)
	if err != nil {
		t.Fatalf("Failed to set int: %v", err)
	}

	// Get int value
	value, err := repo.GetInt(ctx, "int_key")
	if err != nil {
		t.Fatalf("Failed to get int: %v", err)
	}

	if value != 42 {
		t.Errorf("Expected value 42, got %d", value)
	}
}

// TestSettingsGetIntWrongType verifies error for wrong type
func TestSettingsGetIntWrongType(t *testing.T) {
	db := setupSettingsTestDB(t)
	defer db.Close()

	repo := NewSettingsRepository(db)
	ctx := context.Background()

	// Set string value
	err := repo.SetString(ctx, "string_key", "not_an_int")
	if err != nil {
		t.Fatalf("Failed to set string: %v", err)
	}

	// Try to get as int
	_, err = repo.GetInt(ctx, "string_key")
	if err == nil {
		t.Fatal("Expected error for wrong type, got nil")
	}
}

// TestSettingsGetBool verifies bool value retrieval
func TestSettingsGetBool(t *testing.T) {
	db := setupSettingsTestDB(t)
	defer db.Close()

	repo := NewSettingsRepository(db)
	ctx := context.Background()

	// Set bool value (true)
	err := repo.SetBool(ctx, "bool_key_true", true)
	if err != nil {
		t.Fatalf("Failed to set bool: %v", err)
	}

	// Get bool value (true)
	value, err := repo.GetBool(ctx, "bool_key_true")
	if err != nil {
		t.Fatalf("Failed to get bool: %v", err)
	}

	if !value {
		t.Error("Expected value true, got false")
	}

	// Set bool value (false)
	err = repo.SetBool(ctx, "bool_key_false", false)
	if err != nil {
		t.Fatalf("Failed to set bool: %v", err)
	}

	// Get bool value (false)
	value, err = repo.GetBool(ctx, "bool_key_false")
	if err != nil {
		t.Fatalf("Failed to get bool: %v", err)
	}

	if value {
		t.Error("Expected value false, got true")
	}
}

// TestSettingsGetBoolWrongType verifies error for wrong type
func TestSettingsGetBoolWrongType(t *testing.T) {
	db := setupSettingsTestDB(t)
	defer db.Close()

	repo := NewSettingsRepository(db)
	ctx := context.Background()

	// Set int value
	err := repo.SetInt(ctx, "int_key", 1)
	if err != nil {
		t.Fatalf("Failed to set int: %v", err)
	}

	// Try to get as bool
	_, err = repo.GetBool(ctx, "int_key")
	if err == nil {
		t.Fatal("Expected error for wrong type, got nil")
	}
}

// TestSettingsSetString verifies string setter
func TestSettingsSetString(t *testing.T) {
	db := setupSettingsTestDB(t)
	defer db.Close()

	repo := NewSettingsRepository(db)
	ctx := context.Background()

	err := repo.SetString(ctx, "test", "value")
	if err != nil {
		t.Fatalf("Failed to set string: %v", err)
	}

	setting, err := repo.Get(ctx, "test")
	if err != nil {
		t.Fatalf("Failed to get setting: %v", err)
	}

	if setting.Type != string(models.SettingTypeString) {
		t.Errorf("Expected type string, got %s", setting.Type)
	}
	if setting.Value != "value" {
		t.Errorf("Expected value 'value', got '%s'", setting.Value)
	}
}

// TestSettingsSetInt verifies int setter
func TestSettingsSetInt(t *testing.T) {
	db := setupSettingsTestDB(t)
	defer db.Close()

	repo := NewSettingsRepository(db)
	ctx := context.Background()

	err := repo.SetInt(ctx, "test", 123)
	if err != nil {
		t.Fatalf("Failed to set int: %v", err)
	}

	setting, err := repo.Get(ctx, "test")
	if err != nil {
		t.Fatalf("Failed to get setting: %v", err)
	}

	if setting.Type != string(models.SettingTypeInt) {
		t.Errorf("Expected type int, got %s", setting.Type)
	}
	if setting.Value != "123" {
		t.Errorf("Expected value '123', got '%s'", setting.Value)
	}
}

// TestSettingsSetBool verifies bool setter
func TestSettingsSetBool(t *testing.T) {
	db := setupSettingsTestDB(t)
	defer db.Close()

	repo := NewSettingsRepository(db)
	ctx := context.Background()

	err := repo.SetBool(ctx, "test", true)
	if err != nil {
		t.Fatalf("Failed to set bool: %v", err)
	}

	setting, err := repo.Get(ctx, "test")
	if err != nil {
		t.Fatalf("Failed to get setting: %v", err)
	}

	if setting.Type != string(models.SettingTypeBool) {
		t.Errorf("Expected type bool, got %s", setting.Type)
	}
	if setting.Value != "true" {
		t.Errorf("Expected value 'true', got '%s'", setting.Value)
	}
}

// TestSettingsTypeSafety verifies type-safe getters prevent wrong type access
func TestSettingsTypeSafety(t *testing.T) {
	db := setupSettingsTestDB(t)
	defer db.Close()

	repo := NewSettingsRepository(db)
	ctx := context.Background()

	// Set different types
	repo.SetString(ctx, "str", "text")
	repo.SetInt(ctx, "num", 42)
	repo.SetBool(ctx, "flag", true)

	// Try to get with wrong type
	tests := []struct {
		key      string
		getFunc  func(context.Context, string) (interface{}, error)
		expected bool // should error
	}{
		{"str", func(ctx context.Context, k string) (interface{}, error) { return repo.GetInt(ctx, k) }, true},
		{"str", func(ctx context.Context, k string) (interface{}, error) { return repo.GetBool(ctx, k) }, true},
		{"num", func(ctx context.Context, k string) (interface{}, error) { return repo.GetString(ctx, k) }, true},
		{"num", func(ctx context.Context, k string) (interface{}, error) { return repo.GetBool(ctx, k) }, true},
		{"flag", func(ctx context.Context, k string) (interface{}, error) { return repo.GetString(ctx, k) }, true},
		{"flag", func(ctx context.Context, k string) (interface{}, error) { return repo.GetInt(ctx, k) }, true},
	}

	for _, test := range tests {
		_, err := test.getFunc(ctx, test.key)
		if test.expected && err == nil {
			t.Errorf("Expected error for getting %s with wrong type, got nil", test.key)
		}
	}
}
