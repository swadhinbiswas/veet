package ui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/swadhinbiswas/veet/internal/model"
)

// AppTable wraps the bubbles table with checkbox + source badge columns.
type AppTable struct {
	table table.Model
	// apps mirrors the filtered rows so selection can map back to AppInfo.
	apps []*model.AppInfo
	// selected tracks chosen apps by name.
	selected map[string]bool
	lastW    int
	lastH    int
}

// NewAppTable builds the main list.
func NewAppTable() *AppTable {
	cols := defaultColumns(80)
	km := table.DefaultKeyMap()
	km.PageDown = key.NewBinding(
		key.WithKeys("down", "pgdown"),
		key.WithHelp("↓/pgdn", "page down"),
	)
	t := table.New(
		table.WithColumns(cols),
		table.WithHeight(8),
		table.WithFocused(true),
		table.WithKeyMap(km),
	)
	s := table.DefaultStyles()
	s.Header = lipgloss.NewStyle().Bold(true).Foreground(primary).Padding(0, 1)
	s.Selected = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFFFFF")).
		Background(lipgloss.Color("#1E3A8A")).
		Bold(true)
	s.Cell = lipgloss.NewStyle().Foreground(neutral).Padding(0, 1)
	t.SetStyles(s)
	return &AppTable{table: t, selected: map[string]bool{}}
}

func defaultColumns(width int) []table.Column {
	avail := width - 6
	checkW := 3
	sourceW := 12
	dateW := 12
	sizeW := 10
	verW := 12
	if avail > 75 {
		verW = 16
	}
	nameW := avail - checkW - verW - sourceW - dateW - sizeW - 6
	if nameW < 18 {
		nameW = 18
	}

	return []table.Column{
		{Title: "", Width: checkW},
		{Title: "App/Package", Width: nameW},
		{Title: "Version", Width: verW},
		{Title: "Source", Width: sourceW},
		{Title: "Installed", Width: dateW},
		{Title: "Size", Width: sizeW},
	}
}

// SetApps replaces the visible (filtered) rows.
func (a *AppTable) SetApps(apps []*model.AppInfo) {
	a.apps = apps
	rows := make([]table.Row, len(apps))
	for i, app := range apps {
		rows[i] = a.row(app)
	}
	a.table.SetRows(rows)
	if a.table.Cursor() >= len(rows) && len(rows) > 0 {
		a.table.SetCursor(len(rows) - 1)
	}
	if len(rows) == 0 {
		a.table.SetCursor(0)
	}
	a.table.UpdateViewport()
}

func (a *AppTable) row(app *model.AppInfo) table.Row {
	check := "[ ]"
	if a.selected[app.Name] {
		check = "[✓]"
	}
	name := app.Name
	if app.Protected {
		name = "🛡 " + name
	}
	inst := "-"
	if !app.InstalledOn.IsZero() {
		inst = app.InstalledOn.Format("2006-01-02")
	}
	return table.Row{
		check,
		name,
		app.Version,
		model.Meta(app.Source).Label,
		inst,
		model.HumanSize(app.InstallSizeKB),
	}
}

// Selected reports whether the app at a given row index is checked.
func (a *AppTable) IsSelected(i int) bool {
	if i < 0 || i >= len(a.apps) {
		return false
	}
	return a.selected[a.apps[i].Name]
}

// Toggle flips selection for the current row; returns the app.
func (a *AppTable) Toggle() *model.AppInfo {
	app := a.Current()
	if app == nil {
		return nil
	}
	if app.Protected {
		return app // caller shows refusal note
	}
	if a.selected[app.Name] {
		delete(a.selected, app.Name)
	} else {
		a.selected[app.Name] = true
	}
	a.SetApps(a.apps)
	return app
}

// SelectAll checks every visible (filtered) app; protected ones are skipped.
// Returns the number of protected apps that were refused.
func (a *AppTable) SelectAll() int {
	refused := 0
	all := true
	for _, app := range a.apps {
		if app.Protected {
			refused++
			continue
		}
		if !a.selected[app.Name] {
			all = false
		}
	}
	if all {
		for _, app := range a.apps {
			if !app.Protected {
				delete(a.selected, app.Name)
			}
		}
	} else {
		for _, app := range a.apps {
			if !app.Protected {
				a.selected[app.Name] = true
			}
		}
	}
	a.SetApps(a.apps)
	return refused
}

// Current returns the app under the cursor.
func (a *AppTable) Current() *model.AppInfo {
	i := a.table.Cursor()
	if i < 0 || i >= len(a.apps) {
		return nil
	}
	return a.apps[i]
}

// SelectedApps returns all checked apps in row order.
func (a *AppTable) SelectedApps() []*model.AppInfo {
	var out []*model.AppInfo
	for _, app := range a.apps {
		if a.selected[app.Name] {
			out = append(out, app)
		}
	}
	return out
}

// Count returns the number of visible rows.
func (a *AppTable) Count() int { return len(a.apps) }

// Model exposes the underlying bubbles table.
func (a *AppTable) Model() *table.Model { return &a.table }

// Update forwards a message to the bubbles table, keeping the returned model.
func (a *AppTable) Update(msg tea.Msg) tea.Cmd {
	t, cmd := a.table.Update(msg)
	a.table = t
	return cmd
}

// Summary renders "N selected / Total / Reclaimable".
func (a *AppTable) Summary(extra string) string {
	sel := a.SelectedApps()
	total := int64(0)
	reclaim := int64(0)
	for _, app := range sel {
		total += app.InstallSizeKB
		if app.Removable.Staged() {
			reclaim += app.Removable.TotalKB()
		}
	}
	selCount := fmt.Sprintf("%d selected", len(sel))
	if len(sel) > 0 {
		selCount = HighlightText.Render(selCount)
	} else {
		selCount = Muted.Render(selCount)
	}
	line := fmt.Sprintf(" %s   Total: %s   Reclaimable: %s",
		selCount,
		InfoText.Render(model.HumanSize(total)),
		HighlightText.Render(model.HumanSize(reclaim)))
	if extra != "" {
		line += "   " + extra
	}
	return line
}

// Resize adjusts column widths and table dimensions.
func (a *AppTable) Resize(width, height int) {
	if width != a.lastW && width > 0 {
		a.lastW = width
		a.table.SetColumns(defaultColumns(width))
		a.table.SetWidth(width)
	}
	if height != a.lastH && height > 0 {
		a.lastH = height
		a.table.SetHeight(height)
	}
	a.table.UpdateViewport()
}

// View renders the table panel.
func (a *AppTable) View(width, height int) string {
	a.Resize(width, height)
	return PanelFocus.Render(a.table.View())
}
