package detector

import (
	"bufio"
	"context"
	"strings"

	"github.com/swadhinbiswas/veet/internal/model"
)

type orphanDetector struct {
	ex Execer
}

// NewOrphanDetector finds unused orphaned dependencies across package managers.
func NewOrphanDetector(ex Execer) Detector {
	return &orphanDetector{ex: ex}
}

func (d *orphanDetector) Source() string { return "orphan" }

func (d *orphanDetector) Available() bool {
	for _, tool := range []string{"pacman", "apt-get", "dnf", "zypper"} {
		if _, err := d.ex.LookPath(tool); err == nil {
			return true
		}
	}
	return false
}

func (d *orphanDetector) Detect(ctx context.Context) ([]model.AppInfo, error) {
	if _, err := d.ex.LookPath("pacman"); err == nil {
		return d.detectPacmanOrphans(ctx)
	}
	if _, err := d.ex.LookPath("apt-get"); err == nil {
		return d.detectAptOrphans(ctx)
	}
	if _, err := d.ex.LookPath("dnf"); err == nil {
		return d.detectDnfOrphans(ctx)
	}
	return nil, nil
}

func (d *orphanDetector) detectPacmanOrphans(ctx context.Context) ([]model.AppInfo, error) {
	out, err := Exec(ctx, d.ex, "pacman", "-Qdt")
	if err != nil || strings.TrimSpace(out) == "" {
		return nil, nil
	}

	var apps []model.AppInfo
	scanner := bufio.NewScanner(strings.NewReader(out))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		pkgName := fields[0]
		pkgVer := fields[1]

		// Get package size from pacman -Qi
		var sizeKB int64
		if qi, err := Exec(ctx, d.ex, "pacman", "-Qi", pkgName); err == nil {
			for _, qline := range strings.Split(qi, "\n") {
				if strings.HasPrefix(qline, "Installed Size") {
					parts := strings.SplitN(qline, ":", 2)
					if len(parts) == 2 {
						sizeKB = parseHumanSize(parts[1])
					}
					break
				}
			}
		}

		apps = append(apps, model.AppInfo{
			Name:          pkgName,
			Version:       pkgVer,
			Source:        "orphan",
			InstallSizeKB: sizeKB,
			Status:        "Installed",
		})
	}
	return apps, nil
}

func (d *orphanDetector) detectAptOrphans(ctx context.Context) ([]model.AppInfo, error) {
	out, err := Exec(ctx, d.ex, "apt-get", "-s", "autoremove")
	if err != nil || strings.TrimSpace(out) == "" {
		return nil, nil
	}
	var apps []model.AppInfo
	scanner := bufio.NewScanner(strings.NewReader(out))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "Remv ") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				ver := "-"
				if len(fields) >= 3 {
					ver = strings.Trim(fields[2], "[]()")
				}
				apps = append(apps, model.AppInfo{
					Name:    fields[1],
					Version: ver,
					Source:  "orphan",
					Status:  "Installed",
				})
			}
		}
	}
	return apps, nil
}

func (d *orphanDetector) detectDnfOrphans(ctx context.Context) ([]model.AppInfo, error) {
	out, err := Exec(ctx, d.ex, "dnf", "repoquery", "--unneeded", "-q")
	if err != nil || strings.TrimSpace(out) == "" {
		return nil, nil
	}
	var apps []model.AppInfo
	scanner := bufio.NewScanner(strings.NewReader(out))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		apps = append(apps, model.AppInfo{
			Name:    line,
			Version: "-",
			Source:  "orphan",
			Status:  "Installed",
		})
	}
	return apps, nil
}
