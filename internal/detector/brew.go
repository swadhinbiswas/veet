package detector

import (
	"context"
	"strings"

	"github.com/swadhinbiswas/veet/internal/model"
)

type brewDetector struct {
	ex   Execer
	path string
}

// NewBrewDetector detects Homebrew formulae and casks on Linux.
func NewBrewDetector(ex Execer) Detector {
	path, _ := ex.LookPath("brew")
	return &brewDetector{ex: ex, path: path}
}

func (d *brewDetector) Source() string  { return "brew" }
func (d *brewDetector) Available() bool { return d.path != "" }

func (d *brewDetector) Detect(ctx context.Context) ([]model.AppInfo, error) {
	out, err := Exec(ctx, d.ex, "brew", "list", "--versions")
	if err != nil {
		return nil, err
	}
	var apps []model.AppInfo
	for _, line := range strings.Split(out, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		name := fields[0]
		version := fields[1]
		apps = append(apps, model.AppInfo{
			Name:    name,
			Version: version,
			Source:  "brew",
			Status:  "Installed",
		})
	}
	return apps, nil
}
