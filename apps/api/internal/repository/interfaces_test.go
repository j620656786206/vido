package repository

import (
	"testing"
)

// TestInterfaceImplementations verifies that concrete repository types
// implement their respective interfaces at compile time.
// These tests will fail to compile if the implementations don't satisfy the interfaces.
func TestInterfaceImplementations(t *testing.T) {
	t.Run("MovieRepository implements MovieRepositoryInterface", func(t *testing.T) {
		// This is a compile-time check, not a runtime test
		// The actual verification is done via the type assertions in interfaces.go
		var _ MovieRepositoryInterface = (*MovieRepository)(nil)
	})

	t.Run("SeriesRepository implements SeriesRepositoryInterface", func(t *testing.T) {
		var _ SeriesRepositoryInterface = (*SeriesRepository)(nil)
	})

	t.Run("SettingsRepository implements SettingsRepositoryInterface", func(t *testing.T) {
		var _ SettingsRepositoryInterface = (*SettingsRepository)(nil)
	})

	t.Run("CacheRepository implements CacheRepositoryInterface", func(t *testing.T) {
		var _ CacheRepositoryInterface = (*CacheRepository)(nil)
	})
}
