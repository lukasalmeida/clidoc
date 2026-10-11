package config

import (
	"errors"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Config representa as opções de configuração do CliDoc.
type Config struct {
	Title   string `yaml:"title"`
	Version string `yaml:"version"`
	BaseURL string `yaml:"base_url"`
	Source  string `yaml:"source"`
	Output  string `yaml:"output"`
	Port    int    `yaml:"port"`
	Host    string `yaml:"host"`
}

func Default() Config {
	return Config{
		Title:   "CliDoc API",
		Version: "1.0.0",
		BaseURL: "http://localhost:8080",
		Source:  ".",
		Output:  "openapi.json",
		Port:    8080,
		Host:    "0.0.0.0",
	}
}

func Load(path string) (Config, error) {
	cfg := Default()
	if path == "" {
		return cfg, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return cfg, nil
		}
		return cfg, fmt.Errorf("read config: %w", err)
	}

	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return cfg, fmt.Errorf("parse config: %w", err)
	}

	if cfg.Title == "" {
		cfg.Title = Default().Title
	}
	if cfg.Version == "" {
		cfg.Version = Default().Version
	}
	if cfg.BaseURL == "" {
		cfg.BaseURL = Default().BaseURL
	}
	if cfg.Source == "" {
		cfg.Source = Default().Source
	}
	if cfg.Output == "" {
		cfg.Output = Default().Output
	}
	if cfg.Port == 0 {
		cfg.Port = Default().Port
	}
	if cfg.Host == "" {
		cfg.Host = Default().Host
	}

	return cfg, nil
}

func WriteDefault(path string) error {
	cfg := Default()
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	return os.WriteFile(path, data, 0o644)
}
