package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Sync           SyncConfig `yaml:"sync"`
	DefaultBundles []string   `yaml:"default_bundles,omitempty"`
}

type SyncConfig struct {
	Remote string `yaml:"remote,omitempty"`
}

func DefaultConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "skl", "config.yaml"), nil
}

func Load() (*Config, error) {
	path, err := DefaultConfigPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return createDefault(path)
		}
		return nil, fmt.Errorf("reading config: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}
	return &cfg, nil
}

func Save(cfg *Config) error {
	path, err := DefaultConfigPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}

	header := "# skl configuration\n# See: skl config\n\n"
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, []byte(header+string(data)), 0o644); err != nil {
		return fmt.Errorf("writing config: %w", err)
	}
	return os.Rename(tmp, path)
}

func createDefault(path string) (*Config, error) {
	cfg := &Config{}
	if err := Save(cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}
