package uninstaller

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/spf13/afero"

	"github.com/swadhinbiswas/veet/internal/model"
)

// fakeCmder records commands and simulates failures on demand.
type fakeCmder struct {
	ran     []string
	fail    map[string]bool  // tool name -> always fail
	errs    map[string]error // specific command or tool -> custom error
	hasSudo bool
}

func (f *fakeCmder) LookPath(name string) (string, error) {
	if name == "sudo" && f.hasSudo {
		return "/usr/bin/sudo", nil
	}
	return "", errors.New("not found")
}

func (f *fakeCmder) Run(ctx context.Context, name string, args ...string) error {
	fullCmd := name + " " + strings.Join(args, " ")
	f.ran = append(f.ran, fullCmd)
	if f.errs != nil {
		if err, ok := f.errs[fullCmd]; ok {
			return err
		}
		if err, ok := f.errs[name]; ok {
			return err
		}
	}
	if f.fail != nil && f.fail[name] {
		return errors.New("command failed")
	}
	return nil
}

func testUninstaller(t *testing.T) (*Uninstaller, *fakeCmder) {
	t.Helper()
	fs := afero.NewMemMapFs()
	seed(t, fs, "/home/u/.config/foo/config.toml", "cfg")
	seed(t, fs, "/home/u/.cache/foo/cache.bin", "cache")
	seed(t, fs, "/home/u/.local/share/foo/data.db", "data")
	seed(t, fs, "/home/u/.foo-rc", "dot")
	seed(t, fs, "/etc/foo/foo.conf", "etc")
	cmder := &fakeCmder{hasSudo: false, fail: map[string]bool{}}
	u := &Uninstaller{FS: fs, Cmd: cmder, Home: "/home/u"}
	return u, cmder
}

func seed(t *testing.T, fs afero.Fs, path, content string) {
	t.Helper()
	if err := fs.MkdirAll(parent(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := afero.WriteFile(fs, path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func parent(p string) string {
	i := strings.LastIndex(p, "/")
	if i < 0 {
		return "/"
	}
	return p[:i]
}

func TestStageRemovalClassifiesPaths(t *testing.T) {
	u, _ := testUninstaller(t)
	app := model.AppInfo{Name: "foo", Source: "apt", InstallSizeKB: 2048}
	if err := u.StageRemoval(&app); err != nil {
		t.Fatal(err)
	}
	rem := app.Removable
	if !rem.Staged() {
		t.Fatal("expected staged paths")
	}
	if rem.ConfigKB == 0 || rem.CacheKB == 0 || rem.LocalDataKB == 0 {
		t.Fatalf("categories not sized: %+v", rem)
	}
	if rem.ResidualKB == 0 {
		t.Fatalf("dotfile residual not found: %+v", rem)
	}
	// ~/ paths must never be elevated; /etc/foo must be
	for _, e := range rem.Elevated {
		if strings.HasPrefix(e, "/home/") {
			t.Fatalf("home path must not require sudo: %s", e)
		}
	}
	if len(rem.Elevated) != 1 || rem.Elevated[0] != "/etc/foo" {
		t.Fatalf("expected /etc/foo elevated, got %v", rem.Elevated)
	}
	// /etc/foo exists but the current user cannot write it -> elevated
	if !strings.Contains(strings.Join(rem.Paths, " "), "/etc/foo") {
		t.Fatalf("system path not staged: %v", rem.Paths)
	}
}

func TestStageRemovalProtectedRefused(t *testing.T) {
	u, _ := testUninstaller(t)
	u.Protected = map[string]bool{"kernel": true}
	app := model.AppInfo{Name: "kernel", Source: "apt"}
	if err := u.StageRemoval(&app); !errors.Is(err, ErrProtected) {
		t.Fatalf("expected ErrProtected, got %v", err)
	}
}

func TestRemoveDeletesFilesAndRecordsHistory(t *testing.T) {
	u, cmder := testUninstaller(t)
	u.Log = nil // history tested separately
	app := model.AppInfo{Name: "foo", Source: "apt", InstallSizeKB: 2048}
	if err := u.StageRemoval(&app); err != nil {
		t.Fatal(err)
	}
	var steps []string
	freed, files, err := u.Remove(context.Background(), &app, func(s Step) {
		steps = append(steps, s.Text)
	})
	if err != nil {
		t.Fatal(err)
	}
	// package step must run elevated (sudo present or not — here no sudo, and
	// this test runs as the current user, so no sudo prefix expected)
	if len(cmder.ran) == 0 {
		t.Fatal("package command never ran")
	}
	if !strings.HasPrefix(cmder.ran[0], "apt purge -y foo") {
		t.Fatalf("package cmd = %q", cmder.ran[0])
	}
	if files != 4 {
		t.Fatalf("removed %d files, want 4", files)
	}
	if freed == 0 {
		t.Fatal("freed size should be positive")
	}
	for _, p := range []string{"/home/u/.config/foo", "/home/u/.cache/foo", "/home/u/.local/share/foo", "/home/u/.foo-rc"} {
		if exists, _ := afero.DirExists(u.FS, p); exists {
			t.Errorf("%s still exists after removal", p)
		}
	}
	if len(steps) == 0 {
		t.Fatal("no progress steps emitted")
	}
}

func TestRemoveElevatesSystemPathsOnce(t *testing.T) {
	u, cmder := testUninstaller(t)
	cmder.hasSudo = false // pretend we are not root and sudo exists
	cmder.hasSudo = true
	app := model.AppInfo{Name: "foo", Source: "pacman", InstallSizeKB: 100}
	if err := u.StageRemoval(&app); err != nil {
		t.Fatal(err)
	}
	if _, _, err := u.Remove(context.Background(), &app, nil); err != nil {
		t.Fatal(err)
	}
	// exactly one elevated rm call batching every system path
	var elev []string
	for _, c := range cmder.ran {
		if strings.Contains(c, "rm -rf") {
			elev = append(elev, c)
		}
	}
	if len(elev) != 1 {
		t.Fatalf("expected exactly 1 elevated rm call, got %v", elev)
	}
	if !strings.Contains(elev[0], "/etc/foo") {
		t.Fatalf("elevated call missing system path: %s", elev[0])
	}
	if strings.Contains(elev[0], ".config/foo") {
		t.Fatalf("elevated call must not contain home paths: %s", elev[0])
	}
}

func TestRemovePackageFailureAbortsFiles(t *testing.T) {
	u, cmder := testUninstaller(t)
	cmder.fail = map[string]bool{"apt": true}
	app := model.AppInfo{Name: "foo", Source: "apt", InstallSizeKB: 2048}
	if err := u.StageRemoval(&app); err != nil {
		t.Fatal(err)
	}
	_, files, err := u.Remove(context.Background(), &app, nil)
	if err == nil {
		t.Fatal("expected package failure")
	}
	if files != 0 {
		t.Fatalf("file cleanup must be skipped after package failure, removed %d", files)
	}
	// files still on disk
	if exists, _ := afero.DirExists(u.FS, "/home/u/.config/foo"); !exists {
		t.Fatal("config dir must survive a failed package step")
	}
}

func TestRemoveCommandTable(t *testing.T) {
	cases := []struct {
		src  string
		name string
		want string
	}{
		{"apt", "x", "sudo apt purge -y x"},
		{"dnf", "x", "sudo dnf remove -y x"},
		{"pacman", "x", "sudo pacman -Rns --noconfirm x"},
		{"aur", "x", "sudo pacman -Rns --noconfirm x"},
		{"zypper", "x", "sudo zypper remove -y x"},
		{"flatpak", "x", "flatpak uninstall --noninteractive -y --delete-data x"},
		{"snap", "x", "sudo snap remove --purge x"},
		{"npm", "x", "npm uninstall -g x"},
		{"pipx", "x", "pipx uninstall x"},
		{"cargo", "x", "cargo uninstall x"},
		{"gem", "x", "gem uninstall -a -x x"},
		{"brew", "x", "brew uninstall x"},
		{"nix", "x", "nix-env -e x"},
		{"system", "systemd-journal-logs", "sudo journalctl --vacuum-time=2d --vacuum-size=50M"},
	}
	for _, c := range cases {
		got := strings.Join(RemoveCommand(model.AppInfo{Name: c.name, Source: c.src}, true), " ")
		if got != c.want {
			t.Errorf("RemoveCommand(%s, %s) = %q, want %q", c.src, c.name, got, c.want)
		}
	}
}

func TestIsSafePath(t *testing.T) {
	home := "/home/user"
	cases := []struct {
		path string
		safe bool
	}{
		{"/", false},
		{"/etc", false},
		{"/var", false},
		{"/var/log", false},
		{"/usr", false},
		{"/usr/bin", false},
		{"/home/user", false},
		{"", false},
		{".", false},
		{"/home/user/.config/foo", true},
		{"/home/user/.cache/foo", true},
		{"/var/log/mycustomapp", true},
		{"/etc/mycustomapp", true},
	}
	for _, c := range cases {
		if got := isSafePath(c.path, home); got != c.safe {
			t.Errorf("isSafePath(%q) = %v, want %v", c.path, got, c.safe)
		}
	}
}

func TestRemovePackageNotFoundCleansResidualFiles(t *testing.T) {
	u, cmder := testUninstaller(t)
	// Seed a leftover config and cache directory before staging
	seed(t, u.FS, "/home/u/.config/ai.jan.Jan/config.json", "{}")
	seed(t, u.FS, "/home/u/.cache/ai.jan.Jan/cache.dat", "data")

	app := model.AppInfo{Name: "ai.jan.Jan", Source: "flatpak", InstallSizeKB: 100}
	if err := u.StageRemoval(&app); err != nil {
		t.Fatal(err)
	}

	// Simulate system-level flatpak failing with "No installed refs found"
	cmder.errs = map[string]error{
		"flatpak uninstall --noninteractive -y --delete-data ai.jan.Jan": errors.New("error: No installed refs found for ai.jan.Jan"),
	}

	freed, files, err := u.Remove(context.Background(), &app, nil)
	if err != nil {
		t.Fatalf("expected success with residual cleanup, got: %v", err)
	}
	if freed == 0 || files == 0 {
		t.Fatalf("expected files to be cleaned, freed=%d files=%d", freed, files)
	}

	// Verify that the user-scope fallback command ran
	foundUserFallback := false
	for _, cmd := range cmder.ran {
		if cmd == "flatpak uninstall --user --noninteractive -y ai.jan.Jan" {
			foundUserFallback = true
			break
		}
	}
	if !foundUserFallback {
		t.Fatalf("expected flatpak --user fallback to run, ran: %v", cmder.ran)
	}

	// Config files should have been removed
	if exists, _ := afero.DirExists(u.FS, "/home/u/.config/ai.jan.Jan"); exists {
		t.Fatal("config directory should have been purged")
	}
}

func TestRemoveFlatpakFatalFailureAborts(t *testing.T) {
	u, cmder := testUninstaller(t)
	seed(t, u.FS, "/home/u/.config/ai.jan.Jan/config.json", "{}")

	app := model.AppInfo{Name: "ai.jan.Jan", Source: "flatpak", InstallSizeKB: 100}
	if err := u.StageRemoval(&app); err != nil {
		t.Fatal(err)
	}

	// Fatal uninstaller failure that is not a missing reference
	cmder.errs = map[string]error{
		"flatpak uninstall --noninteractive -y --delete-data ai.jan.Jan": errors.New("error: DBus connection timed out"),
	}

	_, files, err := u.Remove(context.Background(), &app, nil)
	if err == nil {
		t.Fatal("expected Remove to return error on fatal flatpak failure")
	}
	if files != 0 {
		t.Fatalf("expected no files to be removed on fatal failure, removed %d", files)
	}
	if exists, _ := afero.DirExists(u.FS, "/home/u/.config/ai.jan.Jan"); !exists {
		t.Fatal("config directory must survive a fatal package failure")
	}
}
