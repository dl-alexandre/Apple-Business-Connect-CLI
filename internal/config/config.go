package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Config holds the complete application configuration
type Config struct {
	API   APIConfig   `mapstructure:"api"`
	Cache CacheConfig `mapstructure:"cache"`
}

// APIConfig holds API-related configuration
type APIConfig struct {
	URL          string `mapstructure:"url"`
	Timeout      int    `mapstructure:"timeout"`
	ClientID     string `mapstructure:"client_id"`
	ClientSecret string `mapstructure:"client_secret"`
}

// CacheConfig holds caching configuration
type CacheConfig struct {
	Enabled bool          `mapstructure:"enabled"`
	Dir     string        `mapstructure:"dir"`
	TTL     time.Duration `mapstructure:"ttl"`
}

// Flags holds command-line flag values
type Flags struct {
	ConfigFile string
	APIURL     string
	Timeout    int
	NoCache    bool
	CacheDir   string
	CacheTTL   int
	Verbose    bool
	Debug      bool
	Format     string
}

// default values
const (
	DefaultAPIURL     = "https://api.businessconnect.apple.com/v3.0"
	DefaultTimeout    = 30
	DefaultCacheTTL   = 60 * time.Minute
	DefaultConfigName = "config"
	DefaultConfigType = "yaml"
)

// Load loads configuration from file and environment variables
func Load(flags Flags) (*Config, error) {
	v := viper.New()

	// Set defaults
	v.SetDefault("api.url", DefaultAPIURL)
	v.SetDefault("api.timeout", DefaultTimeout)
	v.SetDefault("cache.enabled", true)
	v.SetDefault("cache.ttl", DefaultCacheTTL)

	// Set config file if provided
	if flags.ConfigFile != "" {
		v.SetConfigFile(flags.ConfigFile)
	} else {
		// Look for config in standard locations
		configDir := getConfigDir()
		v.AddConfigPath(configDir)
		v.SetConfigName(DefaultConfigName)
		v.SetConfigType(DefaultConfigType)
	}

	// Read environment variables
	v.SetEnvPrefix("ABC")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// Read config file (optional)
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
		// Config file not found; use defaults and flags
	}

	// Override with flags
	if flags.APIURL != "" {
		v.Set("api.url", flags.APIURL)
	}
	if flags.Timeout != 0 {
		v.Set("api.timeout", flags.Timeout)
	}
	if flags.NoCache {
		v.Set("cache.enabled", false)
	}
	if flags.CacheDir != "" {
		v.Set("cache.dir", flags.CacheDir)
	}
	if flags.CacheTTL != 0 {
		v.Set("cache.ttl", time.Duration(flags.CacheTTL)*time.Minute)
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Set default cache directory if not specified
	if cfg.Cache.Dir == "" {
		cfg.Cache.Dir = filepath.Join(getConfigDir(), "cache")
	}

	// Ensure cache directory exists if caching is enabled
	if cfg.Cache.Enabled {
		if err := os.MkdirAll(cfg.Cache.Dir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create cache directory: %w", err)
		}
	}

	return &cfg, nil
}

// getConfigDir returns the platform-specific config directory
func getConfigDir() string {
	// Check XDG_CONFIG_HOME first
	if xdgConfig := os.Getenv("XDG_CONFIG_HOME"); xdgConfig != "" {
		return filepath.Join(xdgConfig, "abc")
	}

	// Use OS-specific config directory
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}

	return filepath.Join(home, ".config", "abc")
}

// Save saves the configuration to the default config file
func Save(cfg *Config) error {
	v := viper.New()

	// Set values
	v.Set("api", cfg.API)
	v.Set("cache", cfg.Cache)

	// Ensure config directory exists
	configDir := getConfigDir()
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Write config file
	configFile := filepath.Join(configDir, DefaultConfigName+"."+DefaultConfigType)
	v.SetConfigFile(configFile)

	if err := v.WriteConfig(); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	return nil
}

// IsCacheEnabled returns whether caching is enabled
func (c *Config) IsCacheEnabled() bool {
	return c.Cache.Enabled
}

// GetCacheDir returns the cache directory path
func (c *Config) GetCacheDir() string {
	return c.Cache.Dir
}

// GetCacheTTL returns the cache TTL
func (c *Config) GetCacheTTL() time.Duration {
	return c.Cache.TTL
}
