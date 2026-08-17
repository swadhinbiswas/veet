package detector

import (
	"bytes"
	"context"
	"strings"
	"sync"
	"testing"

	"github.com/swadhinbiswas/veet/internal/model"
)

func TestParseHumanSize(t *testing.T) {
	cases := []struct {
		in   string
		want int64
	}{
		{"33.41 MiB", 34211},
		{"52.3 MB", 53555},
		{"1,024 KB", 1024},
		{"2 GiB", 2 * 1024 * 1024},
		{"512.0 MB", 524288},
		{"1.5 GIO", 1572864},
		{"12345678", 12056}, // bytes
		{"500 B", 1},        // small non-zero rounds to 1KB
		{"0.5 KiB", 1},      // small non-zero fractional rounds to 1KB
		{"0 B", 0},
		{"", 0},
		{"garbage", 0},
	}
	for _, c := range cases {
		if got := parseHumanSize(c.in); got != c.want {
			t.Errorf("parseHumanSize(%q) = %d, want %d", c.in, got, c.want)
		}
	}
}

func TestParseDate(t *testing.T) {
	cases := []struct {
		in   string
		want bool // whether a non-zero time was parsed
	}{
		{"Sun 16 Aug 2026 01:23:45 AM EDT", true},
		{"Sun 16 Aug 2026 01:23:45 AM +06", true},
		{"Sun 16 Aug 2026 01:23:45 PM +06", true},
		{"Sun 16 Aug 2026 01:23:45 PM", true},
		{"Sun 16 Aug 2026 15:04:05 MST", true},
		{"Sun 16 Aug 2026 15:04:05 +06", true},
		{"2026-08-16 01:23:45", true},
		{"2026-08-16 01:23", true},
		{"2026-08-16", true},
		{"2026-08-16T01:23:45Z", true},
		{"", false},
		{"-", false},
		{"unknown", false},
	}
	for _, c := range cases {
		got := parseDate(c.in)
		if !got.IsZero() != c.want {
			t.Errorf("parseDate(%q).IsZero() = %v, want non-zero: %v", c.in, got.IsZero(), c.want)
		}
	}
}

// fakeExecer emulates package managers on the system PATH.
type fakeExecer struct {
	paths map[string]string // tool -> location
	cmds  map[string]string // "cmd args..." -> output
}

func (f fakeExecer) LookPath(name string) (string, error) {
	if _, ok := f.paths[name]; ok {
		return "/usr/bin/" + name, nil
	}
	return "", errNotFound
}

func (f fakeExecer) CommandContext(ctx context.Context, name string, args ...string) Cmd {
	return &fakeCmd{output: f.cmds[name+" "+strings.Join(args, " ")]}
}

type errNotFoundT struct{}

func (errNotFoundT) Error() string { return "executable not found" }

var errNotFound = errNotFoundT{}

type fakeCmd struct {
	output string
	err    error
	stdout *bytes.Buffer
}

func (c *fakeCmd) SetStdout(b *bytes.Buffer) { c.stdout = b }
func (c *fakeCmd) SetStderr(b *bytes.Buffer) {}
func (c *fakeCmd) Run() error {
	if c.stdout != nil {
		c.stdout.WriteString(c.output)
	}
	return c.err
}

func TestPacmanUnionDedupe(t *testing.T) {
	ex := fakeExecer{
		paths: map[string]string{"pacman": "/usr/bin/pacman", "yay": "/usr/bin/yay"},
		cmds: map[string]string{
			"pacman -Qe":           "zsh 5.9-1\nalacritty 0.17.0-1\nmyaur 1.0-1\n",
			"pacman -Qm":           "myaur 1.0-1\notheraur 2.0-1\n",
			"pacman -Qi zsh":       "Installed Size   : 10.00 MiB\n",
			"pacman -Qi alacritty": "Installed Size   : 5.00 MiB\n",
			"pacman -Qi myaur":     "Installed Size   : 3.00 MiB\n",
			"pacman -Qi otheraur":  "Installed Size   : 2.00 MiB\n",
		},
	}
	d := NewPacmanDetector(ex)
	apps, err := d.Detect(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(apps) != 4 {
		t.Fatalf("expected 4 unique apps, got %d: %v", len(apps), names(apps))
	}
	byName := map[string]model.AppInfo{}
	for _, a := range apps {
		byName[a.Name] = a
	}
	if byName["myaur"].Source != "aur" {
		t.Errorf("myaur source = %s, want aur", byName["myaur"].Source)
	}
	if byName["zsh"].Source != "pacman" {
		t.Errorf("zsh source = %s, want pacman", byName["zsh"].Source)
	}
	if byName["myaur"].InstallSizeKB != 3*1024 {
		t.Errorf("myaur size = %d", byName["myaur"].InstallSizeKB)
	}
}

func TestRunAllConcurrent(t *testing.T) {
	ex := fakeExecer{
		paths: map[string]string{"dpkg-query": "/usr/bin/dpkg-query"},
		cmds: map[string]string{
			"dpkg-query -W -f=${binary:Package}\t${Version}\t${Installed-Size}\n": "vim\t9.0\t1000\ncurl\t7.8\t500\n",
		},
	}
	detectors := []Detector{NewAptDetector(ex), NewFlatpakDetector(ex), NewCacheDetector("/nonexistent")}
	results := RunAll(context.Background(), ex, detectors)
	total := 0
	for _, r := range results {
		if r.Skipped {
			continue
		}
		total += len(r.Apps)
	}
	if total != 2 {
		t.Fatalf("expected 2 apps from apt, got %d", total)
	}
}

func TestBrewDetector(t *testing.T) {
	ex := fakeExecer{
		paths: map[string]string{"brew": "/usr/bin/brew"},
		cmds: map[string]string{
			"brew list --versions": "htop 3.2.2\nripgrep 14.1.0\n",
		},
	}
	d := NewBrewDetector(ex)
	if !d.Available() {
		t.Fatal("expected brew detector to be available")
	}
	apps, err := d.Detect(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(apps) != 2 {
		t.Fatalf("expected 2 brew apps, got %d", len(apps))
	}
	if apps[0].Name != "htop" || apps[0].Version != "3.2.2" || apps[0].Source != "brew" {
		t.Errorf("unexpected first app: %+v", apps[0])
	}
}

func TestNixDetector(t *testing.T) {
	ex := fakeExecer{
		paths: map[string]string{"nix-env": "/usr/bin/nix-env"},
		cmds: map[string]string{
			"nix-env -q": "git-2.42.0\nhello-2.12.1\n",
		},
	}
	d := NewNixDetector(ex)
	if !d.Available() {
		t.Fatal("expected nix detector to be available")
	}
	apps, err := d.Detect(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(apps) != 2 {
		t.Fatalf("expected 2 nix apps, got %d", len(apps))
	}
	if apps[0].Name != "git" || apps[0].Version != "2.42.0" || apps[0].Source != "nix" {
		t.Errorf("unexpected first app: %+v", apps[0])
	}
}

func TestScanStreamingEvents(t *testing.T) {
	ex := fakeExecer{
		paths: map[string]string{"dpkg-query": "/usr/bin/dpkg-query"},
		cmds: map[string]string{
			"dpkg-query -W -f=${binary:Package}\t${Version}\t${Installed-Size}\n": "vim\t9.0\t1000\n",
		},
	}
	var mu sync.Mutex
	var events []ProgressEvent
	apps, skipped, errs := ScanStreaming(context.Background(), ex, "/tmp/fakehome", func(ev ProgressEvent) {
		mu.Lock()
		events = append(events, ev)
		mu.Unlock()
	})
	if len(apps) == 0 {
		t.Fatal("expected detected apps")
	}
	_ = skipped
	_ = errs
	mu.Lock()
	count := len(events)
	mu.Unlock()
	if count == 0 {
		t.Fatal("expected progress events")
	}
}

func names(apps []model.AppInfo) []string {
	var out []string
	for _, a := range apps {
		out = append(out, a.Name)
	}
	return out
}
