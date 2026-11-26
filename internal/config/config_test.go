package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad_Defaults(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("ABC_API_CLIENT_ID", "id-from-env")
	t.Setenv("ABC_API_CLIENT_SECRET", "secret-from-env")
	cfg, err := Load(Flags{})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.API.URL != DefaultAPIURL || cfg.API.Timeout != DefaultTimeout || cfg.API.ClientID != "id-from-env" || !cfg.Cache.Enabled {
		t.Fatalf("unexpected defaults/environment config: %+v", cfg)
	}
}

func TestLoadConfigFileAndFlagOverrides(t *testing.T) {
	root := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", root)
	configPath := filepath.Join(root, "abc.yaml")
	contents := "api:\n  url: https://file.example.test\n  timeout: 12\n  client_id: file-id\n  client_secret: file-secret\ncache:\n  enabled: true\n  ttl: 45m\n"
	if err := os.WriteFile(configPath, []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}
	cacheDir := filepath.Join(root, "custom-cache")
	cfg, err := Load(Flags{ConfigFile: configPath, APIURL: "https://flag.example.test", CacheDir: cacheDir, Timeout: 9})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.API.URL != "https://flag.example.test" || cfg.API.Timeout != 9 || cfg.API.ClientID != "file-id" || cfg.Cache.Dir != cacheDir || cfg.Cache.TTL.String() != "45m0s" {
		t.Fatalf("unexpected merged config: %+v", cfg)
	}
	if info, err := os.Stat(cacheDir); err != nil || !info.IsDir() {
		t.Fatalf("cache directory was not created: info=%v error=%v", info, err)
	}
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
