package detector

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/swadhinbiswas/veet/internal/model"
)

type cacheDetector struct {
	home string
}

// NewCacheDetector surfaces reclaimable caches, shared data and orphaned
// dotfiles under the user's home directory. Always available.
func NewCacheDetector(home string) Detector {
	return &cacheDetector{home: home}
}

func (d *cacheDetector) Source() string  { return "cache" }
func (d *cacheDetector) Available() bool { return d.home != "" }

// skipHome are top-level home entries that must never be reported.
var skipHome = map[string]bool{
	".config": true, ".local": true, ".cache": true, ".ssh": true,
	".gnupg": true, ".mozilla": true, ".npm": true, ".cargo": true,
	".rustup": true, ".go": true, ".pyenv": true, ".nvm": true,
	".node_modules": true, ".bundle": true, ".gem": true, ".vim": true,
	".oh-my-zsh": true, ".tmux": true, ".dbus": true, ".gconf": true,
	".pki": true, ".x": true, ".Trash": true, ".thumbnails": true,
	".pulse": true, ".esd_auth": true, ".ICEauthority": true,
	".Xauthority": true, ".Xsession": true, ".xorgxrdb": true,
}

// knownHomeFiles are regular files that are part of the shell environment.
var knownHomeFiles = map[string]bool{
	".bashrc": true, ".bash_profile": true, ".bash_login": true,
	".profile": true, ".zshrc": true, ".zprofile": true, ".zshenv": true,
	".gitconfig": true, ".gitignore": true, ".xinitrc": true,
	".xprofile": true, ".Xresources": true, ".inputrc": true,
	".wget-hsts": true, ".lesshst": true, ".viminfo": true,
	".python_history": true, ".zsh_history": true, ".bash_history": true,
	".config": true, ".local": true,
}

func (d *cacheDetector) Detect(ctx context.Context) ([]model.AppInfo, error) {
	var apps []model.AppInfo

	scanDir := func(dir string, prefix string) {
		entries, err := os.ReadDir(dir)
		if err != nil {
			return
		}
		for _, e := range entries {
			if ctx.Err() != nil {
				return
			}
			name := e.Name()
			if strings.HasPrefix(name, ".") {
				continue
			}
			path := filepath.Join(dir, name)
			size, err := dirSizeKB(path)
			if err != nil {
				continue
			}
			if size < 16 {
				continue // noise threshold
			}
			rem := model.RemovableFiles{Paths: []string{path}}
			rem.CacheKB = size
			apps = append(apps, model.AppInfo{
				Name:          prefix + name,
				Version:       "-",
				Source:        "cache",
				InstallSizeKB: size,
				Status:        "Installed",
				Removable:     rem,
			})
		}
	}

	scanDir(filepath.Join(d.home, ".cache"), "")
	scanDir(filepath.Join(d.home, ".local", "share"), "share/")

	// Orphaned dotfile directories in $HOME.
	if entries, err := os.ReadDir(d.home); err == nil {
		for _, e := range entries {
			if ctx.Err() != nil {
				break
			}
			name := e.Name()
			if !strings.HasPrefix(name, ".") || !e.IsDir() {
				continue
			}
			if skipHome[name] || knownHomeFiles[name] {
				continue
			}
			path := filepath.Join(d.home, name)
			size, err := dirSizeKB(path)
			if err != nil || size < 16 {
				continue
			}
			rem := model.RemovableFiles{Paths: []string{path}}
			rem.ResidualKB = size
			apps = append(apps, model.AppInfo{
				Name:          "." + strings.TrimPrefix(name, "."),
				Version:       "-",
				Source:        "cache",
				InstallSizeKB: size,
				Status:        "Installed",
				Removable:     rem,
			})
		}
	}

	sort.Slice(apps, func(i, j int) bool { return apps[i].InstallSizeKB > apps[j].InstallSizeKB })
	return apps, nil
}
