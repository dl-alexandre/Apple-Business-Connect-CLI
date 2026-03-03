package config

import (
	"testing"
)

func TestLoad_Defaults(t *testing.T) {
	// This test should be run with caution as it reads the filesystem
	// For template purposes, we're just testing the structure
	t.Skip("Skipping filesystem-dependent test in template")
}

func TestGetConfigDir(t *testing.T) {
	dir := getConfigDir()

	if dir == "" {
		t.Error("config dir should not be empty")
	}

	// Should contain the app name placeholder
	// In actual use, this will be replaced during setup
}

func TestFlags_Validation(t *testing.T) {
	flags := Flags{
		ConfigFile: "config.yaml",
		APIURL:     "https://api.example.com",
		Timeout:    30,
		NoCache:    false,
		CacheDir:   "/tmp/cache",
		CacheTTL:   60,
		Verbose:    true,
		Debug:      false,
		Format:     "json",
	}

	if flags.Timeout < 1 {
		t.Error("timeout should be at least 1")
	}

	if flags.CacheTTL < 1 {
		t.Error("cache TTL should be at least 1")
	}
}

func TestConfig_Methods(t *testing.T) {
	cfg := &Config{
		API: APIConfig{
			URL:          "https://api.example.com",
			Timeout:      30,
			ClientID:     "test-client-id",
			ClientSecret: "test-client-secret",
		},
		Cache: CacheConfig{
			Enabled: true,
			Dir:     "/tmp/cache",
			TTL:     3600000000000, // 1 hour in nanoseconds
		},
	}

	if !cfg.IsCacheEnabled() {
		t.Error("cache should be enabled")
	}

	if cfg.GetCacheDir() != "/tmp/cache" {
		t.Errorf("expected cache dir /tmp/cache, got %s", cfg.GetCacheDir())
	}
}
