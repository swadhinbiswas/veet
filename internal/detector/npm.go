package detector

import (
	"context"
	"encoding/json"
	"path/filepath"
	"strings"

	"github.com/swadhinbiswas/veet/internal/model"
)

type npmDetector struct {
	ex   Execer
	path string
}

// NewNpmDetector detects globally installed npm packages.
func NewNpmDetector(ex Execer) Detector {
	path, _ := ex.LookPath("npm")
	return &npmDetector{ex: ex, path: path}
}

func (d *npmDetector) Source() string  { return "npm" }
func (d *npmDetector) Available() bool { return d.path != "" }

func (d *npmDetector) Detect(ctx context.Context) ([]model.AppInfo, error) {
	out, err := Exec(ctx, d.ex, "npm", "list", "-g", "--depth=0", "--json")
	if err != nil {
		return nil, err
	}
	var parsed struct {
		Dependencies map[string]struct {
			Version string `json:"version"`
		} `json:"dependencies"`
	}
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		return nil, err
	}
	root, _ := Exec(ctx, d.ex, "npm", "root", "-g")

	sink := &appSink{}
	for name, dep := range parsed.Dependencies {
		dir := strings.TrimSpace(root)
		size, err := dirSizeKB(filepath.Join(dir, name))
		if err != nil {
			size = 0
		}
		sink.add(model.AppInfo{
			Name:          name,
			Version:       dep.Version,
			Source:        "npm",
			InstallSizeKB: size,
			Status:        "Installed",
		})
	}
	return sink.apps, nil
}
