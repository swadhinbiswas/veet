package model

import (
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// AppInfo describes a single installed application found by a detector.
type AppInfo struct {
	Name          string
	Version       string
	Source        string // "apt", "pacman", "flatpak", "npm", ...
	InstalledOn   time.Time
	InstallSizeKB int64
	Status        string // "Installed", "Removing", "Removed"
	Dependencies  []string
	Maintainer    string
	Homepage      string
	Protected     bool // core system component, deep clean refused
	Removable     RemovableFiles
}

// RemovableFiles lists everything staged for deletion for an app.
type RemovableFiles struct {
	PackageKB   int64
	ConfigKB    int64
	CacheKB     int64
	LogKB       int64
	LocalDataKB int64
	ResidualKB  int64
	Paths       []string // exact paths staged for deletion, shown in preview
	Elevated    []string // subset of Paths that need sudo
}

// Staged reports whether a deep-clean scan has run for this app.
func (r RemovableFiles) Staged() bool { return r.Paths != nil }

// TotalKB is the combined reclaimable size of every category.
func (r RemovableFiles) TotalKB() int64 {
	return r.PackageKB + r.ConfigKB + r.CacheKB + r.LogKB + r.LocalDataKB + r.ResidualKB
}

// FileCount returns the number of staged paths.
func (r RemovableFiles) FileCount() int { return len(r.Paths) }

// AddPath records a staged path, tracking elevation and category.
func (r *RemovableFiles) AddPath(p string, sizeKB int64, elevated bool, category *int64) {
	r.Paths = append(r.Paths, p)
	if elevated {
		r.Elevated = append(r.Elevated, p)
	}
	if category != nil {
		*category += sizeKB
	}
}

// SourceMeta holds display metadata per source.
type SourceMeta struct {
	Label   string // human name, e.g. "AUR (yay)"
	Icon    string
	Color   string // hex used for the badge
	Enabled bool
}

var SourceMetaByID = map[string]SourceMeta{
	"apt":      {Label: "apt", Icon: "⬡", Color: "#22C55E"},
	"dnf":      {Label: "dnf", Icon: "◈", Color: "#FACC15"},
	"pacman":   {Label: "pacman", Icon: "◎", Color: "#00E5FF"},
	"aur":      {Label: "AUR", Icon: "▲", Color: "#A855F7"},
	"zypper":   {Label: "zypper", Icon: "◇", Color: "#38BDF8"},
	"flatpak":  {Label: "flatpak", Icon: "◆", Color: "#06B6D4"},
	"snap":     {Label: "snap", Icon: "▣", Color: "#F43F5E"},
	"npm":      {Label: "npm -g", Icon: "⬢", Color: "#CBD5E1"},
	"pipx":     {Label: "pipx", Icon: "🐍", Color: "#FACC15"},
	"cargo":    {Label: "cargo", Icon: "🦀", Color: "#FB923C"},
	"gem":      {Label: "gem", Icon: "💎", Color: "#EC4899"},
	"go":       {Label: "go install", Icon: "🐹", Color: "#00E5FF"},
	"appimage": {Label: "AppImage", Icon: "📦", Color: "#38BDF8"},
	"orphan":   {Label: "Orphan", Icon: "🍂", Color: "#F59E0B"},
	"system":   {Label: "System Log", Icon: "📋", Color: "#A78BFA"},
	"cache":    {Label: "Cache & Logs", Icon: "🧹", Color: "#EAB308"},
}

// Meta returns source display metadata, falling back to a neutral entry.
func Meta(source string) SourceMeta {
	if m, ok := SourceMetaByID[source]; ok {
		return m
	}
	return SourceMeta{Label: source, Icon: "•", Color: "#94A3B8"}
}

// HumanSize renders a byte/KB value compactly, e.g. "52.3MB".
func HumanSize(kb int64) string {
	if kb < 1 {
		return "0B"
	}
	const (
		mb = 1024
		gb = 1024 * mb
		tb = 1024 * gb
	)
	switch {
	case kb >= tb:
		return trim(fmtFloat(float64(kb)/float64(tb))) + "TB"
	case kb >= gb:
		return trim(fmtFloat(float64(kb)/float64(gb))) + "GB"
	case kb >= mb:
		return trim(fmtFloat(float64(kb)/float64(mb))) + "MB"
	default:
		return fmtInt(kb) + "KB"
	}
}

func fmtFloat(f float64) string {
	return strconv.FormatFloat(f, 'f', 1, 64)
}

func fmtInt(i int64) string {
	return strconv.FormatInt(i, 10)
}

func trim(s string) string {
	return strings.TrimSuffix(s, ".0")
}

// HomeCandidates maps a source to extra user-dir patterns probed during
// deep-clean staging, relative to the user's home directory.
func HomeCandidates(source string, name string) []string {
	switch source {
	case "flatpak":
		return []string{filepath.Join(".var", "app", name)}
	case "snap":
		return []string{filepath.Join("snap", name)}
	}
	return nil
}

// SystemCandidates are root-owned locations probed during staging.
func SystemCandidates(name string) []string {
	return []string{
		filepath.Join("/etc", name),
		filepath.Join("/var/log", name),
		filepath.Join("/usr/share", name),
	}
}
