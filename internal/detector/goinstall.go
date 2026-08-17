package detector

import (
	"context"
	"debug/buildinfo"
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
			continue // only executable files
		}
		fullPath := filepath.Join(bin, e.Name())
		size := (info.Size() + 1023) / 1024
		version := "-"
		homepage := ""
		if bi, err := buildinfo.ReadFile(fullPath); err == nil {
			if bi.Main.Version != "" && bi.Main.Version != "(devel)" {
				version = bi.Main.Version
			}
			if bi.Main.Path != "" {
				homepage = "https://" + bi.Main.Path
			}
		}
		apps = append(apps, model.AppInfo{
			Name:          e.Name(),
			Version:       version,
			Source:        "go",
			InstalledOn:   info.ModTime(),
			InstallSizeKB: size,
			Status:        "Installed",
			Homepage:      homepage,
		})
	}
	return apps, nil
}
