package ui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/afero"

	"github.com/swadhinbiswas/veet/internal/history"
	"github.com/swadhinbiswas/veet/internal/model"
	"github.com/swadhinbiswas/veet/internal/uninstaller"
)

func keyMsg(s string) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
}

func testModel() *Model {
	cfg := model.DefaultConfig()
	hlog := history.New(afero.NewMemMapFs(), "/tmp/history.log")
	un := uninstaller.New(cfg.Home, hlog, cfg.ProtectedSet())
	m := newModel(cfg, un, hlog)
	m.allApps = []model.AppInfo{
		{Name: "ivpn", Version: "1.0", Source: "aur", Status: "Installed"},
		{Name: "ivpn-ui", Version: "2.0", Source: "aur", Status: "Installed"},
		{Name: "neovim", Version: "0.9", Source: "pacman", Status: "Installed"},
		{Name: "zsh", Version: "5.9", Source: "pacman", Status: "Installed", Protected: true},
	}
	for i := range m.allApps {
		m.pointers = append(m.pointers, &m.allApps[i])
	}
	m.rebuildFilter()
	m.applyFilter()
	m.state = stateReady
	return m
}

func TestSearchFocusAndFilter(t *testing.T) {
	m := testModel()
	if m.table.Count() != 4 {
		t.Fatalf("expected 4 rows, got %d", m.table.Count())
	}
	if m.search.Focused() {
		t.Fatal("search should start unfocused")
	}
	m.handleKey(keyMsg("/"))
	if !m.search.Focused() {
		t.Fatal("search not focused after '/'")
	}
	m.handleKey(keyMsg("ivpn"))
	if got := m.search.Value(); got != "ivpn" {
		t.Fatalf("search value = %q, want ivpn", got)
	}
	if m.table.Count() != 2 {
		t.Fatalf("filtered rows = %d, want 2 (ivpn, ivpn-ui)", m.table.Count())
	}
	// esc blurs and clears
	m.handleKey(tea.KeyMsg{Type: tea.KeyEsc})
	if m.search.Focused() {
		t.Fatal("search should be blurred after esc")
	}
	if m.table.Count() != 4 {
		t.Fatalf("rows after clear = %d, want 4", m.table.Count())
	}
}

func TestGlobalKeysIgnoredWhileSearching(t *testing.T) {
	m := testModel()
	m.handleKey(keyMsg("/"))
	// 'a', 'q', 's' typed into search must not trigger global actions
	m.handleKey(keyMsg("aqs"))
	if m.search.Value() != "aqs" {
		t.Fatalf("value = %q, want aqs", m.search.Value())
	}
	if m.state != stateReady {
		t.Fatalf("state = %d, want ready", m.state)
	}
	if m.table.Count() != 0 {
		t.Fatalf("filter 'aqs' should match nothing, got %d rows", m.table.Count())
	}
}

func TestProtectedRefused(t *testing.T) {
	m := testModel()
	m.table.Model().SetCursor(3) // zsh (protected)
	if app := m.table.Current(); app == nil || !app.Protected {
		t.Fatal("expected protected app under cursor")
	}
	if app := m.table.Toggle(); !app.Protected {
		t.Fatal("protected toggle should return app")
	}
	if len(m.table.SelectedApps()) != 0 {
		t.Fatal("protected app must never be selected")
	}
	if refused := m.table.SelectAll(); refused != 1 {
		t.Fatalf("SelectAll should refuse 1 protected, refused=%d", refused)
	}
	if len(m.table.SelectedApps()) != 3 {
		t.Fatalf("SelectAll selected %d, want 3", len(m.table.SelectedApps()))
	}
}

func TestFuzzyMatch(t *testing.T) {
	cases := []struct {
		q, n string
		want bool
	}{
		{"nvim", "neovim", true},
		{"neo", "neovim", true},
		{"foo", "neovim", false},
		{"GO", "googlesearch", true},
		{"", "anything", true},
	}
	for _, c := range cases {
		if got := fuzzyMatch(c.q, c.n); got != c.want {
			t.Errorf("fuzzyMatch(%q,%q)=%v want %v", c.q, c.n, got, c.want)
		}
	}
}

func TestHumanSize(t *testing.T) {
	cases := []struct {
		kb   int64
		want string
	}{
		{0, "0B"}, {1, "1KB"}, {1023, "1023KB"}, {1024, "1MB"},
		{1024 * 1024, "1GB"}, {53 * 1024, "53MB"},
	}
	for _, c := range cases {
		if got := model.HumanSize(c.kb); got != c.want {
			t.Errorf("HumanSize(%d)=%s want %s", c.kb, got, c.want)
		}
	}
}

func TestSourceMetaComplete(t *testing.T) {
	for _, s := range []string{"apt", "dnf", "pacman", "aur", "zypper", "flatpak", "snap", "npm", "pipx", "cargo", "gem", "go", "cache"} {
		if model.Meta(s).Label == s || strings.Contains(model.Meta(s).Label, "•") && model.Meta(s).Label != s {
			// Meta must resolve to a real entry
		}
		if model.Meta(s).Icon == "•" {
			t.Errorf("source %s has fallback icon", s)
		}
	}
}

func TestRenderMainHeight(t *testing.T) {
	m := testModel()
	m.width, m.height = 120, 40
	m.stats = Stats{Total: 1400, GlobalTools: 12, CacheKB: 1024 * 800, Reclaimable: 1024 * 900}
	out := m.renderMain()
	if got := strings.Count(out, "\n") + 1; got > m.height {
		t.Fatalf("renderMain height %d exceeds terminal height %d", got, m.height)
	}
	// also when the progress panel is active
	m.progress.Start("test")
	m.progress.Push("step 1")
	if got := strings.Count(m.renderMain(), "\n") + 1; got > m.height {
		t.Fatalf("renderMain (progress running) height %d exceeds terminal height %d", got, m.height)
	}
	m.progress.Finish()
}

func TestTableCursorMove(t *testing.T) {
	m := testModel()
	if got := m.table.Current().Name; got != "ivpn" {
		t.Fatalf("cursor 0 = %s, want ivpn", got)
	}
	m.handleKey(keyMsg("j"))
	if got := m.table.Current().Name; got != "ivpn-ui" {
		t.Fatalf("after j cursor = %s, want ivpn-ui", got)
	}
	m.handleKey(keyMsg("k"))
	if got := m.table.Current().Name; got != "ivpn" {
		t.Fatalf("after k cursor = %s, want ivpn", got)
	}
}

func TestSpaceSelectionAndBatch(t *testing.T) {
	m := testModel()
	m.handleKey(tea.KeyMsg{Type: tea.KeySpace})
	if got := len(m.table.SelectedApps()); got != 1 {
		t.Fatalf("after space, %d selected, want 1", got)
	}
	m.handleKey(keyMsg("j"))
	m.handleKey(tea.KeyMsg{Type: tea.KeySpace})
	sel := m.table.SelectedApps()
	if len(sel) != 2 {
		t.Fatalf("after 2 spaces, %d selected, want 2", len(sel))
	}
	if sel[0].Name != "ivpn" || sel[1].Name != "ivpn-ui" {
		t.Fatalf("selected order wrong: %v %v", sel[0].Name, sel[1].Name)
	}
	// space again deselects
	m.handleKey(tea.KeyMsg{Type: tea.KeySpace})
	if len(m.table.SelectedApps()) != 1 {
		t.Fatalf("space should toggle off, got %d selected", len(m.table.SelectedApps()))
	}
}

func TestTableResize(t *testing.T) {
	tab := NewAppTable()
	tab.Resize(120, 20)
	if tab.lastW != 120 {
		t.Fatalf("expected lastW=120, got %d", tab.lastW)
	}
	tab.Resize(60, 10)
	if tab.lastW != 60 {
		t.Fatalf("expected lastW=60, got %d", tab.lastW)
	}
}

func TestStatCardsWidths(t *testing.T) {
	stats := Stats{Total: 10, Flatpak: 2, Snap: 1, GlobalTools: 3, CacheKB: 1024, Reclaimable: 2048}
	for _, w := range []int{80, 100, 120, 160} {
		out := RenderStatCards(stats, w, 0)
		if out == "" {
			t.Fatalf("stat cards output empty for width %d", w)
		}
	}
}

func TestCategoryFilterAndSorting(t *testing.T) {
	m := testModel()
	// test pressing '2' for flatpak
	m.handleKey(keyMsg("2"))
	if m.catFilter != CatFlatpak {
		t.Fatalf("catFilter = %d, want %d", m.catFilter, CatFlatpak)
	}

	// test pressing '1' for all
	m.handleKey(keyMsg("1"))
	if m.catFilter != CatAll {
		t.Fatalf("catFilter = %d, want %d", m.catFilter, CatAll)
	}

	// test cycling sort with 'o'
	m.handleKey(keyMsg("o"))
	if m.sortMode != SortNameAsc {
		t.Fatalf("sortMode = %d, want %d (SortNameAsc)", m.sortMode, SortNameAsc)
	}
}

func TestRenderMainSmallTerminal(t *testing.T) {
	m := testModel()
	m.width, m.height = 80, 24
	out := m.renderMain()
	if got := strings.Count(out, "\n") + 1; got > m.height {
		t.Fatalf("renderMain on 24-row terminal produced %d rows, want <= 24", got)
	}
}
