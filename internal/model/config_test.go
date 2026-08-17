package model

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg == nil {
		t.Fatal("expected non-nil config")
	}
	if cfg.Home == "" {
		t.Fatal("expected non-empty Home")
	}
	if cfg.Theme != "cyan" {
		t.Fatalf("expected theme cyan, got %s", cfg.Theme)
	}
}

func TestConfigLoadWithFile(t *testing.T) {
	tmpDir := t.TempDir()
	cfgFile := filepath.Join(tmpDir, "config.yaml")
	content := []byte("theme: \"catppuccin\"\nprotected_packages:\n  - my-custom-daemon\n  - secret-tool\n")
	if err := os.WriteFile(cfgFile, content, 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := &Config{ConfigDir: tmpDir}
	if err := cfg.Load(); err != nil {
		t.Fatalf("unexpected error loading config: %v", err)
	}
	if cfg.Theme != "catppuccin" {
		t.Fatalf("expected theme catppuccin, got %s", cfg.Theme)
	}
	if len(cfg.ProtectedPackages) != 2 {
		t.Fatalf("expected 2 protected packages, got %d", len(cfg.ProtectedPackages))
	}
	set := cfg.ProtectedSet()
	if !set["my-custom-daemon"] || !set["secret-tool"] {
		t.Fatal("expected custom protected packages in set")
	}
	if !set["glibc"] || !set["linux"] || !set["systemd"] {
		t.Fatal("expected core system packages in set")
	}
}

func TestRemovableFilesTotalAndAdd(t *testing.T) {
	var rem RemovableFiles
	if rem.Staged() {
		t.Fatal("expected initially unstaged")
	}
	rem.AddPath("/path/to/cfg", 100, false, &rem.ConfigKB)
	rem.AddPath("/var/log/app", 200, true, &rem.LogKB)
	if !rem.Staged() {
		t.Fatal("expected staged after AddPath")
	}
	if rem.FileCount() != 2 {
		t.Fatalf("expected 2 files, got %d", rem.FileCount())
	}
	if len(rem.Elevated) != 1 || rem.Elevated[0] != "/var/log/app" {
		t.Fatalf("unexpected elevated slice: %v", rem.Elevated)
	}
	if rem.TotalKB() != 300 {
		t.Fatalf("expected TotalKB 300, got %d", rem.TotalKB())
	}
}

func TestCandidates(t *testing.T) {
	flatpakC := HomeCandidates("flatpak", "org.videolan.VLC")
	if len(flatpakC) != 1 || flatpakC[0] != filepath.Join(".var", "app", "org.videolan.VLC") {
		t.Fatalf("unexpected flatpak home candidate: %v", flatpakC)
	}
	snapC := HomeCandidates("snap", "spotify")
	if len(snapC) != 1 || snapC[0] != filepath.Join("snap", "spotify") {
		t.Fatalf("unexpected snap home candidate: %v", snapC)
	}
	sysC := SystemCandidates("nginx")
	if len(sysC) < 3 {
		t.Fatalf("expected at least 3 system candidates, got %d", len(sysC))
	}
}

func TestHumanSizeUnits(t *testing.T) {
	cases := []struct {
		kb   int64
		want string
	}{
		{0, "0B"},
		{500, "500KB"},
		{1024, "1MB"},
		{1536, "1.5MB"},
		{1024 * 1024, "1GB"},
		{1024 * 1024 * 1024, "1TB"},
	}
	for _, c := range cases {
		if got := HumanSize(c.kb); got != c.want {
			t.Errorf("HumanSize(%d) = %q, want %q", c.kb, got, c.want)
		}
	}
}
