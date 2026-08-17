package detector

import (
	"context"
	"sort"

	"github.com/swadhinbiswas/veet/internal/model"
)

// Registry builds the ordered list of detectors for the current system.
func Registry(ex Execer, home string) []Detector {
	return []Detector{
		NewAptDetector(ex),
		NewDnfDetector(ex),
		NewPacmanDetector(ex),
		NewZypperDetector(ex),
		NewFlatpakDetector(ex),
		NewSnapDetector(ex),
		NewAppImageDetector(home),
		NewBrewDetector(ex),
		NewNixDetector(ex),
		NewOrphanDetector(ex),
		NewNpmDetector(ex),
		NewPipxDetector(ex),
		NewCargoDetector(ex),
		NewGemDetector(ex),
		NewGoInstallDetector(ex),
		NewCacheDetector(home),
		NewSystemLogsDetector(),
	}
}

// Scan runs every detector concurrently and merges results.
// It returns all apps, skipped sources and any non-fatal errors.
func Scan(ctx context.Context, ex Execer, home string) ([]model.AppInfo, []string, []error) {
	return ScanStreaming(ctx, ex, home, nil)
}

// ScanStreaming runs every detector concurrently and streams progress events.
func ScanStreaming(ctx context.Context, ex Execer, home string, onProgress func(ProgressEvent)) ([]model.AppInfo, []string, []error) {
	results := RunAllStreaming(ctx, ex, Registry(ex, home), onProgress)
	var apps []model.AppInfo
	var skipped []string
	var errs []error
	for _, r := range results {
		if r.Skipped {
			skipped = append(skipped, r.Source+" ("+r.Reason+")")
			continue
		}
		if r.Err != nil {
			errs = append(errs, r.Err)
			skipped = append(skipped, r.Source+": "+r.Err.Error())
			continue
		}
		apps = append(apps, r.Apps...)
	}
	sort.Slice(apps, func(i, j int) bool {
		if apps[i].Source != apps[j].Source {
			return apps[i].Source < apps[j].Source
		}
		return apps[i].Name < apps[j].Name
	})
	return apps, skipped, errs
}
