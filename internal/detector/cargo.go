package detector

import (
	"context"
	"os"
	"strings"

	"github.com/swadhinbiswas/veet/internal/model"
)

type cargoDetector struct {
	ex   Execer
	path string
}

// NewCargoDetector detects binaries installed via `cargo install`.
func NewCargoDetector(ex Execer) Detector {
	path, _ := ex.LookPath("cargo")
	return &cargoDetector{ex: ex, path: path}
}

func (d *cargoDetector) Source() string  { return "cargo" }
func (d *cargoDetector) Available() bool { return d.path != "" }

func (d *cargoDetector) Detect(ctx context.Context) ([]model.AppInfo, error) {
	out, err := Exec(ctx, d.ex, "cargo", "install", "--list")
	if err != nil {
		return nil, err
	}
	var apps []model.AppInfo
	curIdx := -1
	for _, line := range strings.Split(out, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			curIdx = -1
			continue
		}
		indented := line != trimmed
		if !indented {
			name, ver, ok := strings.Cut(trimmed, " v")
			if !ok {
				continue
			}
			apps = append(apps, model.AppInfo{
				Name:    strings.TrimSpace(name),
				Version: strings.TrimSpace(ver),
				Source:  "cargo",
				Status:  "Installed",
			})
			curIdx = len(apps) - 1
		} else if strings.HasPrefix(trimmed, "/") {
			if curIdx < 0 {
				continue
			}
			if size, err := binSizeKB(trimmed); err == nil {
				apps[curIdx].InstallSizeKB += size
			}
		}
	}
	return apps, nil
}

// binSizeKB measures a single regular file in KiB.
func binSizeKB(p string) (int64, error) {
	info, err := os.Stat(p)
	if err != nil {
		return 0, err
	}
	return (info.Size() + 1023) / 1024, nil
}
