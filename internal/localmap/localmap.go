package localmap

import (
	"encoding/json"
	"os"
	"path/filepath"
)

const FileName = ".mkpdfs.json"

type Entry struct {
	TemplateID      string `json:"templateId"`
	Name            string `json:"name"`
	RemoteUpdatedAt string `json:"remoteUpdatedAt,omitempty"`
}

type Map struct {
	Environment string           `json:"environment"`
	UserID      string           `json:"userId,omitempty"`
	Templates   map[string]Entry `json:"templates"`
}

// Key normalizes a file path to its mapping key (basename).
func Key(path string) string { return filepath.Base(path) }

// Load reads .mkpdfs.json from dir (cwd only — no parent search by design).
func Load(dir string) (*Map, error) {
	m := &Map{Templates: map[string]Entry{}}
	data, err := os.ReadFile(filepath.Join(dir, FileName))
	if err != nil {
		if os.IsNotExist(err) {
			return m, nil
		}
		return nil, err
	}
	if err := json.Unmarshal(data, m); err != nil {
		return nil, err
	}
	if m.Templates == nil {
		m.Templates = map[string]Entry{}
	}
	return m, nil
}

func Save(dir string, m *Map) error {
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, FileName), append(data, '\n'), 0644)
}
