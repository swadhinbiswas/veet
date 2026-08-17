package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/swadhinbiswas/veet/internal/model"
)

// ThemePalette defines colors for the TUI.
type ThemePalette struct {
	Primary      string
	Secondary    string
	Danger       string
	Success      string
	Highlight    string
	Info         string
	Neutral      string
	MutedColor   string
	BorderColor  string
	BorderActive string
}

var Palettes = map[string]ThemePalette{
	"cyan": {
		Primary:      "#00E5FF",
		Secondary:    "#A855F7",
		Danger:       "#F43F5E",
		Success:      "#22C55E",
		Highlight:    "#FACC15",
		Info:         "#38BDF8",
		Neutral:      "#E2E8F0",
		MutedColor:   "#94A3B8",
		BorderColor:  "#334155",
		BorderActive: "#00E5FF",
	},
	"catppuccin": {
		Primary:      "#89B4FA",
		Secondary:    "#CBA6F7",
		Danger:       "#F38BA8",
		Success:      "#A6E3A1",
		Highlight:    "#F9E2AF",
		Info:         "#89DCEB",
		Neutral:      "#CDD6F4",
		MutedColor:   "#7F849C",
		BorderColor:  "#45475A",
		BorderActive: "#89B4FA",
	},
	"nord": {
		Primary:      "#88C0D0",
		Secondary:    "#B48EAD",
		Danger:       "#BF616A",
		Success:      "#A3BE8C",
		Highlight:    "#EBCB8B",
		Info:         "#81A1C1",
		Neutral:      "#ECEFF4",
		MutedColor:   "#D8DEE9",
		BorderColor:  "#4C566A",
		BorderActive: "#88C0D0",
	},
	"dracula": {
		Primary:      "#8BE9FD",
		Secondary:    "#BD93F9",
		Danger:       "#FF5555",
		Success:      "#50FA7B",
		Highlight:    "#F1FA8C",
		Info:         "#FFB86C",
		Neutral:      "#F8F8F2",
		MutedColor:   "#6272A4",
		BorderColor:  "#44475A",
		BorderActive: "#8BE9FD",
	},
	"gruvbox": {
		Primary:      "#8EC07C",
		Secondary:    "#D3869B",
		Danger:       "#FB4934",
		Success:      "#B8BB26",
		Highlight:    "#FABD2F",
		Info:         "#83A598",
		Neutral:      "#EBDBB2",
		MutedColor:   "#A89984",
		BorderColor:  "#504945",
		BorderActive: "#8EC07C",
	},
	"tokyo-night": {
		Primary:      "#7DCFFF",
		Secondary:    "#BB9AF7",
		Danger:       "#F7768E",
		Success:      "#9ECE6A",
		Highlight:    "#E0AF68",
		Info:         "#7AA2F7",
		Neutral:      "#C0CAF5",
		MutedColor:   "#565F89",
		BorderColor:  "#3B4261",
		BorderActive: "#7DCFFF",
	},
	"monokai": {
		Primary:      "#66D9EF",
		Secondary:    "#AE81FF",
		Danger:       "#F92672",
		Success:      "#A6E22E",
		Highlight:    "#E6DB74",
		Info:         "#FD971F",
		Neutral:      "#F8F8F2",
		MutedColor:   "#75715E",
		BorderColor:  "#49483E",
		BorderActive: "#66D9EF",
	},
}

// Global active palette color variables
var (
	Primary      = Palettes["cyan"].Primary
	Secondary    = Palettes["cyan"].Secondary
	Danger       = Palettes["cyan"].Danger
	Success      = Palettes["cyan"].Success
	Highlight    = Palettes["cyan"].Highlight
	Info         = Palettes["cyan"].Info
	Neutral      = Palettes["cyan"].Neutral
	MutedColor   = Palettes["cyan"].MutedColor
	BorderColor  = Palettes["cyan"].BorderColor
	BorderActive = Palettes["cyan"].BorderActive

	neutral          = lipgloss.Color(Neutral)
	mutedColor       = lipgloss.Color(MutedColor)
	primary          = lipgloss.Color(Primary)
	secondary        = lipgloss.Color(Secondary)
	danger           = lipgloss.Color(Danger)
	success          = lipgloss.Color(Success)
	highlight        = lipgloss.Color(Highlight)
	info             = lipgloss.Color(Info)
	panelBorder      = lipgloss.Color(BorderColor)
	panelBorderFocus = lipgloss.Color(BorderActive)

	Header        = lipgloss.NewStyle().Foreground(primary).Bold(true).Padding(0, 1)
	HeaderMuted   = lipgloss.NewStyle().Foreground(mutedColor).Padding(0, 1)
	Title         = lipgloss.NewStyle().Bold(true).Foreground(primary)
	Sub           = lipgloss.NewStyle().Foreground(neutral)
	Panel         = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(panelBorder).Padding(0, 1)
	PanelFocus    = Panel.Copy().BorderForeground(panelBorderFocus)
	DangerText    = lipgloss.NewStyle().Foreground(danger).Bold(true)
	SuccessText   = lipgloss.NewStyle().Foreground(success).Bold(true)
	HighlightText = lipgloss.NewStyle().Foreground(highlight).Bold(true)
	InfoText      = lipgloss.NewStyle().Foreground(info)
	Muted         = lipgloss.NewStyle().Foreground(mutedColor)
	KeyBadge      = lipgloss.NewStyle().Bold(true).Foreground(primary)
)

// ApplyTheme configures the active color palette and updates all styled components.
func ApplyTheme(themeName string) {
	p, ok := Palettes[strings.ToLower(themeName)]
	if !ok {
		p = Palettes["cyan"]
	}
	Primary = p.Primary
	Secondary = p.Secondary
	Danger = p.Danger
	Success = p.Success
	Highlight = p.Highlight
	Info = p.Info
	Neutral = p.Neutral
	MutedColor = p.MutedColor
	BorderColor = p.BorderColor
	BorderActive = p.BorderActive

	neutral = lipgloss.Color(Neutral)
	mutedColor = lipgloss.Color(MutedColor)
	primary = lipgloss.Color(Primary)
	secondary = lipgloss.Color(Secondary)
	danger = lipgloss.Color(Danger)
	success = lipgloss.Color(Success)
	highlight = lipgloss.Color(Highlight)
	info = lipgloss.Color(Info)
	panelBorder = lipgloss.Color(BorderColor)
	panelBorderFocus = lipgloss.Color(BorderActive)

	Header = lipgloss.NewStyle().Foreground(primary).Bold(true).Padding(0, 1)
	HeaderMuted = lipgloss.NewStyle().Foreground(mutedColor).Padding(0, 1)
	Title = lipgloss.NewStyle().Bold(true).Foreground(primary)
	Sub = lipgloss.NewStyle().Foreground(neutral)
	Panel = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(panelBorder).Padding(0, 1)
	PanelFocus = Panel.Copy().BorderForeground(panelBorderFocus)
	DangerText = lipgloss.NewStyle().Foreground(danger).Bold(true)
	SuccessText = lipgloss.NewStyle().Foreground(success).Bold(true)
	HighlightText = lipgloss.NewStyle().Foreground(highlight).Bold(true)
	InfoText = lipgloss.NewStyle().Foreground(info)
	Muted = lipgloss.NewStyle().Foreground(mutedColor)
	KeyBadge = lipgloss.NewStyle().Bold(true).Foreground(primary)
}

// SourceBadge renders the colored source label with its icon.
func SourceBadge(source string) string {
	meta := model.Meta(source)
	c := lipgloss.Color(meta.Color)
	return lipgloss.NewStyle().
		Bold(true).
		Foreground(c).
		Render(meta.Icon + " " + meta.Label)
}

// StatusCol renders the status indicator.
func StatusCol(status string) string {
	if status == "Installed" {
		return SuccessText.Render("● " + status)
	}
	if status == "Removing" {
		return HighlightText.Render("● " + status)
	}
	if status == "Removed" {
		return Muted.Render("○ Removed")
	}
	return Muted.Render(status)
}

// Boxed renders a framed string using Panel.
func Boxed(s string, focused bool) string {
	if focused {
		return PanelFocus.Render(s)
	}
	return Panel.Render(s)
}

// Truncate keeps a string within n runes, adding an ellipsis.
func Truncate(s string, n int) string {
	if n <= 0 {
		return ""
	}
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	if n <= 1 {
		return "…"
	}
	return string(r[:n-1]) + "…"
}
