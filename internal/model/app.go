package model

import (
	"os"
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

// IconMode controls which symbol set is used across the interface.
type IconMode int

const (
	IconModeNerd IconMode = iota
	IconModeUnicode
	IconModeASCII
)

var currentIconMode = IconModeNerd

// SetIconMode sets the active global icon set.
func SetIconMode(mode IconMode) {
	currentIconMode = mode
}

// CurrentIconMode returns the active global icon set.
func CurrentIconMode() IconMode {
	return currentIconMode
}

// DetectIconMode resolves an icon configuration string into an IconMode.
func DetectIconMode(val string) IconMode {
	switch strings.ToLower(strings.TrimSpace(val)) {
	case "nerd", "nerdfont", "nerdfonts":
		return IconModeNerd
	case "unicode", "plain", "emoji":
		return IconModeUnicode
	case "ascii", "none", "off":
		return IconModeASCII
	case "auto", "":
		term := strings.ToLower(os.Getenv("TERM"))
		if term == "linux" || term == "dumb" || term == "vt100" || term == "vt220" {
			return IconModeASCII
		}
		return IconModeNerd
	default:
		return IconModeNerd
	}
}

// SourceMeta holds display metadata per source.
type SourceMeta struct {
	Label   string
	Icon    string // active resolved icon
	Nerd    string
	Unicode string
	ASCII   string
	Color   string
	Enabled bool
}

type sourceDef struct {
	label   string
	nerd    string
	unicode string
	ascii   string
	color   string
}

var sourceDefinitions = map[string]sourceDef{
	"apt":      {label: "apt", nerd: "", unicode: "⬡", ascii: "[apt]", color: "#22C55E"},
	"dnf":      {label: "dnf", nerd: "", unicode: "◈", ascii: "[dnf]", color: "#FACC15"},
	"pacman":   {label: "pacman", nerd: "󰮯", unicode: "◎", ascii: "[pac]", color: "#00E5FF"},
	"aur":      {label: "AUR", nerd: "", unicode: "▲", ascii: "[aur]", color: "#A855F7"},
	"zypper":   {label: "zypper", nerd: "", unicode: "◇", ascii: "[zyp]", color: "#38BDF8"},
	"flatpak":  {label: "flatpak", nerd: "", unicode: "◆", ascii: "[flat]", color: "#06B6D4"},
	"snap":     {label: "snap", nerd: "󰏖", unicode: "▣", ascii: "[snap]", color: "#F43F5E"},
	"npm":      {label: "npm -g", nerd: "", unicode: "⬢", ascii: "[npm]", color: "#CBD5E1"},
	"pipx":     {label: "pipx", nerd: "", unicode: "🐍", ascii: "[pip]", color: "#FACC15"},
	"cargo":    {label: "cargo", nerd: "", unicode: "🦀", ascii: "[crg]", color: "#FB923C"},
	"gem":      {label: "gem", nerd: "", unicode: "💎", ascii: "[gem]", color: "#EC4899"},
	"go":       {label: "go install", nerd: "", unicode: "🐹", ascii: "[go]", color: "#00E5FF"},
	"appimage": {label: "AppImage", nerd: "", unicode: "📦", ascii: "[app]", color: "#38BDF8"},
	"brew":     {label: "Homebrew", nerd: "󰏓", unicode: "🍺", ascii: "[brew]", color: "#F59E0B"},
	"nix":      {label: "Nix", nerd: "", unicode: "❄", ascii: "[nix]", color: "#38BDF8"},
	"orphan":   {label: "Orphan", nerd: "󰩈", unicode: "🍂", ascii: "[orph]", color: "#F59E0B"},
	"system":   {label: "System Log", nerd: "󰒋", unicode: "📋", ascii: "[sys]", color: "#A78BFA"},
	"cache":    {label: "Cache & Logs", nerd: "󰃢", unicode: "🧹", ascii: "[cach]", color: "#EAB308"},
}

// SourceMetaByID provides direct access to source metadata.
var SourceMetaByID = map[string]SourceMeta{
	"apt":      {Label: "apt", Icon: "", Color: "#22C55E"},
	"dnf":      {Label: "dnf", Icon: "", Color: "#FACC15"},
	"pacman":   {Label: "pacman", Icon: "󰮯", Color: "#00E5FF"},
	"aur":      {Label: "AUR", Icon: "", Color: "#A855F7"},
	"zypper":   {Label: "zypper", Icon: "", Color: "#38BDF8"},
	"flatpak":  {Label: "flatpak", Icon: "", Color: "#06B6D4"},
	"snap":     {Label: "snap", Icon: "󰏖", Color: "#F43F5E"},
	"npm":      {Label: "npm -g", Icon: "", Color: "#CBD5E1"},
	"pipx":     {Label: "pipx", Icon: "", Color: "#FACC15"},
	"cargo":    {Label: "cargo", Icon: "", Color: "#FB923C"},
	"gem":      {Label: "gem", Icon: "", Color: "#EC4899"},
	"go":       {Label: "go install", Icon: "", Color: "#00E5FF"},
	"appimage": {Label: "AppImage", Icon: "", Color: "#38BDF8"},
	"brew":     {Label: "Homebrew", Icon: "󰏓", Color: "#F59E0B"},
	"nix":      {Label: "Nix", Icon: "", Color: "#38BDF8"},
	"orphan":   {Label: "Orphan", Icon: "󰩈", Color: "#F59E0B"},
	"system":   {Label: "System Log", Icon: "󰒋", Color: "#A78BFA"},
	"cache":    {Label: "Cache & Logs", Icon: "󰃢", Color: "#EAB308"},
}

// Meta returns source display metadata, adapting icon glyphs to the active IconMode.
func Meta(source string) SourceMeta {
	def, ok := sourceDefinitions[source]
	if !ok {
		return SourceMeta{Label: source, Icon: "•", Color: "#94A3B8"}
	}
	icon := def.nerd
	switch currentIconMode {
	case IconModeUnicode:
		icon = def.unicode
	case IconModeASCII:
		icon = def.ascii
	}
	return SourceMeta{
		Label:   def.label,
		Icon:    icon,
		Nerd:    def.nerd,
		Unicode: def.unicode,
		ASCII:   def.ascii,
		Color:   def.color,
	}
}

// UISymbols holds UI glyphs adapted to the current terminal icon mode.
type UISymbols struct {
	Disk     string
	Clock    string
	Sudo     string
	Shield   string
	Warning  string
	Check    string
	Cross    string
	Scanning string
	Pending  string
	Skipped  string
	CardAll  string
	CardFlat string
	CardSnap string
	CardTool string
	CardCach string
	CardRecl string
}

// GetUISymbols returns the symbol set for the active IconMode.
func GetUISymbols() UISymbols {
	switch currentIconMode {
	case IconModeUnicode:
		return UISymbols{
			Disk:     "💾",
			Clock:    "⏱",
			Sudo:     "🔒",
			Shield:   "🛡",
			Warning:  "⚠",
			Check:    "✓",
			Cross:    "✗",
			Scanning: "●",
			Pending:  "○",
			Skipped:  "–",
			CardAll:  "▣",
			CardFlat: "◆",
			CardSnap: "▦",
			CardTool: "⚒",
			CardCach: "🧹",
			CardRecl: "♻",
		}
	case IconModeASCII:
		return UISymbols{
			Disk:     "DISK",
			Clock:    "TIME",
			Sudo:     "[sudo]",
			Shield:   "[SAFE]",
			Warning:  "[!]",
			Check:    "[v]",
			Cross:    "[x]",
			Scanning: "[*]",
			Pending:  "[ ]",
			Skipped:  "[-]",
			CardAll:  "*",
			CardFlat: "#",
			CardSnap: "@",
			CardTool: "&",
			CardCach: "%",
			CardRecl: "$",
		}
	default: // IconModeNerd
		return UISymbols{
			Disk:     "󰋊",
			Clock:    "",
			Sudo:     "",
			Shield:   "󰞀",
			Warning:  "",
			Check:    "",
			Cross:    "",
			Scanning: "",
			Pending:  "",
			Skipped:  "",
			CardAll:  "",
			CardFlat: "",
			CardSnap: "󰏖",
			CardTool: "",
			CardCach: "󰃢",
			CardRecl: "󰚌",
		}
	}
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

// NameVariants returns common variations of an application name for residual file lookup.
// e.g. "org.videolan.VLC" -> ["org.videolan.VLC", "vlc", "VLC"]
// e.g. "google-chrome" -> ["google-chrome", "chrome"]
func NameVariants(name string) []string {
	seen := make(map[string]bool)
	var variants []string
	add := func(s string) {
		s = strings.TrimSpace(s)
		if s == "" || len(s) < 2 || seen[s] {
			return
		}
		seen[s] = true
		variants = append(variants, s)
	}

	add(name)
	add(strings.ToLower(name))

	// Reverse domain naming (e.g. org.videolan.VLC, com.spotify.Client)
	if strings.Contains(name, ".") {
		parts := strings.Split(name, ".")
		last := parts[len(parts)-1]
		add(last)
		add(strings.ToLower(last))
		if len(parts) > 1 {
			penultimate := parts[len(parts)-2]
			add(penultimate)
			add(strings.ToLower(penultimate))
		}
	}

	// Hyphenated names (e.g. google-chrome -> chrome, visual-studio-code -> code)
	if strings.Contains(name, "-") {
		parts := strings.Split(name, "-")
		for _, part := range parts {
			add(part)
			add(strings.ToLower(part))
		}
	}

	return variants
}

// HomeCandidates maps a source and name to extra user-dir and desktop file patterns
// probed during deep-clean staging, relative to the user's home directory.
func HomeCandidates(source string, name string) []string {
	var candidates []string
	seen := make(map[string]bool)
	add := func(p string) {
		if p != "" && !seen[p] {
			seen[p] = true
			candidates = append(candidates, p)
		}
	}

	variants := NameVariants(name)

	for _, v := range variants {
		// App container directories
		add(filepath.Join(".var", "app", v))
		add(filepath.Join("snap", v))

		// XDG directories for variants
		add(filepath.Join(".config", v))
		add(filepath.Join(".cache", v))
		add(filepath.Join(".local", "share", v))
		add(filepath.Join(".local", "state", v))
		add("." + v)

		// Desktop files & launcher residual
		add(filepath.Join(".local", "share", "applications", v+".desktop"))
	}

	if source == "flatpak" {
		add(filepath.Join(".var", "app", name))
	} else if source == "snap" {
		add(filepath.Join("snap", name))
	}

	return candidates
}

// SystemCandidates are root-owned locations probed during staging.
func SystemCandidates(name string) []string {
	var candidates []string
	seen := make(map[string]bool)
	add := func(p string) {
		if p != "" && !seen[p] {
			seen[p] = true
			candidates = append(candidates, p)
		}
	}

	variants := NameVariants(name)
	for _, v := range variants {
		add(filepath.Join("/etc", v))
		add(filepath.Join("/var/log", v))
		add(filepath.Join("/var/cache", v))
		add(filepath.Join("/var/lib", v))
		add(filepath.Join("/usr/share", v))
		add(filepath.Join("/usr/share/applications", v+".desktop"))
	}

	return candidates
}
