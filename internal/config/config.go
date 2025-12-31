package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/adrg/xdg"
	"github.com/pelletier/go-toml/v2"
)

const appConfigDir = "inbox"

// Account represents a single Gmail account configuration
type Account struct {
	Name    string `toml:"name"`
	Email   string `toml:"email"`
	BadgeFg string `toml:"badge_fg"`
	BadgeBg string `toml:"badge_bg"`
}

// Config represents the inbox configuration
type Config struct {
	Accounts []Account `toml:"accounts"`
	Theme    Theme     `toml:"theme"`
	UI       UIConfig  `toml:"ui"`
	Keys     KeyMap    `toml:"keys"`
}

// ConfigDir returns the directory where config files are stored
func ConfigDir() (string, error) {
	path, err := xdg.ConfigFile(filepath.Join(appConfigDir, "config.toml"))
	if err != nil {
		return "", err
	}
	return filepath.Dir(path), nil
}

// ConfigPath returns the path to the config file
func ConfigPath() (string, error) {
	return xdg.ConfigFile(filepath.Join(appConfigDir, "config.toml"))
}

// TokenPath returns the legacy token file path for a given email.
func TokenPath(email string) (string, error) {
	dir, err := ConfigDir()
	if err != nil {
		return "", err
	}
	tokensDir := filepath.Join(dir, "tokens")
	return filepath.Join(tokensDir, email+".json"), nil
}

// Load reads the config file from disk
func Load() (*Config, error) {
	path, err := ConfigPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			// Return empty config if file doesn't exist
			return &Config{Accounts: []Account{}}, nil
		}
		return nil, err
	}

	var cfg Config
	if err := toml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	return &cfg, nil
}

// Save writes the config to disk
func Save(cfg *Config) error {
	path, err := ConfigPath()
	if err != nil {
		return err
	}

	// Ensure config directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := toml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	return nil
}

// CachePath returns the path to the thread cache file
func CachePath() (string, error) {
	return xdg.CacheFile(filepath.Join(appConfigDir, "threads.json"))
}

// CacheDir returns the directory used for caching.
func CacheDir() (string, error) {
	path, err := xdg.CacheFile(filepath.Join(appConfigDir, "threads.json"))
	if err != nil {
		return "", err
	}
	return filepath.Dir(path), nil
}
