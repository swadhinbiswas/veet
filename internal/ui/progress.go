package ui

import (
	"strings"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/lipgloss"
)

// ProgressPanel shows the uninstall progress bar plus a scrolling activity log.
type ProgressPanel struct {
	bar     progress.Model
	spinner spinner.Model
	steps   []string
	percent float64
	label   string
	running bool
}

// NewProgressPanel builds the progress + activity panel.
func NewProgressPanel() *ProgressPanel {
	bar := progress.New(progress.WithSolidFill(Primary))
	bar.Width = 40
	bar.ShowPercentage = true
	s := spinner.New()
	s.Style = lipgloss.NewStyle().Foreground(primary)
	return &ProgressPanel{bar: bar, spinner: s}
}

// Start begins an active run.
func (p *ProgressPanel) Start(label string) {
	p.running = true
	p.label = label
	p.percent = 0
	p.steps = nil
}

// Finish ends the active run.
func (p *ProgressPanel) Finish() {
	p.running = false
}

// Running reports whether a run is active.
func (p *ProgressPanel) Running() bool { return p.running }

// SetPercent updates the progress bar.
func (p *ProgressPanel) SetPercent(v float64) { p.percent = v }

// Push appends a step line, keeping the last 6.
func (p *ProgressPanel) Push(line string) {
	p.steps = append(p.steps, line)
	if len(p.steps) > 6 {
		p.steps = p.steps[len(p.steps)-6:]
	}
}

// Steps exposes the recent activity lines.
func (p *ProgressPanel) Steps() []string { return p.steps }

// View renders the panel (or empty when idle).
func (p *ProgressPanel) View(width ...int) string {
	if !p.running {
		return ""
	}
	w := 56
	if len(width) > 0 && width[0] > 10 {
		w = width[0]
	}
	var b strings.Builder
	b.WriteString(p.spinner.View() + " " + HighlightText.Render("UNINSTALLING & DEEP CLEANING..."))
	b.WriteString("\n")
	b.WriteString(" " + Sub.Render(Truncate(p.label, w-4)))
	b.WriteString("\n\n")
	p.bar.Width = w - 6
	b.WriteString(" " + p.bar.ViewAs(p.percent))
	b.WriteString("\n\n")
	b.WriteString(" " + Muted.Render("Recent Actions:"))
	b.WriteString("\n")
	if len(p.steps) == 0 {
		b.WriteString("   " + Muted.Render("• starting process..."))
		b.WriteString("\n")
	} else {
		for _, s := range p.steps {
			b.WriteString("   " + Truncate(s, w-6))
			b.WriteString("\n")
		}
	}
	if len(width) > 0 && width[0] > 0 {
		return Panel.Width(width[0]).Render(strings.TrimRight(b.String(), "\n"))
	}
	return Panel.Render(strings.TrimRight(b.String(), "\n"))
}
