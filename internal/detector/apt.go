package detector

import (
	"context"
	"strconv"
	"strings"

	"github.com/swadhinbiswas/veet/internal/model"
)

type aptDetector struct {
	ex   Execer
	path string
}

// NewAptDetector detects deb-based packages (Debian/Ubuntu).
func NewAptDetector(ex Execer) Detector {
	path, _ := ex.LookPath("dpkg-query")
	return &aptDetector{ex: ex, path: path}
}

func (d *aptDetector) Source() string  { return "apt" }
func (d *aptDetector) Available() bool { return d.path != "" }

func (d *aptDetector) Detect(ctx context.Context) ([]model.AppInfo, error) {
	out, err := Exec(ctx, d.ex, "dpkg-query", "-W", "-f=${binary:Package}\t${Version}\t${Installed-Size}\n")
	if err != nil {
		return nil, err
	}
	var apps []model.AppInfo
	for _, line := range strings.Split(out, "\n") {
		parts := strings.Split(strings.TrimSpace(line), "\t")
		if len(parts) < 3 || parts[0] == "" {
			continue
		}
		sizeKB, _ := strconv.ParseInt(parts[2], 10, 64)
		apps = append(apps, model.AppInfo{
			Name:          parts[0],
			Version:       parts[1],
			Source:        "apt",
			InstallSizeKB: sizeKB,
			Status:        "Installed",
		})
	}
	return apps, nil
}
