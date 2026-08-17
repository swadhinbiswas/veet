package detector

import (
	"context"
	"strings"

	"github.com/swadhinbiswas/veet/internal/model"
)

type nixDetector struct {
	ex   Execer
	path string
}

// NewNixDetector detects user-installed Nix packages.
func NewNixDetector(ex Execer) Detector {
	path, _ := ex.LookPath("nix-env")
	if path == "" {
		path, _ = ex.LookPath("nix")
	}
	return &nixDetector{ex: ex, path: path}
}

func (d *nixDetector) Source() string  { return "nix" }
func (d *nixDetector) Available() bool { return d.path != "" }

func (d *nixDetector) Detect(ctx context.Context) ([]model.AppInfo, error) {
	out, err := Exec(ctx, d.ex, "nix-env", "-q")
	if err != nil {
		return nil, err
	}
	var apps []model.AppInfo
	for _, line := range strings.Split(out, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		name := trimmed
		version := "-"
		lastDash := strings.LastIndex(trimmed, "-")
		if lastDash > 0 && lastDash < len(trimmed)-1 {
			candidateVer := trimmed[lastDash+1:]
			if len(candidateVer) > 0 && (candidateVer[0] >= '0' && candidateVer[0] <= '9') {
				name = trimmed[:lastDash]
				version = candidateVer
			}
		}
		apps = append(apps, model.AppInfo{
			Name:    name,
			Version: version,
			Source:  "nix",
			Status:  "Installed",
		})
	}
	return apps, nil
}
