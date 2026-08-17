package ui

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/afero"
	"golang.org/x/sys/unix"

	"github.com/swadhinbiswas/veet/internal/detector"
	"github.com/swadhinbiswas/veet/internal/history"
	"github.com/swadhinbiswas/veet/internal/model"
	"github.com/swadhinbiswas/veet/internal/uninstaller"
)

type state int

const (
	stateScanning state = iota
	stateReady
	stateConfirming
	stateConfirmDelete
	stateUninstalling
	stateHistory
	stateSettings
	stateHelp
)

// CategoryFilter maps to the 6 top stat cards.
type CategoryFilter int

const (
	CatAll CategoryFilter = iota
	CatFlatpak
	CatSnap
	CatGlobalTools
	CatCache
	CatReclaimable
)

// SortMode controls list ordering.
type SortMode int

const (
	SortSizeDesc SortMode = iota
	SortNameAsc
	SortSourceAsc
	SortDateDesc
)

func (s SortMode) Label() string {
	switch s {
	case SortSizeDesc:
		return "Size ↓"
	case SortNameAsc:
		return "Name A-Z"
	case SortSourceAsc:
		return "Source"
	case SortDateDesc:
		return "Date ↓"
	default:
		return "Size ↓"
	}
}

// ————— messages —————

type startScanMsg struct{}
type activityMsg string

// ScanSourceState holds the live status of a single detector during scan.
type ScanSourceState struct {
	Status string // "pending", "starting", "done", "skipped", "error"
	Count  int
	Err    error
}

type scanProgressMsg struct {
	ev   detector.ProgressEvent
	pump *scanPump
}

type scanPump struct {
	ch   chan detector.ProgressEvent
	done chan scanDoneMsg
}

type scanDoneMsg struct {
	apps    []model.AppInfo
	skipped []string
	errs    []error
}
type stagedBatchMsg struct {
	apps          []*model.AppInfo
	errs          []string
	confirmDelete bool
}
type uninstallStepMsg struct {
	step uninstaller.Step
	pump *uninstallPump
}
type uninstallDoneMsg struct {
	results []uninstallResult
	err     error
}
type historyMsg struct {
	entries []history.Entry
	err     error
}

type uninstallResult struct {
	app     *model.AppInfo
	freedKB int64
	files   int
	err     error
}

// ————— root model —————

// Model is the root Bubble Tea model.
type Model struct {
	cfg   *model.Config
	un    *uninstaller.Uninstaller
	hlog  *history.Log
	state state

	width, height int
	spinner       spinner.Model
	scanBar       progress.Model
	scanSources   map[string]ScanSourceState
	scanCompleted int
	scanTotal     int
	scanStart     time.Time
	scanPump      *scanPump
	activity      []string // scan/refresh notes, newest last

	allApps  []model.AppInfo
	pointers []*model.AppInfo
	table    *AppTable
	search   textinput.Model
	filter   []string // ["All Sources", "apt", ...]
	filterAt int
	stats    Stats

	details     *viewport.Model
	confirm     *viewport.Model
	pending     []*model.AppInfo // apps on the confirm screen
	pendingErrs []string
	progress    *ProgressPanel
	hview       *viewport.Model
	helpShown   bool

	catFilter       CategoryFilter
	sortMode        SortMode
	showStagedPaths bool

	runStart time.Time
}

// New builds the root model bound to the real system.
func New(cfg *model.Config) *Model {
	hlog := history.New(afero.NewOsFs(), history.DefaultPath(cfg.DataDir))
	un := uninstaller.New(cfg.Home, hlog, cfg.ProtectedSet())
	return newModel(cfg, un, hlog)
}

func newModel(cfg *model.Config, un *uninstaller.Uninstaller, hlog *history.Log) *Model {
	ApplyTheme(cfg.Theme)
	details := viewport.New(60, 12)
	confirm := viewport.New(80, 20)
	hview := viewport.New(80, 20)
	details.MouseWheelEnabled = true
	confirm.MouseWheelEnabled = true
	hview.MouseWheelEnabled = true
	scanBar := progress.New(progress.WithSolidFill(Primary))
	scanBar.Width = 50
	scanBar.ShowPercentage = true
	m := &Model{
		cfg:         cfg,
		un:          un,
		hlog:        hlog,
		state:       stateScanning,
		spinner:     spinner.New(),
		scanBar:     scanBar,
		scanSources: make(map[string]ScanSourceState),
		scanTotal:   17,
		scanStart:   time.Now(),
		table:       NewAppTable(),
		filter:      []string{"All Sources"},
		details:     &details,
		confirm:     &confirm,
		progress:    NewProgressPanel(),
		hview:       &hview,
	}
	m.spinner.Style = lipgloss.NewStyle().Foreground(primary)
	m.search = textinput.New()
	m.search.Placeholder = "type to filter (@flatpak, >100M, name)..."
	m.search.Width = 40
	m.search.Prompt = "Search: "
	m.search.PromptStyle = lipgloss.NewStyle().Bold(true).Foreground(primary)
	m.search.TextStyle = lipgloss.NewStyle().Foreground(neutral)
	m.search.PlaceholderStyle = lipgloss.NewStyle().Foreground(mutedColor)
	m.activity = []string{"starting concurrent detector scan..."}
	return m
}

// Init kicks off the first streaming scan.
func (m *Model) Init() tea.Cmd {
	pump, cmd := startScan(m.cfg)
	m.scanPump = pump
	return tea.Batch(m.spinner.Tick, cmd)
}

func startScan(cfg *model.Config) (*scanPump, tea.Cmd) {
	p := &scanPump{
		ch:   make(chan detector.ProgressEvent, 64),
		done: make(chan scanDoneMsg, 1),
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()
		apps, skipped, errs := detector.ScanStreaming(ctx, detector.RealExec{}, cfg.Home, func(ev detector.ProgressEvent) {
			p.ch <- ev
		})
		close(p.ch)
		p.done <- scanDoneMsg{apps: apps, skipped: skipped, errs: errs}
	}()
	return p, p.next()
}

func (p *scanPump) next() tea.Cmd {
	return func() tea.Msg {
		select {
		case ev, ok := <-p.ch:
			if ok {
				return scanProgressMsg{ev: ev, pump: p}
			}
		default:
		}
		done := <-p.done
		return done
	}
}

// ————— Update —————

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		return m, nil

	case tea.KeyMsg:
		return m, m.handleKey(msg)

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case startScanMsg:
		m.state = stateScanning
		m.scanSources = make(map[string]ScanSourceState)
		m.scanCompleted = 0
		m.scanTotal = 17
		m.scanStart = time.Now()
		m.activity = append(m.activity, "refreshing all sources...")
		pump, cmd := startScan(m.cfg)
		m.scanPump = pump
		return m, cmd

	case scanProgressMsg:
		if m.scanSources == nil {
			m.scanSources = make(map[string]ScanSourceState)
		}
		m.scanSources[msg.ev.Source] = ScanSourceState{
			Status: msg.ev.Status,
			Count:  msg.ev.Count,
			Err:    msg.ev.Err,
		}
		if msg.ev.Total > 0 {
			m.scanTotal = msg.ev.Total
		}
		if msg.ev.Status == "done" || msg.ev.Status == "skipped" || msg.ev.Status == "error" {
			m.scanCompleted++
		}
		if msg.ev.Status == "done" && msg.ev.Count > 0 {
			m.activity = append(m.activity, fmt.Sprintf("%s: found %d package(s)", msg.ev.Source, msg.ev.Count))
		}
		return m, msg.pump.next()

	case scanDoneMsg:
		m.allApps = msg.apps
		m.stats = ComputeStats(msg.apps)
		for _, s := range msg.skipped {
			m.activity = append(m.activity, "skipped: "+s)
		}
		for _, e := range msg.errs {
			m.activity = append(m.activity, "error: "+e.Error())
		}
		m.activity = append(m.activity, fmt.Sprintf("found %d apps across %d sources",
			len(msg.apps), sourceCount(msg.apps)))
		m.pointers = nil
		for i := range m.allApps {
			m.pointers = append(m.pointers, &m.allApps[i])
		}
		m.rebuildFilter()
		m.applyFilter()
		m.state = stateReady
		return m, nil

	case stagedBatchMsg:
		m.pending = msg.apps
		m.pendingErrs = msg.errs
		m.table.SetApps(m.pointers)
		if msg.confirmDelete {
			m.state = stateConfirmDelete
			m.renderDeleteConfirm()
		} else {
			m.state = stateConfirming
			m.renderConfirm()
		}
		return m, nil

	case uninstallStepMsg:
		if m.progress.Running() {
			s := msg.step
			if s.Err != nil {
				m.progress.Push("✗ " + s.Text + " — " + s.Err.Error())
			} else {
				m.progress.Push("✓ " + s.Text)
			}
			m.progress.SetPercent(msg.pump.overallPercent(s))
		}
		return m, msg.pump.next()

	case uninstallDoneMsg:
		m.progress.Finish()
		for _, r := range msg.results {
			status := SuccessText.Render("ok")
			if r.err != nil {
				status = DangerText.Render("failed: " + r.err.Error())
			}
			m.activity = append(m.activity,
				fmt.Sprintf("uninstalled %s (%s) freed %s files=%d %s",
					r.app.Name, r.app.Source, model.HumanSize(r.freedKB), r.files, status))
		}
		if msg.err != nil {
			m.activity = append(m.activity, "batch error: "+msg.err.Error())
		}
		m.pending = nil
		m.state = stateReady
		return m, nil

	case historyMsg:
		m.hview.SetContent(renderHistory(msg.entries, msg.err))
		m.state = stateHistory
		return m, nil

	case activityMsg:
		m.activity = append(m.activity, string(msg))
		return m, nil
	}
	return m, nil
}

func (m *Model) handleKey(msg tea.KeyMsg) tea.Cmd {
	switch m.state {
	case stateScanning:
		if msg.String() == "ctrl+c" || msg.String() == "q" {
			return tea.Quit
		}
		return nil
	case stateReady:
		return m.handleReadyKey(msg)
	case stateConfirming:
		return m.handleConfirmKey(msg)
	case stateConfirmDelete:
		return m.handleDeleteConfirmKey(msg)
	case stateUninstalling:
		return nil // input blocked during an active uninstall
	case stateHistory, stateSettings, stateHelp:
		return m.handleOverlayKey(msg)
	}
	return nil
}

func (m *Model) handleReadyKey(msg tea.KeyMsg) tea.Cmd {
	// While the search input is focused, ALL keys feed the input except
	// esc (blur + clear) so global bindings don't hijack typed letters.
	if m.search.Focused() {
		if msg.String() == "esc" {
			m.search.Blur()
			m.search.SetValue("")
			m.applyFilter()
			return nil
		}
		if msg.String() == "enter" {
			m.search.Blur()
			return nil
		}
		var cmd tea.Cmd
		m.search, cmd = m.search.Update(msg)
		m.applyFilter()
		return cmd
	}

	switch msg.String() {
	case "ctrl+c", "q":
		return tea.Quit
	case "h", "?":
		m.state = stateHelp
		return nil
	case "r":
		return func() tea.Msg { return startScanMsg{} }
	case "l":
		return func() tea.Msg {
			entries, err := m.hlog.Read()
			return historyMsg{entries: entries, err: err}
		}
	case "s":
		m.state = stateSettings
		return nil
	case "1":
		m.setCategory(CatAll)
		return nil
	case "2":
		m.setCategory(CatFlatpak)
		return nil
	case "3":
		m.setCategory(CatSnap)
		return nil
	case "4":
		m.setCategory(CatGlobalTools)
		return nil
	case "5":
		m.setCategory(CatCache)
		return nil
	case "6":
		m.setCategory(CatReclaimable)
		return nil
	case "[", "left":
		m.catFilter = (m.catFilter + 5) % 6
		m.setCategory(m.catFilter)
		return nil
	case "]", "right":
		m.catFilter = (m.catFilter + 1) % 6
		m.setCategory(m.catFilter)
		return nil
	case "o", "O":
		m.sortMode = (m.sortMode + 1) % 4
		m.applyFilter()
		return nil
	case "p", "P":
		m.showStagedPaths = !m.showStagedPaths
		if app := m.table.Current(); app != nil && !app.Removable.Staged() {
			_ = m.un.StageRemoval(app)
		}
		m.refreshDetails()
		return nil
	case "e", "E":
		return m.exportReportCmd()
	case "/":
		m.search.Focus()
		m.search.CursorEnd()
		return nil
	case "tab":
		m.cycleFilter(1)
		return nil
	case "shift+tab":
		m.cycleFilter(-1)
		return nil
	case "n":
		m.table.Model().MoveDown(10)
		m.refreshDetails()
		return nil
	case "enter":
		sel := m.table.SelectedApps()
		if len(sel) > 0 {
			return m.stageApps(sel)
		}
		if app := m.table.Current(); app != nil {
			if app.Protected {
				m.activity = append(m.activity, "protected: "+app.Name+" is a core system component")
				return nil
			}
			return m.stageApps([]*model.AppInfo{app})
		}
		return nil
	case " ", "space":
		if app := m.table.Toggle(); app != nil && app.Protected {
			m.activity = append(m.activity, "protected: "+app.Name+" refused selection")
		}
		return nil
	case "a":
		if refused := m.table.SelectAll(); refused > 0 {
			m.activity = append(m.activity, fmt.Sprintf("skipped %d protected system components", refused))
		}
		return nil
	case "d", "D":
		// Dedicated delete key: stage the selected apps (or the row under
		// the cursor) and open the Yes/No confirmation popup.
		apps := m.table.SelectedApps()
		if len(apps) == 0 {
			if app := m.table.Current(); app != nil {
				if app.Protected {
					m.activity = append(m.activity, "protected: "+app.Name+" is a core system component")
					return nil
				}
				apps = []*model.AppInfo{app}
			}
		}
		if len(apps) == 0 {
			m.activity = append(m.activity, "nothing selected — move to an app first")
			return nil
		}
		return m.stageForDelete(apps)
	case "c":
		var cacheApps []*model.AppInfo
		for _, p := range m.pointers {
			if p.Source == "cache" {
				cacheApps = append(cacheApps, p)
			}
		}
		if len(cacheApps) == 0 {
			m.activity = append(m.activity, "nothing cached — nothing to clean")
			return nil
		}
		return m.stageApps(cacheApps)
	}

	var cmd tea.Cmd
	m.table.Update(msg)
	m.refreshDetails()
	if cmd == nil && (msg.String() == "pgup" || msg.String() == "pgdown") {
		_, cmd = m.details.Update(msg)
	}
	return cmd
}

func (m *Model) handleConfirmKey(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "esc":
		m.pending = nil
		m.pendingErrs = nil
		m.state = stateReady
		return nil
	case "enter":
		apps := m.pending
		if len(apps) == 0 {
			m.state = stateReady
			return nil
		}
		m.pending = nil
		m.pendingErrs = nil
		m.state = stateUninstalling
		m.progress.Start(fmt.Sprintf("Uninstalling %d app(s)...", len(apps)))
		m.runStart = time.Now()
		pump := startUninstall(m.un, apps)
		return pump.next()
	case "j", "k", "up", "down", "pgup", "pgdown", " ", "ctrl+d", "ctrl+u", "g", "G", "home", "end":
		vp, cmd := m.confirm.Update(msg)
		m.confirm = &vp
		return cmd
	}
	return nil
}

// handleDeleteConfirmKey answers the "really delete?" popup:
//   - y / Y / Enter  -> run the uninstall now
//   - n / N / Esc / q -> back to the selection screen untouched
func (m *Model) handleDeleteConfirmKey(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "y", "Y", "enter":
		apps := m.pending
		if len(apps) == 0 {
			m.state = stateReady
			return nil
		}
		m.pending = nil
		m.pendingErrs = nil
		m.state = stateUninstalling
		m.progress.Start(fmt.Sprintf("Uninstalling %d app(s)...", len(apps)))
		m.runStart = time.Now()
		pump := startUninstall(m.un, apps)
		return pump.next()
	case "n", "N", "esc", "q":
		// No: return to the previous selection screen, nothing deleted.
		m.pending = nil
		m.pendingErrs = nil
		m.state = stateReady
		m.activity = append(m.activity, "delete cancelled — nothing was removed")
		return nil
	case "j", "k", "up", "down", "pgup", "pgdown", " ", "ctrl+d", "ctrl+u", "g", "G", "home", "end":
		vp, cmd := m.confirm.Update(msg)
		m.confirm = &vp
		return cmd
	}
	return nil
}

// renderDeleteConfirm builds the Yes/No popup content for the staged apps.
func (m *Model) renderDeleteConfirm() {
	var b strings.Builder
	b.WriteString(DangerText.Render("⚠ REALLY DELETE?"))
	b.WriteString("\n\n")
	var totalKB int64
	for i, app := range m.pending {
		rem := app.Removable
		b.WriteString("  " + SuccessText.Render("✓ ") + Title.Render(app.Name) + "  " + SourceBadge(app.Source))
		if i < len(m.pendingErrs) && m.pendingErrs[i] != "" {
			b.WriteString("  " + DangerText.Render(m.pendingErrs[i]))
			b.WriteString("\n")
			continue
		}
		b.WriteString("\n")
		b.WriteString(fmt.Sprintf("     %s across %d path(s)",
			HighlightText.Render(model.HumanSize(rem.TotalKB())), rem.FileCount()))
		if len(rem.Elevated) > 0 {
			b.WriteString(" " + DangerText.Render(fmt.Sprintf("(%d needs sudo)", len(rem.Elevated))))
		}
		b.WriteString("\n")
		for _, p := range rem.Paths {
			line := "       " + p
			if isElevated(p, rem) {
				line += " " + DangerText.Render("🔒 sudo")
			}
			b.WriteString(line + "\n")
		}
		totalKB += rem.TotalKB()
		b.WriteString("\n")
	}
	b.WriteString(HighlightText.Render(fmt.Sprintf("Total: %s reclaimable — %d app(s)",
		model.HumanSize(totalKB), len(m.pending))))
	b.WriteString("\n\n")
	b.WriteString(lipgloss.NewStyle().Bold(true).Foreground(success).Render("[Y] Yes, delete") +
		"    " + lipgloss.NewStyle().Bold(true).Foreground(neutral).Render("[N] No, go back"))
	b.WriteString("\n")
	m.confirm.SetContent(b.String())
}

// renderDeleteConfirmOverlay draws the popup over the main view.
func (m *Model) renderDeleteConfirmOverlay() string {
	w := m.width - 12
	h := m.height - 14
	if w < 56 {
		w = 56
	}
	if h < 10 {
		h = 10
	}
	m.confirm.Width = w - 6
	m.confirm.Height = h - 4
	box := lipgloss.NewStyle().
		Border(lipgloss.DoubleBorder()).
		BorderForeground(danger).
		Padding(1, 2).
		Width(w).
		Render(m.confirm.View())
	return lipgloss.NewStyle().Width(m.width).Align(lipgloss.Center).Render(box)
}

func (m *Model) handleOverlayKey(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "ctrl+c", "q":
		return tea.Quit
	case "esc", "h", "?", "enter":
		m.state = stateReady
		return nil
	case "c", "C":
		if m.state == stateHistory {
			_ = m.hlog.Clear()
			m.hview.SetContent(renderHistory(nil, nil))
			m.activity = append(m.activity, "audit history cleared")
			return nil
		}
	case "l":
		return func() tea.Msg {
			entries, err := m.hlog.Read()
			return historyMsg{entries: entries, err: err}
		}
	}
	if m.state == stateHistory {
		vp, cmd := m.hview.Update(msg)
		m.hview = &vp
		return cmd
	}
	return nil
}

// ————— staging & uninstall plumbing —————

// stageApps computes the deep-clean preview for apps before confirmation.
func (m *Model) stageApps(apps []*model.AppInfo) tea.Cmd {
	return m.stage(apps, false)
}

// stageForDelete stages apps and opens the Yes/No delete popup instead of
// the full preview screen.
func (m *Model) stageForDelete(apps []*model.AppInfo) tea.Cmd {
	return m.stage(apps, true)
}

func (m *Model) stage(apps []*model.AppInfo, confirmDelete bool) tea.Cmd {
	m.activity = append(m.activity, fmt.Sprintf("staging deep-clean preview for %d app(s)...", len(apps)))
	return func() tea.Msg {
		var errs []string
		for _, app := range apps {
			if err := m.un.StageRemoval(app); err != nil {
				if err == uninstaller.ErrProtected {
					errs = append(errs, app.Name+": protected system component")
				} else {
					errs = append(errs, app.Name+": "+err.Error())
				}
			}
		}
		return stagedBatchMsg{apps: apps, errs: errs, confirmDelete: confirmDelete}
	}
}

// uninstallPump streams step messages from the uninstall goroutine.
type uninstallPump struct {
	un      *uninstaller.Uninstaller
	apps    []*model.AppInfo
	ch      chan uninstaller.Step
	done    chan struct{}
	results []uninstallResult
	idx     int // app index of the latest step
}

func startUninstall(un *uninstaller.Uninstaller, apps []*model.AppInfo) *uninstallPump {
	p := &uninstallPump{
		un:   un,
		apps: apps,
		ch:   make(chan uninstaller.Step, 64),
		done: make(chan struct{}),
	}
	go func() {
		ctx := context.Background()
		var firstErr error
		for _, app := range apps {
			app.Status = "Removing"
			freed, files, err := un.Remove(ctx, app, func(s uninstaller.Step) {
				p.ch <- s
			})
			app.Status = "Removed"
			p.results = append(p.results, uninstallResult{app: app, freedKB: freed, files: files, err: err})
			if err != nil && firstErr == nil {
				firstErr = err
			}
		}
		close(p.ch)
		close(p.done)
	}()
	return p
}

// overallPercent maps a per-app step to the batch-wide progress.
func (p *uninstallPump) overallPercent(s uninstaller.Step) float64 {
	for i := p.idx; i < len(p.apps); i++ {
		if p.apps[i].Name == s.App {
			p.idx = i
			break
		}
	}
	if len(p.apps) == 0 {
		return 1
	}
	return (float64(p.idx) + s.Percent) / float64(len(p.apps))
}

// next returns a cmd that emits the next step message, or the final result.
func (p *uninstallPump) next() tea.Cmd {
	return func() tea.Msg {
		select {
		case step, ok := <-p.ch:
			if ok {
				return uninstallStepMsg{step: step, pump: p}
			}
		default:
		}
		<-p.done
		firstErr := error(nil)
		for _, r := range p.results {
			if r.err != nil {
				firstErr = r.err
				break
			}
		}
		return uninstallDoneMsg{results: p.results, err: firstErr}
	}
}

// ————— filtering —————

func sourceCount(apps []model.AppInfo) int {
	seen := map[string]bool{}
	for _, a := range apps {
		seen[a.Source] = true
	}
	return len(seen)
}

func (m *Model) rebuildFilter() {
	seen := map[string]bool{"All Sources": true}
	for _, a := range m.allApps {
		seen[a.Source] = true
	}
	m.filter = []string{"All Sources"}
	for _, s := range []string{"apt", "dnf", "pacman", "aur", "zypper", "flatpak", "snap", "appimage", "brew", "nix", "orphan", "npm", "pipx", "cargo", "gem", "go", "cache", "system"} {
		if seen[s] {
			m.filter = append(m.filter, s)
		}
	}
	if m.filterAt >= len(m.filter) {
		m.filterAt = 0
	}
}

func (m *Model) cycleFilter(delta int) {
	m.filterAt = (m.filterAt + delta + len(m.filter)) % len(m.filter)
	m.applyFilter()
}

// fuzzyMatch: substring or ordered-subsequence match, case-insensitive.
func fuzzyMatch(query, name string) bool {
	q := strings.ToLower(query)
	n := strings.ToLower(name)
	if strings.Contains(n, q) {
		return true
	}
	i := 0
	for j := 0; j < len(n) && i < len(q); j++ {
		if n[j] == q[i] {
			i++
		}
	}
	return i == len(q)
}

type parsedQuery struct {
	sourceFilter string
	minSizeKB    int64
	maxSizeKB    int64
	protected    *bool
	text         string
}

func parseSearchQuery(raw string) parsedQuery {
	fields := strings.Fields(raw)
	var textParts []string
	var pq parsedQuery

	for _, f := range fields {
		lower := strings.ToLower(f)
		if strings.HasPrefix(lower, "@") {
			pq.sourceFilter = strings.TrimPrefix(lower, "@")
		} else if strings.HasPrefix(lower, "source:") {
			pq.sourceFilter = strings.TrimPrefix(lower, "source:")
		} else if strings.HasPrefix(lower, "protected:") {
			val := strings.TrimPrefix(lower, "protected:")
			b := val == "true" || val == "yes" || val == "1"
			pq.protected = &b
		} else if strings.HasPrefix(lower, ">") || strings.HasPrefix(lower, "size:>") {
			val := strings.TrimPrefix(strings.TrimPrefix(lower, "size:"), ">")
			pq.minSizeKB = detectorParseHumanSize(val)
		} else if strings.HasPrefix(lower, "<") || strings.HasPrefix(lower, "size:<") {
			val := strings.TrimPrefix(strings.TrimPrefix(lower, "size:"), "<")
			pq.maxSizeKB = detectorParseHumanSize(val)
		} else {
			textParts = append(textParts, f)
		}
	}
	pq.text = strings.Join(textParts, " ")
	return pq
}

func detectorParseHumanSize(s string) int64 {
	s = strings.TrimSpace(strings.ToUpper(s))
	if s == "" {
		return 0
	}
	var num float64
	var unit string
	n, _ := fmt.Sscanf(s, "%f%s", &num, &unit)
	if n < 1 || num <= 0 {
		return 0
	}
	switch {
	case strings.HasPrefix(unit, "T"):
		return int64(num * 1024 * 1024 * 1024)
	case strings.HasPrefix(unit, "G"):
		return int64(num * 1024 * 1024)
	case strings.HasPrefix(unit, "M"):
		return int64(num * 1024)
	case strings.HasPrefix(unit, "K"):
		return int64(num)
	case strings.HasPrefix(unit, "B"):
		return int64(num) / 1024
	default:
		return int64(num * 1024)
	}
}

func (m *Model) setCategory(cat CategoryFilter) {
	m.catFilter = cat
	m.filterAt = 0
	m.applyFilter()
}

func (m *Model) matchesCategory(a *model.AppInfo) bool {
	switch m.catFilter {
	case CatAll:
		return true
	case CatFlatpak:
		return a.Source == "flatpak"
	case CatSnap:
		return a.Source == "snap"
	case CatGlobalTools:
		return a.Source == "npm" || a.Source == "pipx" || a.Source == "cargo" || a.Source == "gem" || a.Source == "go" || a.Source == "brew" || a.Source == "nix"
	case CatCache:
		return a.Source == "cache"
	case CatReclaimable:
		return a.Source == "cache" || a.Removable.Staged() || a.InstallSizeKB > 0
	}
	return true
}

func (m *Model) applyFilter() {
	pq := parseSearchQuery(m.search.Value())
	src := m.filter[m.filterAt]
	if pq.sourceFilter != "" {
		src = pq.sourceFilter
	}

	var ptrs []*model.AppInfo
	for i := range m.allApps {
		a := &m.allApps[i]
		if !m.matchesCategory(a) {
			continue
		}
		if src != "All Sources" && !strings.EqualFold(a.Source, src) {
			continue
		}
		if pq.protected != nil && a.Protected != *pq.protected {
			continue
		}
		if pq.minSizeKB > 0 && a.InstallSizeKB < pq.minSizeKB {
			continue
		}
		if pq.maxSizeKB > 0 && a.InstallSizeKB > pq.maxSizeKB {
			continue
		}
		if pq.text != "" && !fuzzyMatch(pq.text, a.Name) {
			continue
		}
		ptrs = append(ptrs, a)
	}

	// Sort filtered apps
	sort.SliceStable(ptrs, func(i, j int) bool {
		switch m.sortMode {
		case SortSizeDesc:
			if ptrs[i].InstallSizeKB != ptrs[j].InstallSizeKB {
				return ptrs[i].InstallSizeKB > ptrs[j].InstallSizeKB
			}
			return strings.ToLower(ptrs[i].Name) < strings.ToLower(ptrs[j].Name)
		case SortNameAsc:
			return strings.ToLower(ptrs[i].Name) < strings.ToLower(ptrs[j].Name)
		case SortSourceAsc:
			if ptrs[i].Source != ptrs[j].Source {
				return ptrs[i].Source < ptrs[j].Source
			}
			return strings.ToLower(ptrs[i].Name) < strings.ToLower(ptrs[j].Name)
		case SortDateDesc:
			if !ptrs[i].InstalledOn.Equal(ptrs[j].InstalledOn) {
				return ptrs[i].InstalledOn.After(ptrs[j].InstalledOn)
			}
			return strings.ToLower(ptrs[i].Name) < strings.ToLower(ptrs[j].Name)
		default:
			return ptrs[i].InstallSizeKB > ptrs[j].InstallSizeKB
		}
	})

	m.table.SetApps(ptrs)
	m.refreshDetails()
}

func (m *Model) refreshDetails() {
	m.details.SetContent(Details(m.table.Current(), m.showStagedPaths))
}

func (m *Model) exportReportCmd() tea.Cmd {
	return func() tea.Msg {
		reportPath := filepath.Join(m.cfg.DataDir, "apps-report.json")
		_ = os.MkdirAll(m.cfg.DataDir, 0o755)
		data, err := json.MarshalIndent(m.allApps, "", "  ")
		if err != nil {
			return activityMsg("failed to generate export: " + err.Error())
		}
		if err := os.WriteFile(reportPath, data, 0o644); err != nil {
			return activityMsg("export write error: " + err.Error())
		}
		return activityMsg("✓ exported report to " + reportPath)
	}
}

// renderConfirm builds the confirmation panel content (every path listed).
func (m *Model) renderConfirm() {
	var b strings.Builder
	b.WriteString(DangerText.Render("⚠ CONFIRM UNINSTALL — CANNOT BE UNDONE"))
	b.WriteString("\n\n")

	var totalKB, totalFiles int64
	hasElevated := false
	for i, app := range m.pending {
		rem := app.Removable
		b.WriteString(SuccessText.Render("✓ ") + Title.Render(app.Name) + "  " + SourceBadge(app.Source))
		b.WriteString("\n")
		if i < len(m.pendingErrs) && m.pendingErrs[i] != "" {
			b.WriteString("    " + DangerText.Render(m.pendingErrs[i]) + "\n\n")
			continue
		}
		b.WriteString(fmt.Sprintf("    package + %s across %d path(s)",
			HighlightText.Render(model.HumanSize(rem.TotalKB())), rem.FileCount()))
		if len(rem.Elevated) > 0 {
			b.WriteString(" " + DangerText.Render(fmt.Sprintf("(%d requires sudo)", len(rem.Elevated))))
			hasElevated = true
		}
		b.WriteString("\n")
		for _, line := range strings.Split(removeRow("config", rem.ConfigKB)+
			removeRow("cache", rem.CacheKB)+
			removeRow("logs", rem.LogKB)+
			removeRow("local data", rem.LocalDataKB)+
			removeRow("residuals", rem.ResidualKB), "\n") {
			if strings.TrimSpace(line) != "" {
				b.WriteString("    " + strings.TrimSpace(line) + "\n")
			}
		}
		if rem.FileCount() > 0 {
			b.WriteString("    " + Muted.Render("paths:") + "\n")
			for _, p := range rem.Paths {
				line := "      " + p
				if isElevated(p, rem) {
					line += " " + DangerText.Render("🔒 sudo")
				}
				b.WriteString(line + "\n")
			}
		}
		totalKB += rem.TotalKB()
		totalFiles += int64(rem.FileCount())
		b.WriteString("\n")
	}
	b.WriteString(HighlightText.Render(fmt.Sprintf("Total: %s reclaimable across %d path(s) — %d app(s)",
		model.HumanSize(totalKB), totalFiles, len(m.pending))))
	b.WriteString("\n")
	if hasElevated {
		b.WriteString(DangerText.Render("🔒 Some paths require sudo — you will be prompted for your password once.\n"))
	}
	b.WriteString("\n" + DangerText.Render("⚠ This cannot be undone. Press Enter to confirm, Esc to cancel."))
	m.confirm.SetContent(b.String())
}

// ————— View —————

func (m *Model) View() string {
	if m.width == 0 {
		return "loading..."
	}
	switch m.state {
	case stateScanning:
		return m.renderScan()
	case stateReady:
		return m.renderMain()
	case stateConfirming:
		return m.renderMain() + "\n" + m.renderConfirmOverlay()
	case stateConfirmDelete:
		return m.renderMain() + "\n" + m.renderDeleteConfirmOverlay()
	case stateUninstalling:
		return m.renderMain()
	case stateHistory:
		return m.renderMain() + "\n" + m.renderOverlay("HISTORY LOG", m.hview.View(), "Esc to close · [c] clear history")
	case stateSettings:
		return m.renderMain() + "\n" + m.renderOverlay("SETTINGS", m.renderSettings(), "Esc to close")
	case stateHelp:
		return m.renderMain() + "\n" + m.renderOverlay("KEYBINDINGS", helpKeys(), "Esc to close")
	}
	return ""
}

func (m *Model) renderScan() string {
	w := m.width
	if w < 60 {
		w = 60
	}
	m.scanBar.Width = min(60, w-10)

	var b strings.Builder
	b.WriteString(m.renderHeader())
	b.WriteString("\n\n")

	// 1. Live Banner & Progress Bar
	elapsed := int(time.Since(m.scanStart).Seconds())
	pct := float64(0)
	if m.scanTotal > 0 {
		pct = float64(m.scanCompleted) / float64(m.scanTotal)
	}
	if pct > 1 {
		pct = 1
	}
	barView := m.scanBar.ViewAs(pct)

	titleLine := fmt.Sprintf("%s %s %s",
		m.spinner.View(),
		HighlightText.Render("SCANNING SYSTEM SOURCES & DETECTORS..."),
		InfoText.Render(fmt.Sprintf("(%d/%d completed · %02d:%02ds)", m.scanCompleted, m.scanTotal, elapsed/60, elapsed%60)))
	b.WriteString(" " + titleLine + "\n")
	b.WriteString(" " + barView + "\n\n")

	// 2. Matrix Grid of all detector tiles
	sources := []string{
		"apt", "dnf", "pacman", "aur", "zypper", "flatpak",
		"snap", "appimage", "brew", "nix", "orphan", "npm",
		"pipx", "cargo", "gem", "go", "cache", "system",
	}

	cols := 3
	if w >= 110 {
		cols = 4
	} else if w < 75 {
		cols = 2
	}
	colW := (w - 6) / cols
	if colW < 20 {
		colW = 20
	}

	var rowCells []string
	for i, s := range sources {
		meta := model.Meta(s)
		st, ok := m.scanSources[s]
		statusLabel := Muted.Render("○ pending")
		borderClr := panelBorder
		if ok {
			switch st.Status {
			case "starting", "scanning":
				statusLabel = HighlightText.Render("● scanning...")
				borderClr = highlight
			case "done":
				if st.Count > 0 {
					statusLabel = SuccessText.Render(fmt.Sprintf("✓ %d apps", st.Count))
					borderClr = success
				} else {
					statusLabel = Muted.Render("✓ 0 found")
					borderClr = panelBorder
				}
			case "skipped":
				statusLabel = Muted.Render("– skipped")
			case "error":
				statusLabel = DangerText.Render("✗ error")
				borderClr = danger
			}
		}

		iconBadge := lipgloss.NewStyle().Foreground(lipgloss.Color(meta.Color)).Bold(true).Render(meta.Icon + " " + meta.Label)
		tileContent := iconBadge + " " + statusLabel
		tile := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(borderClr).
			Width(colW - 2).
			Padding(0, 1).
			Render(Truncate(tileContent, colW-4))
		rowCells = append(rowCells, tile)

		if len(rowCells) == cols || i == len(sources)-1 {
			b.WriteString(" " + lipgloss.JoinHorizontal(lipgloss.Top, rowCells...) + "\n")
			rowCells = nil
		}
	}

	b.WriteString("\n")

	// 3. Live activity stream
	b.WriteString(" " + HighlightText.Render("LIVE ACTIVITY STREAM:") + "\n")
	availLines := m.height - 22
	if availLines < 3 {
		availLines = 3
	}
	for _, line := range lastN(m.activity, availLines) {
		b.WriteString("   " + Muted.Render("• "+Truncate(line, w-8)) + "\n")
	}

	b.WriteString("\n " + Muted.Render("[q/Ctrl+C] Quit  [r] Rescan"))
	return b.String()
}

func diskUsage(path string) string {
	var stat unix.Statfs_t
	if err := unix.Statfs(path, &stat); err != nil {
		return ""
	}
	total := int64(stat.Blocks) * int64(stat.Bsize) / 1024
	free := int64(stat.Bavail) * int64(stat.Bsize) / 1024
	used := total - free
	if total <= 0 {
		return ""
	}
	pct := int(float64(used) * 100 / float64(total))
	return fmt.Sprintf("💾 %s / %s (%d%%)", model.HumanSize(used), model.HumanSize(total), pct)
}

func (m *Model) renderHeader() string {
	logo := Title.Render("VEET")
	sub := HeaderMuted.Render("Universal Linux App Uninstaller")
	disk := diskUsage(m.cfg.Home)
	if disk != "" {
		disk = HighlightText.Render(disk)
	}
	clock := InfoText.Render(time.Now().Format("15:04:05"))
	left := logo + "  " + sub
	if disk != "" {
		left += "   " + disk
	}
	w := m.width
	if w < 20 {
		w = 80
	}
	padWidth := w - lipgloss.Width(left) - lipgloss.Width(clock) - 2
	if padWidth < 1 {
		padWidth = 1
	}
	pad := strings.Repeat(" ", padWidth)
	return " " + left + pad + clock + " "
}

func (m *Model) renderMain() string {
	w, h := m.width, m.height
	var b strings.Builder
	b.WriteString(m.renderHeader())
	b.WriteString("\n")
	b.WriteString(RenderStatCards(m.stats, w, int(m.catFilter)))
	b.WriteString("\n")

	// search row
	filterLabel := "[" + m.filter[m.filterAt] + " ▾]"
	sortLabel := "Sort: [" + m.sortMode.Label() + "]"
	if m.search.Focused() {
		b.WriteString(" " + m.search.View() + "   " + HighlightText.Render(filterLabel) + "   " + InfoText.Render(sortLabel) + " " + Muted.Render("(Esc to exit)"))
	} else {
		searchVal := m.search.Value()
		if searchVal == "" {
			searchVal = "(press / to search)"
		}
		extra := "   " + Muted.Render("1-6: cards | p: paths | e: export")
		if w < 100 {
			extra = ""
		}
		b.WriteString(" " + Muted.Render("Search: ") + Sub.Render(Truncate(searchVal, 22)) +
			"   " + InfoText.Render(filterLabel) + Muted.Render(" (Tab)") +
			"   " + HighlightText.Render(sortLabel) + Muted.Render(" (o)") +
			extra)
	}
	b.WriteString("\n")

	// middle split: table | details
	// Budget: Top (6) + Middle (innerTableH + 3) + Bottom (10) + Summary (1) + Footer (1) = innerTableH + 21
	innerTableH := h - 21
	if innerTableH < 3 {
		innerTableH = 3
	}
	tableW := (w*3)/5 - 2
	detailW := w - tableW - 3
	if detailW < 36 {
		detailW = 36
		tableW = w - detailW - 3
	}
	if tableW < 24 {
		tableW = 24
	}
	m.details.Width = detailW - 4
	m.details.Height = innerTableH
	left := m.table.View(tableW, innerTableH)
	right := Panel.Width(detailW).Render(m.details.View())
	b.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, left, right))
	b.WriteString("\n")

	// bottom split: progress/activity | quick actions
	leftW := (w*3)/5 - 2
	rightW := w - leftW - 3
	var bottom string
	if m.progress.Running() {
		m.progress.bar.Width = leftW - 8
		elapsed := int(time.Since(m.runStart).Seconds())
		bottom = lipgloss.JoinHorizontal(lipgloss.Top,
			m.progress.View(), QuickActions(rightW, elapsed))
	} else {
		bottom = lipgloss.JoinHorizontal(lipgloss.Top,
			ProgressIdle(leftW, m.activity), QuickActions(rightW, 0))
	}
	b.WriteString(bottom)
	b.WriteString("\n")

	b.WriteString(m.table.Summary(""))
	b.WriteString("\n")
	b.WriteString(FooterKeybinds(w))
	return b.String()
}

func (m *Model) renderConfirmOverlay() string {
	w := m.width - 8
	h := m.height - 10
	if w < 60 {
		w = 60
	}
	if h < 12 {
		h = 12
	}
	m.confirm.Width = w - 6
	m.confirm.Height = h - 4
	box := lipgloss.NewStyle().
		Border(lipgloss.DoubleBorder()).
		BorderForeground(danger).
		Padding(1, 2).
		Width(w).
		Render(m.confirm.View())
	return lipgloss.NewStyle().Width(m.width).Align(lipgloss.Center).Render(box)
}

func (m *Model) renderOverlay(title, content, hint string) string {
	w := m.width - 8
	h := m.height - 8
	if w < 60 {
		w = 60
	}
	if h < 12 {
		h = 12
	}
	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(primary).
		Padding(1, 2).
		Width(w).
		Render(Title.Render(title) + "\n\n" +
			lipgloss.NewStyle().Width(w-6).MaxHeight(h-4).Render(content) +
			"\n\n" + Muted.Render(hint))
	return lipgloss.NewStyle().Width(m.width).Align(lipgloss.Center).Render(box)
}

func (m *Model) renderSettings() string {
	var b strings.Builder
	b.WriteString("Config file: " + Muted.Render(m.cfg.ConfigDir+"/config.yaml") + "\n")
	b.WriteString("History log: " + Muted.Render(m.hlog.Path()) + "\n")
	b.WriteString("Home: " + Muted.Render(m.cfg.Home) + "\n\n")
	b.WriteString("Protected packages (deep clean refused):\n")
	keys := make([]string, 0, len(m.cfg.ProtectedSet()))
	for k := range m.cfg.ProtectedSet() {
		keys = append(keys, k)
	}
	b.WriteString(Muted.Render("  "+Truncate(strings.Join(sortedKeys(keys), ", "), 160)) + "\n")
	return b.String()
}

func lastN(lines []string, n int) []string {
	if n <= 0 {
		return nil
	}
	if len(lines) <= n {
		return lines
	}
	return lines[len(lines)-n:]
}

func sortedKeys(keys []string) []string {
	for i := 1; i < len(keys); i++ {
		for j := i; j > 0 && keys[j] < keys[j-1]; j-- {
			keys[j], keys[j-1] = keys[j-1], keys[j]
		}
	}
	return keys
}
