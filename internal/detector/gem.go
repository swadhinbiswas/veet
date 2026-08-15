package detector

import (
	"context"
	"path/filepath"
	"strings"

	"github.com/swadhinbiswas/veet/internal/model"
)

type gemDetector struct {
	ex   Execer
	path string
}

// NewGemDetector detects locally installed Ruby gems.
func NewGemDetector(ex Execer) Detector {
	path, _ := ex.LookPath("gem")
	return &gemDetector{ex: ex, path: path}
}

func (d *gemDetector) Source() string  { return "gem" }
func (d *gemDetector) Available() bool { return d.path != "" }

func (d *gemDetector) Detect(ctx context.Context) ([]model.AppInfo, error) {
	out, err := Exec(ctx, d.ex, "gem", "list", "--local")
	if err != nil {
		return nil, err
	}
	gemdir := ""
	if dir, err := Exec(ctx, d.ex, "gem", "environment", "gemdir"); err == nil {
		gemdir = strings.TrimSpace(dir)
	}
	var apps []model.AppInfo
	for _, line := range strings.Split(out, "\n") {
		name, vers, ok := strings.Cut(line, " (")
		if !ok || strings.Contains(name, " ") {
			// Skip lines like "*** LOCAL GEMS ***"
			continue
		}
		name = strings.TrimSpace(name)
		vers = strings.TrimSuffix(vers, ")")
		version := ""
		if v, _, found := strings.Cut(vers, ", "); found {
			version = v
		} else {
			version = vers
		}
		app := model.AppInfo{
			Name:    name,
			Version: version,
			Source:  "gem",
			Status:  "Installed",
		}
		if gemdir != "" {
			dir := filepath.Join(gemdir, "gems", name+"-"+version)
			if size, err := dirSizeKB(dir); err == nil {
				app.InstallSizeKB = size
			}
		}
		apps = append(apps, app)
	}
	return apps, nil
}
