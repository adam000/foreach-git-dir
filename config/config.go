package config

import (
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
)

type Config struct {
	RootDir  string   `json:"rootDir"`
	Excludes []string `json:"excludes"`
	Shell    []string `json:"shell"`
}

// LoadConfig loads configuration from $XDG_CONFIG_HOME(defaulting to ~/.config)/foreach-git-dir/config.json.
// If the file does not exist, returns an empty Config and nil error.
func LoadConfig() (Config, error) {
	var cfg Config

	homeDir, _ := os.UserHomeDir()
	xdgConfigHome := filepath.Join(homeDir, ".config")
	if x := os.Getenv("XDG_CONFIG_HOME"); x != "" {
		xdgConfigHome = x
	}
	cfgPath := filepath.Join(xdgConfigHome, "foreach-git-dir", "config.json")

	if _, err := os.Stat(cfgPath); err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return cfg, err
	}

	slog.Info("Loading config", "path", cfgPath)
	b, err := os.ReadFile(cfgPath)
	if err != nil {
		return cfg, err
	}

	if err := json.Unmarshal(b, &cfg); err != nil {
		return cfg, err
	}

	return cfg, nil
}
