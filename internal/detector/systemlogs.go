package detector

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/swadhinbiswas/veet/internal/model"
)

type systemLogsDetector struct{}

// NewSystemLogsDetector surfaces systemd journal logs and rotated log archives.
func NewSystemLogsDetector() Detector {
	return &systemLogsDetector{}
}

func (d *systemLogsDetector) Source() string  { return "system" }
func (d *systemLogsDetector) Available() bool { return true }

func (d *systemLogsDetector) Detect(ctx context.Context) ([]model.AppInfo, error) {
	var apps []model.AppInfo

	// Check /var/log/journal
	journalPath := "/var/log/journal"
	if size, err := dirSizeKB(journalPath); err == nil && size > 1024 {
		rem := model.RemovableFiles{
			Paths:    []string{journalPath},
			Elevated: []string{journalPath},
			LogKB:    size,
		}
		apps = append(apps, model.AppInfo{
			Name:          "systemd-journal-logs",
			Version:       "-",
			Source:        "system",
			InstallSizeKB: size,
			Status:        "Installed",
			Removable:     rem,
		})
	}

	// Check rotated archived logs in /var/log
	if entries, err := os.ReadDir("/var/log"); err == nil {
		var archivePaths []string
		var totalKB int64
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			name := e.Name()
			if strings.HasSuffix(name, ".gz") || strings.HasSuffix(name, ".old") || strings.HasSuffix(name, ".1") {
				p := filepath.Join("/var/log", name)
				if fi, err := e.Info(); err == nil {
					totalKB += fi.Size() / 1024
					archivePaths = append(archivePaths, p)
				}
			}
		}
		if totalKB > 512 && len(archivePaths) > 0 {
			rem := model.RemovableFiles{
				Paths:    archivePaths,
				Elevated: archivePaths,
				LogKB:    totalKB,
			}
			apps = append(apps, model.AppInfo{
				Name:          "system-log-archives",
				Version:       "-",
				Source:        "system",
				InstallSizeKB: totalKB,
				Status:        "Installed",
				Removable:     rem,
			})
		}
	}

	return apps, nil
}
