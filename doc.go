// Package main provides the entry point and CLI commands for VEET.
//
// VEET (Universal Linux App Uninstaller & Deep-Clean Residual Purger) is a modern,
// zero-latency terminal user interface (TUI) and headless CLI utility for Linux.
// It aggregates installed software across 15+ package managers, universal container
// formats, portable AppImages, and language toolchains, staging each candidate
// residual path (~/.config, ~/.cache, ~/.local/share, /etc, /var/log) for user inspection
// before performing atomic removals with batched privilege elevation.
//
// Installation:
//
//	go install github.com/swadhinbiswas/veet@latest
//
// Usage:
//
//	veet         # Launch interactive terminal UI (default)
//	veet scan    # List all installed applications across all package managers
//	veet scan --json # Output machine-readable JSON inventory
//	veet clean <app> # Deep-clean uninstall an application with staged preview
package main
