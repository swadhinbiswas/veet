package detector

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/swadhinbiswas/veet/internal/model"
)

type appImageDetector struct {
	home string
}

// NewAppImageDetector discovers standalone .AppImage bundles in common directories.
func NewAppImageDetector(home string) Detector {
	return &appImageDetector{home: home}
}

func (d *appImageDetector) Source() string  { return "appimage" }
func (d *appImageDetector) Available() bool { return d.home != "" }

func (d *appImageDetector) Detect(ctx context.Context) ([]model.AppInfo, error) {
	var apps []model.AppInfo
	dirs := []string{
		filepath.Join(d.home, "Applications"),
		filepath.Join(d.home, ".local", "bin"),
		filepath.Join(d.home, "bin"),
		filepath.Join(d.home, "Downloads"),
		"/opt",
	}

	seen := map[string]bool{}

	for _, dir := range dirs {
		if ctx.Err() != nil {
			break
		}
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if ctx.Err() != nil {
				break
			}
			if e.IsDir() {
				continue
			}
			name := e.Name()
			lower := strings.ToLower(name)
			if !strings.HasSuffix(lower, ".appimage") {
				continue
			}
			fullPath := filepath.Join(dir, name)
			if seen[fullPath] {
				continue
			}
			seen[fullPath] = true

			fi, err := e.Info()
			if err != nil {
				continue
			}
			sizeKB := fi.Size() / 1024
			if sizeKB == 0 && fi.Size() > 0 {
				sizeKB = 1
			}

			cleanName := strings.TrimSuffix(name, filepath.Ext(name))
			rem := model.RemovableFiles{
				Paths:     []string{fullPath},
				PackageKB: sizeKB,
			}
			if !strings.HasPrefix(fullPath, d.home) {
				rem.Elevated = []string{fullPath}
			}

			// Look for matching desktop files
			desktopPath := filepath.Join(d.home, ".local", "share", "applications", "appimagekit-"+cleanName+".desktop")
			if _, err := os.Stat(desktopPath); err == nil {
				rem.Paths = append(rem.Paths, desktopPath)
			}

			apps = append(apps, model.AppInfo{
				Name:          cleanName,
				Version:       "-",
				Source:        "appimage",
				InstalledOn:   fi.ModTime(),
				InstallSizeKB: sizeKB,
				Status:        "Installed",
				Removable:     rem,
			})
		}
	}
	return apps, nil
}
