package uninstaller

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/afero"
	"golang.org/x/sys/unix"

	"github.com/swadhinbiswas/veet/internal/history"
	"github.com/swadhinbiswas/veet/internal/model"
)

// Cmder abstracts process execution for removal commands.
type Cmder interface {
	LookPath(name string) (string, error)
	Run(ctx context.Context, name string, args ...string) error
}

// RealCmder shells out via os/exec.
type RealCmder struct{}

func (RealCmder) LookPath(name string) (string, error) { return exec.LookPath(name) }
func (RealCmder) Run(ctx context.Context, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s: %v: %s", name, err, strings.TrimSpace(string(out)))
	}
	return nil
}

// Uninstaller performs staged, confirmed removal of apps.
type Uninstaller struct {
	FS        afero.Fs
	Cmd       Cmder
	Home      string
	Log       *history.Log
	Protected map[string]bool

	// DryRun stages and reports but never executes removal.
	DryRun bool
}

// New builds an Uninstaller bound to the real filesystem.
func New(home string, log *history.Log, protected map[string]bool) *Uninstaller {
	return &Uninstaller{
		FS:        afero.NewOsFs(),
		Cmd:       RealCmder{},
		Home:      home,
		Log:       log,
		Protected: protected,
	}
}

// StageResult is the outcome of a deep-clean scan.
type StageResult struct {
	App  model.AppInfo
	Note string // e.g. "protected system component"
}

// ErrProtected signals a refused deep-clean.
var ErrProtected = errors.New("protected system component: deep clean refused")

// StageRemoval walks candidate config/cache/log/data/residual paths for an
// app and fills Removable. It never touches disk — pure inspection.
func (u *Uninstaller) StageRemoval(app *model.AppInfo) error {
	if app.Source == "cache" || app.Source == "appimage" || app.Source == "system" {
		if !app.Removable.Staged() {
			app.Removable.Paths = []string{}
		}
		return nil
	}
	if u.Protected[app.Name] || app.Protected {
		return ErrProtected
	}

	var rem model.RemovableFiles
	rem.PackageKB = app.InstallSizeKB

	home := u.Home
	candidates := []struct {
		path     string
		category *int64
	}{
		{filepath.Join(home, ".config", app.Name), &rem.ConfigKB},
		{filepath.Join(home, ".cache", app.Name), &rem.CacheKB},
		{filepath.Join(home, ".local", "share", app.Name), &rem.LocalDataKB},
		{filepath.Join(home, ".local", "state", app.Name), &rem.LocalDataKB},
		{filepath.Join(home, "."+app.Name), &rem.ResidualKB},
	}
	for _, extra := range model.HomeCandidates(app.Source, app.Name) {
		candidates = append(candidates, struct {
			path     string
			category *int64
		}{filepath.Join(home, extra), &rem.LocalDataKB})
	}
	for _, sys := range model.SystemCandidates(app.Name) {
		candidates = append(candidates, struct {
			path     string
			category *int64
		}{sys, &rem.ResidualKB})
	}

	for _, c := range candidates {
		exists, err := afero.DirExists(u.FS, c.path)
		if err != nil || !exists {
			continue
		}
		size, err := dirSizeKB(u.FS, c.path)
		if err != nil {
			continue
		}
		elevated := !u.isUserWritable(c.path)
		rem.AddPath(c.path, size, elevated, c.category)
	}

	app.Removable = rem
	return nil
}

// isUserWritable reports whether the current user can write to a path.
// Paths inside the home directory are always considered user-writable.
func (u *Uninstaller) isUserWritable(p string) bool {
	if strings.HasPrefix(p, u.Home) {
		return true
	}
	if os.Geteuid() == 0 {
		return true
	}
	for cur := p; ; cur = filepath.Dir(cur) {
		if err := unix.Access(cur, unix.W_OK); err == nil {
			return true
		}
		if _, err := afero.Exists(u.FS, cur); err != nil {
			// Walk up to the nearest existing ancestor.
			if cur == "/" {
				return false
			}
			continue
		}
		return false
	}
}

// needsSudo reports whether a removal command must be elevated.
func (u *Uninstaller) needsSudo() bool {
	if os.Geteuid() == 0 {
		return false
	}
	_, err := u.Cmd.LookPath("sudo")
	return err == nil
}

// Step is streamed back to the UI as work progresses.
type Step struct {
	App     string
	Source  string
	Text    string
	Percent float64
	Err     error
}

// RemoveCommand returns the package-manager removal command for a source.
func RemoveCommand(app model.AppInfo, sudo bool) []string {
	args := []string{app.Name}
	switch app.Source {
	case "apt":
		args = append([]string{"apt", "purge", "-y"}, args...)
	case "dnf":
		args = append([]string{"dnf", "remove", "-y"}, args...)
	case "pacman", "aur":
		args = append([]string{"pacman", "-Rns", "--noconfirm"}, args...)
	case "orphan":
		if _, err := exec.LookPath("pacman"); err == nil {
			args = append([]string{"pacman", "-Rns", "--noconfirm"}, args...)
		} else if _, err := exec.LookPath("apt"); err == nil {
			args = append([]string{"apt", "purge", "-y"}, args...)
		} else if _, err := exec.LookPath("dnf"); err == nil {
			args = append([]string{"dnf", "remove", "-y"}, args...)
		} else {
			args = append([]string{"zypper", "remove", "-y"}, args...)
		}
	case "zypper":
		args = append([]string{"zypper", "remove", "-y"}, args...)
	case "flatpak":
		return []string{"flatpak", "uninstall", "-y", app.Name}
	case "snap":
		args = append([]string{"snap", "remove"}, args...)
	case "npm":
		return []string{"npm", "uninstall", "-g", app.Name}
	case "pipx":
		return []string{"pipx", "uninstall", app.Name}
	case "cargo":
		return []string{"cargo", "uninstall", app.Name}
	case "gem":
		return []string{"gem", "uninstall", app.Name}
	case "go":
		return []string{"rm", "-rf", filepath.Join(gopathBin(), app.Name)}
	case "cache", "appimage", "system":
		return nil
	}
	if sudo {
		args = append([]string{"sudo"}, args...)
	}
	return args
}

func gopathBin() string {
	if gp := os.Getenv("GOPATH"); gp != "" {
		return filepath.Join(gp, "bin")
	}
	if home, err := os.UserHomeDir(); err == nil {
		return filepath.Join(home, "go", "bin")
	}
	return "/root/go/bin"
}

// Remove uninstalls one app: package step, then staged file cleanup.
// Steps are reported through the onStep callback. Removal is recorded
// in the history log on completion (success or failure).
func (u *Uninstaller) Remove(ctx context.Context, app *model.AppInfo, onStep func(Step)) (freedKB int64, files int, err error) {
	total := 1
	if app.Removable.Staged() {
		total += app.Removable.FileCount()
	}
	done := 0
	emit := func(text string, e error) {
		if e != nil {
			err = e
		}
		if onStep != nil {
			p := float64(done) / float64(total)
			if p > 1 {
				p = 1
			}
			onStep(Step{App: app.Name, Source: app.Source, Text: text, Percent: p, Err: e})
		}
	}

	cmd := RemoveCommand(*app, u.needsSudo())
	if cmd != nil {
		emit("running: "+strings.Join(cmd, " "), nil)
		done++
		if !u.DryRun {
			if err := u.Cmd.Run(ctx, cmd[0], cmd[1:]...); err != nil {
				emit("package removal failed", err)
				u.record(app, 0, 0, false, err.Error())
				return 0, 0, err
			}
		}
		done++
	}

	if app.Removable.Staged() {
		userPaths := u.userPaths(app)
		for _, p := range userPaths {
			if ctx.Err() != nil {
				emit("cancelled", ctx.Err())
				u.record(app, freedKB, files, false, "cancelled")
				return freedKB, files, ctx.Err()
			}
			emit("removing: "+p, nil)
			if !u.DryRun {
				if err := u.FS.RemoveAll(p); err != nil {
					emit("remove failed: "+p, err)
					u.record(app, freedKB, files, false, err.Error())
					return freedKB, files, err
				}
			}
			files++
			done++
			emit("removed: "+p, nil)
		}
		if elevated := u.elevatedPaths(app); len(elevated) > 0 {
			emit(fmt.Sprintf("elevated removal of %d system path(s)", len(elevated)), nil)
			if !u.DryRun {
				args := append([]string{"rm", "-rf", "--"}, elevated...)
				argv := append([]string{"sudo"}, args...)
				if err := u.Cmd.Run(ctx, argv[0], argv[1:]...); err != nil {
					emit("elevated removal failed", err)
					u.record(app, freedKB, files, false, err.Error())
					return freedKB, files, err
				}
			}
			files += len(elevated)
			done += len(elevated)
		}
	}

	freedKB = app.Removable.TotalKB()
	u.record(app, freedKB, files, true, "")
	emit("done", nil)
	return freedKB, files, nil
}

func (u *Uninstaller) userPaths(app *model.AppInfo) []string {
	var out []string
	for _, p := range app.Removable.Paths {
		if !u.isUserWritable(p) {
			continue
		}
		out = append(out, p)
	}
	return out
}

func (u *Uninstaller) elevatedPaths(app *model.AppInfo) []string {
	var out []string
	for _, p := range app.Removable.Paths {
		if u.isUserWritable(p) {
			continue
		}
		out = append(out, p)
	}
	return out
}

func (u *Uninstaller) record(app *model.AppInfo, freedKB int64, files int, ok bool, detail string) {
	if u.Log == nil {
		return
	}
	status := "ok"
	if !ok {
		status = "failed"
	}
	u.Log.Append(history.Entry{
		Time:    time.Now(),
		App:     app.Name,
		Source:  app.Source,
		Status:  status,
		FreedKB: freedKB,
		Files:   files,
		Detail:  detail,
	})
}
