package main

import (
	"bytes"
	"testing"

	"github.com/swadhinbiswas/veet/internal/model"
)

func TestPrintTable(t *testing.T) {
	apps := []model.AppInfo{
		{Name: "neovim", Version: "0.9.5", Source: "pacman", InstallSizeKB: 10240},
		{Name: "vlc", Version: "3.0.18", Source: "flatpak", InstallSizeKB: 204800},
	}
	printTable(apps)
}

func TestPrintJSON(t *testing.T) {
	apps := []model.AppInfo{
		{Name: "neovim", Version: "0.9.5", Source: "pacman", InstallSizeKB: 10240},
	}
	if err := printJSON(apps); err != nil {
		t.Fatalf("unexpected printJSON error: %v", err)
	}
}

func TestRootCmdExecution(t *testing.T) {
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)

	// Test --help flag
	rootCmd.SetArgs([]string{"--help"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("rootCmd --help failed: %v", err)
	}
	if !bytes.Contains(buf.Bytes(), []byte("VEET finds and completely uninstalls")) {
		t.Fatalf("unexpected help output: %s", buf.String())
	}
}

func TestSubcommandsRegistration(t *testing.T) {
	cmds := rootCmd.Commands()
	found := map[string]bool{}
	for _, c := range cmds {
		found[c.Name()] = true
	}
	for _, expected := range []string{"scan", "clean", "orphans", "cache", "history", "tui"} {
		if !found[expected] {
			t.Errorf("expected subcommand %q to be registered", expected)
		}
	}
}
