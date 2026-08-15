package ui

import (
	"github.com/charmbracelet/lipgloss"

	"github.com/swadhinbiswas/veet/internal/model"
)

// Curated modern color palette.
const (
	Primary      = "#00E5FF" // Neon Cyan: logo, headers, active borders
	Secondary    = "#A855F7" // Purple: accents, aur, sub-headers
	Danger       = "#F43F5E" // Rose / Red: remove actions, warnings, protected
	Success      = "#22C55E" // Emerald Green: installed status, confirmations, ok
	Highlight    = "#FACC15" // Amber / Gold: reclaimable size, stats, cache
	Info         = "#38BDF8" // Sky Blue: URLs, filters, details
	Neutral      = "#E2E8F0" // Slate 200: body text, primary labels
	MutedColor   = "#94A3B8" // Slate 400: muted captions, secondary info
	BorderColor  = "#334155" // Slate 700: clear, elegant unfocused borders
	BorderActive = "#00E5FF" // Primary Cyan: focused panel borders
)

var (
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
)

var (
	Header = lipgloss.NewStyle().
		Foreground(primary).
		Bold(true).
		Padding(0, 1)

	HeaderMuted = lipgloss.NewStyle().
		Foreground(mutedColor).
		Padding(0, 1)

	Title = lipgloss.NewStyle().Bold(true).Foreground(primary)

	Sub = lipgloss.NewStyle().Foreground(neutral)

	Panel = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(panelBorder).
		Padding(0, 1)

	PanelFocus = Panel.Copy().BorderForeground(panelBorderFocus)

	DangerText = lipgloss.NewStyle().Foreground(danger).Bold(true)

	SuccessText = lipgloss.NewStyle().Foreground(success).Bold(true)

	HighlightText = lipgloss.NewStyle().Foreground(highlight).Bold(true)

	InfoText = lipgloss.NewStyle().Foreground(info)

	Muted = lipgloss.NewStyle().Foreground(mutedColor)

	KeyBadge = lipgloss.NewStyle().
		Bold(true).
		Foreground(primary)
)

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
