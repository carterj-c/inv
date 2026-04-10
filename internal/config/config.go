package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/carter/inv/internal/model"
	"github.com/pelletier/go-toml/v2"
)

// Dir returns the config directory, respecting XDG_CONFIG_HOME.
func Dir() string {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "invoice")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "invoice")
}

// DataDir returns the data directory within config.
func DataDir() string {
	return filepath.Join(Dir(), "data")
}

// ClientsDir returns the clients directory within config.
func ClientsDir() string {
	return filepath.Join(Dir(), "clients")
}

// PDFArchiveDir returns the tracked PDF archive directory within config data.
func PDFArchiveDir() string {
	return filepath.Join(DataDir(), "pdfs")
}

// EnsureDirs creates the config directory structure if it doesn't exist.
func EnsureDirs() error {
	dirs := []string{
		Dir(),
		DataDir(),
		ClientsDir(),
		PDFArchiveDir(),
	}
	for _, d := range dirs {
		if err := os.MkdirAll(d, 0755); err != nil {
			return fmt.Errorf("creating directory %s: %w", d, err)
		}
	}
	return nil
}

// Exists returns true if the global config file exists.
func Exists() bool {
	_, err := os.Stat(filepath.Join(Dir(), "config.toml"))
	return err == nil
}

// LoadGlobal reads the global config.toml.
func LoadGlobal() (model.GlobalConfig, error) {
	var cfg model.GlobalConfig
	data, err := os.ReadFile(filepath.Join(Dir(), "config.toml"))
	if err != nil {
		return cfg, err
	}
	err = toml.Unmarshal(data, &cfg)
	return cfg, err
}

// SaveGlobal writes the global config.toml.
func SaveGlobal(cfg model.GlobalConfig) error {
	data, err := toml.Marshal(cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(Dir(), "config.toml"), data, 0644)
}

// LoadClient reads a per-client config file.
func LoadClient(slug string) (model.ClientConfig, error) {
	var cfg model.ClientConfig
	data, err := os.ReadFile(filepath.Join(ClientsDir(), slug+".toml"))
	if err != nil {
		return cfg, err
	}
	err = toml.Unmarshal(data, &cfg)
	return cfg, err
}

// SaveClient writes a per-client config file.
func SaveClient(slug string, cfg model.ClientConfig) error {
	data, err := toml.Marshal(cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(ClientsDir(), slug+".toml"), data, 0644)
}

// ListClients returns slugs of all configured clients.
func ListClients() ([]string, error) {
	entries, err := os.ReadDir(ClientsDir())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var slugs []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".toml") {
			slugs = append(slugs, strings.TrimSuffix(e.Name(), ".toml"))
		}
	}
	return slugs, nil
}

// Slugify converts a name to a filename-safe slug.
func Slugify(name string) string {
	s := strings.ToLower(name)
	s = strings.Map(func(r rune) rune {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' {
			return r
		}
		return '-'
	}, s)
	// collapse multiple dashes
	for strings.Contains(s, "--") {
		s = strings.ReplaceAll(s, "--", "-")
	}
	s = strings.Trim(s, "-")
	return s
}

// ExpandPath expands ~ to the user's home directory.
func ExpandPath(path string) string {
	if strings.HasPrefix(path, "~/") {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, path[2:])
	}
	return path
}
