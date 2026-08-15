package detector

import (
	"context"
	"strings"

	"github.com/swadhinbiswas/veet/internal/model"
)

type flatpakDetector struct {
	ex   Execer
	path string
}

// NewFlatpakDetector detects flatpak applications.
func NewFlatpakDetector(ex Execer) Detector {
	path, _ := ex.LookPath("flatpak")
	return &flatpakDetector{ex: ex, path: path}
}

func (d *flatpakDetector) Source() string  { return "flatpak" }
func (d *flatpakDetector) Available() bool { return d.path != "" }

func (d *flatpakDetector) Detect(ctx context.Context) ([]model.AppInfo, error) {
	out, err := Exec(ctx, d.ex, "flatpak", "list", "--app",
		"--columns=application,version,size,installation,origin")
	if err != nil {
		return nil, err
	}
	var apps []model.AppInfo
	for _, line := range strings.Split(out, "\n") {
		parts := strings.Split(line, "\t")
		if len(parts) < 3 || parts[0] == "" || !strings.Contains(parts[0], ".") {
			continue
		}
		app := model.AppInfo{
			Name:   parts[0],
			Source: "flatpak",
			Status: "Installed",
		}
		if len(parts) > 1 {
			app.Version = strings.TrimSpace(parts[1])
		}
		if len(parts) > 2 {
			app.InstallSizeKB = parseHumanSize(parts[2])
		}
		apps = append(apps, app)
	}
	return apps, nil
}
