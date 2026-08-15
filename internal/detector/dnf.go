package detector

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/swadhinbiswas/veet/internal/model"
)

type rpmDetector struct {
	ex   Execer
	path string
}

// NewDnfDetector detects rpm packages via rpm (Fedora/RHEL/openSUSE etc).
func NewDnfDetector(ex Execer) Detector {
	path, _ := ex.LookPath("rpm")
	return &rpmDetector{ex: ex, path: path}
}

func (d *rpmDetector) Source() string  { return "dnf" }
func (d *rpmDetector) Available() bool { return d.path != "" }

func (d *rpmDetector) Detect(ctx context.Context) ([]model.AppInfo, error) {
	out, err := Exec(ctx, d.ex, "rpm", "-qa", "--queryformat",
		"%{INSTALLTIME}\t%{SIZE}\t%{NAME}\t%{VERSION}\n")
	if err != nil {
		return nil, err
	}
	var apps []model.AppInfo
	for _, line := range strings.Split(out, "\n") {
		parts := strings.Split(strings.TrimSpace(line), "\t")
		if len(parts) < 4 || parts[2] == "" {
			continue
		}
		installed := time.Time{}
		if sec, err := strconv.ParseInt(parts[0], 10, 64); err == nil && sec > 0 {
			installed = time.Unix(sec, 0)
		}
		sizeBytes, _ := strconv.ParseInt(parts[1], 10, 64)
		apps = append(apps, model.AppInfo{
			Name:          parts[2],
			Version:       parts[3],
			Source:        "dnf",
			InstalledOn:   installed,
			InstallSizeKB: sizeBytes / 1024,
			Status:        "Installed",
		})
	}
	return apps, nil
}
