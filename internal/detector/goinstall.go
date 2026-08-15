package detector

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/swadhinbiswas/veet/internal/model"
)

type goinstallDetector struct {
	ex   Execer
	path string
}

// NewGoInstallDetector detects binaries installed via `go install`.
func NewGoInstallDetector(ex Execer) Detector {
	path, _ := ex.LookPath("go")
	return &goinstallDetector{ex: ex, path: path}
}

func (d *goinstallDetector) Source() string  { return "go" }
func (d *goinstallDetector) Available() bool { return d.path != "" }

func (d *goinstallDetector) Detect(ctx context.Context) ([]model.AppInfo, error) {
	gopath := ""
	if dir, err := Exec(ctx, d.ex, "go", "env", "GOPATH"); err == nil {
		gopath = strings.TrimSpace(dir)
	} else {
		return nil, err
	}
	bin := filepath.Join(gopath, "bin")
	entries, err := os.ReadDir(bin)
	if err != nil {
		return nil, err
	}
	var apps []model.AppInfo
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		info, err := e.Info()
		if err != nil || info.Mode()&0111 == 0 {
			continue // heuristic: only executable files are go-installed binaries
		}
		size := (info.Size() + 1023) / 1024
		apps = append(apps, model.AppInfo{
			Name:          e.Name(),
			Version:       "unknown",
			Source:        "go",
			InstallSizeKB: size,
			Status:        "Installed",
		})
	}
	return apps, nil
}
