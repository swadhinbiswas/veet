package detector

import (
	"context"
	"strings"

	"github.com/swadhinbiswas/veet/internal/model"
)

// pacmanDetector lists explicitly installed repo packages (pacman -Qe)
// plus foreign/AUR packages (pacman -Qm, installed via yay/paru).
type pacmanDetector struct {
	ex      Execer
	pacman  string
	aurTool string
}

// NewPacmanDetector detects Arch-based packages.
func NewPacmanDetector(ex Execer) Detector {
	pm, _ := ex.LookPath("pacman")
	aur := ""
	if p, err := ex.LookPath("yay"); err == nil {
		aur = p
	} else if p, err := ex.LookPath("paru"); err == nil {
		aur = p
	}
	return &pacmanDetector{ex: ex, pacman: pm, aurTool: aur}
}

func (d *pacmanDetector) Source() string  { return "pacman" }
func (d *pacmanDetector) Available() bool { return d.pacman != "" }

func (d *pacmanDetector) Detect(ctx context.Context) ([]model.AppInfo, error) {
	out, err := Exec(ctx, d.ex, "pacman", "-Qe")
	if err != nil {
		return nil, err
	}
	names := map[string]string{} // name -> version (union of -Qe and -Qm)
	aurSet := map[string]bool{}
	for _, line := range strings.Split(out, "\n") {
		fields := strings.Fields(line)
		if len(fields) >= 2 {
			names[fields[0]] = fields[1]
		}
	}

	// Foreign (AUR) packages may also appear in -Qe; union them under "aur".
	if d.aurTool != "" {
		if foreign, err := Exec(ctx, d.ex, "pacman", "-Qm"); err == nil {
			for _, line := range strings.Split(foreign, "\n") {
				fields := strings.Fields(line)
				if len(fields) >= 2 {
					names[fields[0]] = fields[1]
					aurSet[fields[0]] = true
				}
			}
		}
	}

	allKeys := keysOf(names)
	if len(allKeys) == 0 {
		return nil, nil
	}

	// Try bulk pacman -Qi first (single process, ~50x faster)
	if bulkInfo, err := Exec(ctx, d.ex, "pacman", "-Qi"); err == nil && strings.Contains(bulkInfo, "Name") {
		apps := parsePacmanQiBulk(bulkInfo, names, aurSet)
		if len(apps) > 0 {
			return apps, nil
		}
	}

	sink := &appSink{}
	workerPool(allKeys, 8, func(name string) {
		var sizeKB int64
		var installed string
		var deps []string
		var url, maint string
		if info, err := Exec(ctx, d.ex, "pacman", "-Qi", name); err == nil {
			for _, line := range strings.Split(info, "\n") {
				key, val, ok := strings.Cut(line, ":")
				if !ok {
					continue
				}
				key, val = strings.TrimSpace(key), strings.TrimSpace(val)
				switch key {
				case "Installed Size":
					sizeKB = parseHumanSize(val)
				case "Install Date":
					installed = val
				case "Depends On":
					if val != "None" {
						for _, dep := range strings.Split(val, " ") {
							if dep != "" {
								deps = append(deps, dep)
							}
						}
					}
				case "URL":
					url = val
				case "Packager":
					maint = val
				}
			}
		}
		source := "pacman"
		if aurSet[name] {
			source = "aur"
		}
		sink.add(model.AppInfo{
			Name:          name,
			Version:       names[name],
			Source:        source,
			InstalledOn:   parseDate(installed),
			InstallSizeKB: sizeKB,
			Status:        "Installed",
			Dependencies:  deps,
			Maintainer:    maint,
			Homepage:      url,
		})
	})
	return sink.apps, nil
}

func parsePacmanQiBulk(output string, filterNames map[string]string, aurSet map[string]bool) []model.AppInfo {
	var apps []model.AppInfo
	entries := strings.Split(output, "\n\n")
	for _, entry := range entries {
		var name, ver, installed, url, maint string
		var sizeKB int64
		var deps []string
		for _, line := range strings.Split(entry, "\n") {
			key, val, ok := strings.Cut(line, ":")
			if !ok {
				continue
			}
			key, val = strings.TrimSpace(key), strings.TrimSpace(val)
			switch key {
			case "Name":
				name = val
			case "Version":
				ver = val
			case "Installed Size":
				sizeKB = parseHumanSize(val)
			case "Install Date":
				installed = val
			case "Depends On":
				if val != "None" {
					for _, dep := range strings.Split(val, " ") {
						if dep != "" {
							deps = append(deps, dep)
						}
					}
				}
			case "URL":
				url = val
			case "Packager":
				maint = val
			}
		}
		if name == "" {
			continue
		}
		if _, needed := filterNames[name]; !needed {
			continue
		}
		if ver == "" {
			ver = filterNames[name]
		}
		source := "pacman"
		if aurSet[name] {
			source = "aur"
		}
		apps = append(apps, model.AppInfo{
			Name:          name,
			Version:       ver,
			Source:        source,
			InstalledOn:   parseDate(installed),
			InstallSizeKB: sizeKB,
			Status:        "Installed",
			Dependencies:  deps,
			Maintainer:    maint,
			Homepage:      url,
		})
	}
	return apps
}

func keysOf(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
