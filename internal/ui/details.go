package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/swadhinbiswas/veet/internal/model"
)

// Details renders the right-hand details & preview panel content.
// showPaths controls whether every staged path is listed (confirmation).
func Details(app *model.AppInfo, showPaths bool) string {
	if app == nil {
		return Muted.Render("Select an application to see details.")
	}
	var b strings.Builder

	b.WriteString(Title.Render(app.Name))
	if app.Version != "" && app.Version != "-" && app.Version != "unknown" {
		b.WriteString("  " + Muted.Render("v"+app.Version))
	}
	b.WriteString("\n")
	b.WriteString(SourceBadge(app.Source))
	if app.Protected {
		b.WriteString("  " + DangerText.Render("🛡 SYSTEM PROTECTED"))
	}
	b.WriteString("\n\n")

	if !app.InstalledOn.IsZero() {
		b.WriteString(fmt.Sprintf("%-16s %s\n", Muted.Render("installed"), InfoText.Render(app.InstalledOn.Format("2006-01-02 15:04"))))
	}
	if app.InstallSizeKB > 0 {
		b.WriteString(fmt.Sprintf("%-16s %s\n", Muted.Render("install size"), HighlightText.Render(model.HumanSize(app.InstallSizeKB))))
	}
	if app.Maintainer != "" {
		b.WriteString(fmt.Sprintf("%-16s %s\n", Muted.Render("maintainer"), Sub.Render(Truncate(app.Maintainer, 50))))
	}
	if app.Homepage != "" {
		b.WriteString(fmt.Sprintf("%-16s %s\n", Muted.Render("homepage"), InfoText.Render(Truncate(app.Homepage, 50))))
	}
	if len(app.Dependencies) > 0 {
		b.WriteString(fmt.Sprintf("%-16s %s\n", Muted.Render("dependencies"), Sub.Render(Truncate(strings.Join(app.Dependencies, ", "), 50))))
	}

	rem := app.Removable
	if rem.Staged() {
		b.WriteString("\n" + HighlightText.Render("WILL BE REMOVED:"))
		b.WriteString(removeRow("package files", rem.PackageKB))
		b.WriteString(removeRow("config files", rem.ConfigKB))
		b.WriteString(removeRow("cache files", rem.CacheKB))
		b.WriteString(removeRow("log files", rem.LogKB))
		b.WriteString(removeRow("local data", rem.LocalDataKB))
		b.WriteString(removeRow("residual files", rem.ResidualKB))
		b.WriteString(fmt.Sprintf("\n\n%-18s %s\n", Muted.Render("total reclaimable"), HighlightText.Render(model.HumanSize(rem.TotalKB()))))

		if showPaths && rem.FileCount() > 0 {
			b.WriteString("\n" + Muted.Render("Paths to remove:"))
			for _, p := range rem.Paths {
				line := "  " + p
				if isElevated(p, rem) {
					line += "  " + DangerText.Render("🔒 requires sudo")
				}
				b.WriteString("\n" + line)
			}
		}
	} else if app.Source == "cache" {
		b.WriteString("\n" + Muted.Render("Cache / residual leftover, ready for cleanup."))
	} else {
		b.WriteString("\n" + Muted.Render("Press Enter to preview what will be removed."))
	}

	b.WriteString("\n\n" + DangerText.Render("⚠ Removal cannot be undone"))
	if !app.Protected {
		b.WriteString("\n\n" +
			lipgloss.NewStyle().Bold(true).Foreground(danger).Render("[Enter: Stage Preview]") +
			"  " + lipgloss.NewStyle().Foreground(primary).Render("[Space: Select]"))
	} else {
		b.WriteString("\n\n" + DangerText.Render("Deep clean refused for protected system components."))
	}
	return b.String()
}

func removeRow(label string, kb int64) string {
	if kb <= 0 {
		return "\n  ✗ " + Muted.Render(fmt.Sprintf("%-16s %s", label, "0B"))
	}
	return "\n  ✓ " + SuccessText.Render(fmt.Sprintf("%-16s", label)) + " " + HighlightText.Render(model.HumanSize(kb))
}

func isElevated(p string, rem model.RemovableFiles) bool {
	for _, e := range rem.Elevated {
		if e == p {
			return true
		}
	}
	return false
}
