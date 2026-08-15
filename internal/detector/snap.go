package detector

import (
	"context"
	"strings"

	"github.com/swadhinbiswas/veet/internal/model"
)

type snapDetector struct {
	ex   Execer
	path string
}

// NewSnapDetector detects snap packages.
func NewSnapDetector(ex Execer) Detector {
	path, _ := ex.LookPath("snap")
	return &snapDetector{ex: ex, path: path}
}

func (d *snapDetector) Source() string  { return "snap" }
func (d *snapDetector) Available() bool { return d.path != "" }

func (d *snapDetector) Detect(ctx context.Context) ([]model.AppInfo, error) {
	out, err := Exec(ctx, d.ex, "snap", "list")
	if err != nil {
		return nil, err
	}
	lines := strings.Split(out, "\n")
	var names []string
	versions := map[string]string{}
	for _, line := range lines[1:] {
		fields := strings.Fields(line)
		if len(fields) >= 2 {
			names = append(names, fields[0])
			versions[fields[0]] = fields[1]
		}
	}

	sink := &appSink{}
	workerPool(names, 6, func(name string) {
		var sizeKB int64
		if info, err := Exec(ctx, d.ex, "snap", "info", name); err == nil {
			for _, line := range strings.Split(info, "\n") {
				if strings.HasPrefix(strings.TrimSpace(line), "installed:") {
					val := strings.TrimSpace(strings.TrimPrefix(line, "installed:"))
					sizeKB = parseHumanSize(val)
					break
				}
			}
		}
		sink.add(model.AppInfo{
			Name:          name,
			Version:       versions[name],
			Source:        "snap",
			InstallSizeKB: sizeKB,
			Status:        "Installed",
		})
	})
	return sink.apps, nil
}
