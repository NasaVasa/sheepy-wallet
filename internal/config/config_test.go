package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func writeTempConfig(t *testing.T, cfg Config) string {
	t.Helper()

	data, err := json.Marshal(cfg)
	if err != nil {
		t.Fatalf("marshal test config: %v", err)
	}

	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatalf("write temp config: %v", err)
	}

	return path
}

func TestLoad_Success(t *testing.T) {
	want := Config{
		Config: ServerConfig{
			Host: "127.0.0.1",
			Port: 8000,
		},
		Gates: []Gate{
			{Name: "ethereum_sepolia", Mnemonic: "abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about"},
		},
	}

	path := writeTempConfig(t, want)

	got, err := Load(path)
	if err != nil {
		t.Fatalf("Load returned unexpected error: %v", err)
	}

	if got.Config.Host != want.Config.Host {
		t.Errorf("host: got %q, want %q", got.Config.Host, want.Config.Host)
	}
	if got.Config.Port != want.Config.Port {
		t.Errorf("port: got %d, want %d", got.Config.Port, want.Config.Port)
	}
	if len(got.Gates) != len(want.Gates) {
		t.Fatalf("gates count: got %d, want %d", len(got.Gates), len(want.Gates))
	}
	if got.Gates[0].Name != want.Gates[0].Name {
		t.Errorf("gate name: got %q, want %q", got.Gates[0].Name, want.Gates[0].Name)
	}
	if got.Gates[0].Mnemonic != want.Gates[0].Mnemonic {
		t.Errorf("gate mnemonic: got %q, want %q", got.Gates[0].Mnemonic, want.Gates[0].Mnemonic)
	}
}

func TestLoad_FileNotFound(t *testing.T) {
	_, err := Load("/nonexistent/path/config.json")
	if err == nil {
		t.Fatal("expected an error for non-existent config file, got nil")
	}
}

func TestLoad_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	if err := os.WriteFile(path, []byte("{not valid json"), 0600); err != nil {
		t.Fatalf("write temp file: %v", err)
	}

	_, err := Load(path)
	if err == nil {
		t.Fatal("expected an error for invalid JSON, got nil")
	}
}

func TestFindGate_Found(t *testing.T) {
	cfg := &Config{
		Gates: []Gate{
			{Name: "ethereum_sepolia", Mnemonic: "first mnemonic phrase here"},
			{Name: "polygon", Mnemonic: "second mnemonic phrase here"},
		},
	}

	gate, err := cfg.FindGate("ethereum_sepolia")
	if err != nil {
		t.Fatalf("FindGate returned unexpected error: %v", err)
	}
	if gate.Name != "ethereum_sepolia" {
		t.Errorf("gate name: got %q, want %q", gate.Name, "ethereum_sepolia")
	}
	if gate.Mnemonic != "first mnemonic phrase here" {
		t.Errorf("gate mnemonic: got %q, want %q", gate.Mnemonic, "first mnemonic phrase here")
	}

	gate2, _ := cfg.FindGate("polygon")
	if gate2.Name != "polygon" {
		t.Errorf("gate name: got %q, want %q", gate2.Name, "polygon")
	}
}

func TestFindGate_NotFound(t *testing.T) {
	cfg := &Config{
		Gates: []Gate{
			{Name: "ethereum_sepolia", Mnemonic: "some mnemonic"},
		},
	}

	_, err := cfg.FindGate("unknown_gate")
	if err == nil {
		t.Fatal("expected an error for unknown gate, got nil")
	}
}
