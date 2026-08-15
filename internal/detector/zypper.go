package detector

import (
	"context"
	"strings"

	"github.com/swadhinbiswas/veet/internal/model"
)

// NewZypperDetector detects SUSE/openSUSE packages.
func NewZypperDetector(ex Execer) Detector {
	path, _ := ex.LookPath("zypper")
	return &simpleCmdDetector{
		ex:      ex,
		path:    path,
		source:  "zypper",
		listCmd: []string{"zypper", "se", "--installed-only", "-s"},
	}
}

type simpleCmdDetector struct {
	ex      Execer
	path    string
	source  string
	listCmd []string
}

func (d *simpleCmdDetector) Source() string  { return d.source }
func (d *simpleCmdDetector) Available() bool { return d.path != "" }

func (d *simpleCmdDetector) Detect(ctx context.Context) ([]model.AppInfo, error) {
	out, err := Exec(ctx, d.ex, d.listCmd[0], d.listCmd[1:]...)
	if err != nil {
		return nil, err
	}
	var names []string
	versions := map[string]string{}
	for _, line := range strings.Split(out, "\n") {
		// zypper se -s prints: S | Name | Type | Version | Arch | Repository
		fields := strings.Fields(line)
		if len(fields) < 4 || !strings.Contains(fields[0], "i") {
			continue
		}
		names = append(names, fields[1])
		versions[fields[1]] = fields[3]
	}

	sink := &appSink{}
	workerPool(names, 6, func(name string) {
		var sizeKB int64
		var installed string
		if info, err := Exec(ctx, d.ex, "zypper", "info", name); err == nil {
			for _, line := range strings.Split(info, "\n") {
				key, val, ok := strings.Cut(line, ":")
				if !ok {
					continue
				}
				key, val = strings.TrimSpace(key), strings.TrimSpace(val)
				switch key {
				case "Installed Size":
					sizeKB = parseHumanSize(val)
				case "Installed":
					installed = val
				}
			}
		}
		sink.add(model.AppInfo{
			Name:          name,
			Version:       versions[name],
			Source:        "zypper",
			InstalledOn:   parseDate(installed),
			InstallSizeKB: sizeKB,
			Status:        "Installed",
		})
	})
	return sink.apps, nil
}
