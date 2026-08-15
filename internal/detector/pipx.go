package detector

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/swadhinbiswas/veet/internal/model"
)

type pipxDetector struct {
	ex   Execer
	path string
}

// NewPipxDetector detects globally installed pipx applications.
func NewPipxDetector(ex Execer) Detector {
	path, _ := ex.LookPath("pipx")
	return &pipxDetector{ex: ex, path: path}
}

func (d *pipxDetector) Source() string  { return "pipx" }
func (d *pipxDetector) Available() bool { return d.path != "" }

func (d *pipxDetector) Detect(ctx context.Context) ([]model.AppInfo, error) {
	out, err := Exec(ctx, d.ex, "pipx", "list", "--json")
	if err != nil {
		return nil, err
	}
	var parsed struct {
		Venvs map[string]struct {
			Metadata struct {
				MainPackage struct {
					PackageVersion string `json:"package_version"`
				} `json:"main_package"`
			} `json:"metadata"`
		} `json:"venvs"`
	}
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		return nil, err
	}
	home, _ := os.UserHomeDir()
	var apps []model.AppInfo
	for name, venv := range parsed.Venvs {
		dir := filepath.Join(home, ".local", "pipx", "venvs", name)
		size, err := dirSizeKB(dir)
		if err != nil {
			size = 0
		}
		apps = append(apps, model.AppInfo{
			Name:          name,
			Version:       venv.Metadata.MainPackage.PackageVersion,
			Source:        "pipx",
			InstallSizeKB: size,
			Status:        "Installed",
		})
	}
	return apps, nil
}
