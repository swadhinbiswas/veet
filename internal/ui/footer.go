package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/swadhinbiswas/veet/internal/history"
	"github.com/swadhinbiswas/veet/internal/model"
)

// QuickActions renders the bottom-right button grid.
func QuickActions(width, elapsed int) string {
	var b strings.Builder
	b.WriteString(HighlightText.Render("QUICK ACTIONS"))
	b.WriteString("\n")
	if width >= 34 {
		grid := [][]string{
			{"1-6", "Cards", "o", "Sort"},
			{"p", "Paths", "c", "Cache"},
			{"e", "Export", "r", "Rescan"},
			{"l", "Logs", "s", "Config"},
		}
		for _, row := range grid {
			col1 := fmt.Sprintf(" %s %-7s", KeyBadge.Render("["+row[0]+"]"), Sub.Render(row[1]))
			col2 := fmt.Sprintf(" %s %s", KeyBadge.Render("["+row[2]+"]"), Sub.Render(row[3]))
			b.WriteString(col1 + col2)
			b.WriteString("\n")
		}
	} else {
		rows := [][]string{
			{"1-6", "Cards"},
			{"o", "Sort"},
			{"p", "Paths"},
			{"c", "Cache"},
		}
		for _, r := range rows {
			b.WriteString(fmt.Sprintf(" %s %s\n", KeyBadge.Render("["+r[0]+"]"), Sub.Render(r[1])))
		}
	}
	if elapsed > 0 {
		b.WriteString(Muted.Render(fmt.Sprintf(" elapsed %02d:%02d", elapsed/60, elapsed%60)))
	} else {
		b.WriteString(Muted.Render(" [h/?] Help  [q] Quit"))
	}
	content := strings.TrimRight(b.String(), "\n")
	if width > 0 {
		return Panel.Width(width).Render(content)
	}
	return Panel.Render(content)
}

// ProgressIdle renders the bottom-left panel when no uninstall is running.
func ProgressIdle(width int, recent []string) string {
	var b strings.Builder
	b.WriteString(HighlightText.Render("PROGRESS & ACTIVITY"))
	b.WriteString("\n")
	b.WriteString(Sub.Render("● System Ready — Space to select, Enter to preview"))
	b.WriteString("\n")
	if len(recent) > 0 {
		b.WriteString(Muted.Render("Recent:"))
		b.WriteString("\n")
		count := 3
		if len(recent) < count {
			count = len(recent)
		}
		for i := len(recent) - count; i < len(recent); i++ {
			b.WriteString("  " + Muted.Render("• "+Truncate(recent[i], max(10, width-8))))
			b.WriteString("\n")
		}
		// Pad if fewer than 3 recent items to keep steady height
		for i := 0; i < 3-count; i++ {
			b.WriteString("\n")
		}
	} else {
		b.WriteString(Muted.Render("No recent actions.\n\n\n"))
	}
	content := strings.TrimRight(b.String(), "\n")
	if width > 0 {
		return Panel.Width(width).Render(content)
	}
	return Panel.Render(content)
}

// FooterKeybinds is the single-line binding strip.
func FooterKeybinds(width int) string {
	items := []string{
		KeyBadge.Render("1-6") + " " + Sub.Render("Cards"),
		KeyBadge.Render("↑↓/jk") + " " + Sub.Render("Nav"),
		KeyBadge.Render("Space") + " " + Sub.Render("Sel"),
		KeyBadge.Render("Enter") + " " + Sub.Render("Action"),
		KeyBadge.Render("o") + " " + Sub.Render("Sort"),
		KeyBadge.Render("p") + " " + Sub.Render("Paths"),
		KeyBadge.Render("Tab") + " " + Sub.Render("Source"),
		KeyBadge.Render("/") + " " + Sub.Render("Search"),
		KeyBadge.Render("c") + " " + Sub.Render("Clean"),
		KeyBadge.Render("e") + " " + Sub.Render("Export"),
		KeyBadge.Render("l") + " " + Sub.Render("Logs"),
		KeyBadge.Render("h") + " " + Sub.Render("Help"),
		KeyBadge.Render("q") + " " + Sub.Render("Quit"),
	}
	line := " " + strings.Join(items, "  ") + " "
	if width < 40 {
		width = 40
	}
	return lipgloss.NewStyle().Width(width).Render(Truncate(line, width))
}

// helpKeys renders the full keybinding reference.
func helpKeys() string {
	rows := [][]string{
		{"1 - 6", "Jump to stat category (All, Flatpak, Snap, Tools, Cache, Reclaim)"},
		{"[ / ] or ← / →", "Cycle through top stat cards"},
		{"↑ ↓ / j k", "Navigate the app list"},
		{"Space", "Toggle selection on current row"},
		{"a", "Select / deselect all filtered rows"},
		{"o / O", "Cycle sort order (Size ↓, Name A-Z, Source, Date ↓)"},
		{"p / P", "Toggle staged file paths inspector in details panel"},
		{"/", "Focus fuzzy search input"},
		{"Tab / Shift+Tab", "Cycle specific source filter (pacman, AUR, flatpak, etc.)"},
		{"Enter", "Open details & preview, or confirm uninstall"},
		{"c", "Quick clean cache & residual files"},
		{"e", "Export installed applications report (JSON)"},
		{"r", "Refresh / rescan all sources"},
		{"l", "View history audit log"},
		{"s", "Settings & protected system components"},
		{"h / ?", "Toggle help reference"},
		{"Esc", "Back out of modal or exit search"},
		{"q / Ctrl+C", "Quit VEET"},
	}
	var b strings.Builder
	for _, r := range rows {
		b.WriteString("  " + lipgloss.NewStyle().Bold(true).Foreground(primary).Render(r[0]))
		b.WriteString(strings.Repeat(" ", max(1, 28-lipgloss.Width(r[0]))))
		b.WriteString(Sub.Render(r[1]))
		b.WriteString("\n")
	}
	return b.String()
}

// renderHistory formats history entries for the viewport.
func renderHistory(entries []history.Entry, err error) string {
	var b strings.Builder
	if err != nil {
		b.WriteString(DangerText.Render("failed to read history: " + err.Error()))
		return b.String()
	}
	if len(entries) == 0 {
		b.WriteString(Muted.Render("No uninstall history yet."))
		return b.String()
	}
	for _, e := range entries {
		status := SuccessText.Render("ok")
		if e.Status == "failed" {
			status = DangerText.Render("failed")
		}
		b.WriteString(fmt.Sprintf("%s  %s  %s  %s  freed %s  %d file(s)",
			Muted.Render(e.Time.Format("2006-01-02 15:04:05")),
			Title.Render(e.App),
			SourceBadge(e.Source),
			status,
			HighlightText.Render(model.HumanSize(e.FreedKB)),
			e.Files))
		if e.Detail != "" {
			b.WriteString("  " + Muted.Render(e.Detail))
		}
		b.WriteString("\n\n")
	}
	return b.String()
}
