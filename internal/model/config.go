package model

import (
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

// Config holds runtime settings loaded via Viper.
type Config struct {
	// ProtectedPackages are extra names never staged for deep clean.
	ProtectedPackages []string
	// Home is the user's home directory (overridable for tests).
	Home string
	// DataDir stores history.log etc.
	DataDir string
	// ConfigDir stores config.yaml.
	ConfigDir string
}

// DefaultConfig returns a Config with standard XDG paths.
func DefaultConfig() *Config {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		home = "/root"
	}
	return &Config{
		Home:              home,
		ConfigDir:         filepath.Join(home, ".config", "veet"),
		DataDir:           filepath.Join(home, ".local", "share", "veet"),
		ProtectedPackages: nil,
	}
}

// Load reads ~/.config/veet/config.yaml if present.
func (c *Config) Load() error {
	v := viper.New()
	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath(c.ConfigDir)
	v.SetDefault("protected_packages", []string{})

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			return nil
		}
		return err
	}
	c.ProtectedPackages = v.GetStringSlice("protected_packages")
	return nil
}

// ProtectedSet merges hardcoded core packages with user config.
func (c *Config) ProtectedSet() map[string]bool {
	set := map[string]bool{}
	for _, name := range []string{
		"kernel", "linux", "linux-lts", "linux-hardened", "linux-headers",
		"linux-lts-headers", "linux-firmware", "linux-azure", "linux-generic",
		"linux-image-generic", "linux-modules-extra", "linux-virtual",
		"glibc", "libc6", "libc6-dev", "systemd", "systemd-sysv",
		"init", "initramfs", "grub", "grub2", "grub2-common",
		"base", "base-devel", "bash", "zsh", "dash", "sh", "coreutils",
		"sudo", "openssl", "openssh", "dbus", "dbus-daemon",
		"xorg-server", "xorg-x11", "wayland", "mesa",
		"gnome-shell", "kde-plasma-desktop", "plasma-desktop", "kwin",
		"kwin-x11", "kwin-wayland", "weston", "sway", "hyprland",
	} {
		set[name] = true
	}
	for _, name := range c.ProtectedPackages {
		set[name] = true
	}
	// The shell currently in use is never a deep-clean target.
	if sh := os.Getenv("SHELL"); sh != "" {
		set[filepath.Base(sh)] = true
	}
	return set
}
