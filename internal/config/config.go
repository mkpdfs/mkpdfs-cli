package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
)

type Creds struct {
	IDToken      string `json:"id_token,omitempty"`
	AccessToken  string `json:"access_token,omitempty"`
	RefreshToken string `json:"refresh_token,omitempty"`
	APIKey       string `json:"api_key,omitempty"`
	LoggedInAt   string `json:"logged_in_at,omitempty"`
}

type Config struct {
	Environment  string            `json:"environment,omitempty"`
	Environments map[string]*Creds `json:"environments,omitempty"`
}

// Creds returns the credentials for env, creating the entry lazily.
func (c *Config) Creds(env string) *Creds {
	if c.Environments == nil {
		c.Environments = map[string]*Creds{}
	}
	if c.Environments[env] == nil {
		c.Environments[env] = &Creds{}
	}
	return c.Environments[env]
}

var (
	instance *Config
	mu       sync.RWMutex
	filePath string
)

func Load() (*Config, error) {
	mu.Lock()
	defer mu.Unlock()
	if instance != nil {
		return instance, nil
	}
	if filePath == "" {
		var err error
		filePath, err = GetConfigPath()
		if err != nil {
			return nil, err
		}
	}
	cfg := &Config{}
	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			instance = cfg
			return instance, nil
		}
		return nil, err
	}
	if len(data) > 0 {
		if err := json.Unmarshal(data, cfg); err != nil {
			return nil, err
		}
	}
	instance = cfg
	return instance, nil
}

func Get() *Config {
	cfg, _ := Load()
	return cfg
}

func SetConfig(cfg *Config) error {
	mu.Lock()
	defer mu.Unlock()
	instance = cfg
	return saveLocked()
}

func saveLocked() error {
	if instance == nil {
		return errors.New("config not loaded")
	}
	if filePath == "" {
		var err error
		filePath, err = GetConfigPath()
		if err != nil {
			return err
		}
	}
	if err := os.MkdirAll(filepath.Dir(filePath), 0700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(instance, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filePath, data, 0600)
}

func Path() string {
	mu.Lock()
	defer mu.Unlock()
	if filePath == "" {
		filePath, _ = GetConfigPath()
	}
	return filePath
}

// MaskToken shows only a safe prefix/suffix — used by `config list` and logs.
func MaskToken(token string) string {
	if token == "" {
		return "(not set)"
	}
	if len(token) <= 8 {
		return "****"
	}
	return token[:4] + "..." + token[len(token)-4:]
}

// test hooks
func overridePathForTest(p string) { mu.Lock(); defer mu.Unlock(); filePath = p; instance = nil }
func resetForTest()                { mu.Lock(); defer mu.Unlock(); instance = nil }
