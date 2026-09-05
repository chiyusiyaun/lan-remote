package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
)

// Data is persisted app configuration.
type Data struct {
	PIN        string `json:"pin"`
	DeviceName string `json:"device_name"`
	Hub        string `json:"hub"`         // host:port of registry, empty = self
	HTTPPort   int    `json:"http_port"`   // control port
	RegistryPort int  `json:"registry_port"`
	Quality    int    `json:"quality"`
	FPS        int    `json:"fps"`
}

func dir() (string, error) {
	if runtime.GOOS == "windows" {
		base := os.Getenv("APPDATA")
		if base == "" {
			home, _ := os.UserHomeDir()
			base = filepath.Join(home, "AppData", "Roaming")
		}
		return filepath.Join(base, "lan-remote"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "lan-remote"), nil
}

func Path() string {
	d, err := dir()
	if err != nil {
		return "lan-remote.json"
	}
	return filepath.Join(d, "config.json")
}

func Load() (*Data, error) {
	p := Path()
	b, err := os.ReadFile(p)
	if err != nil {
		return &Data{HTTPPort: 8765, RegistryPort: 8760, Quality: 70, FPS: 15}, nil
	}
	var d Data
	if err := json.Unmarshal(b, &d); err != nil {
		return &Data{HTTPPort: 8765, RegistryPort: 8760, Quality: 70, FPS: 15}, nil
	}
	if d.HTTPPort == 0 {
		d.HTTPPort = 8765
	}
	if d.RegistryPort == 0 {
		d.RegistryPort = 8760
	}
	if d.Quality == 0 {
		d.Quality = 70
	}
	if d.FPS == 0 {
		d.FPS = 15
	}
	return &d, nil
}

func Save(d *Data) error {
	dirPath, err := dir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dirPath, 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(d, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(Path(), b, 0o600)
}
