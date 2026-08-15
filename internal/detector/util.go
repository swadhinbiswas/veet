package detector

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/swadhinbiswas/veet/internal/model"
)

// dirSizeKB returns the on-disk size of a path in KiB (du-style).
// It works on a plain OS filesystem (used for caches, gem dirs, go bins).
func dirSizeKB(path string) (int64, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return 0, err
	}
	if !info.IsDir() {
		return (info.Size() + 1023) / 1024, nil
	}
	var total int64
	filepath.Walk(path, func(_ string, fi os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !fi.IsDir() {
			total += (fi.Size() + 1023) / 1024
		}
		return nil
	})
	return total, nil
}

// workerPool runs fn across items with bounded concurrency.
func workerPool[T any](items []T, workers int, fn func(T)) {
	if len(items) == 0 {
		return
	}
	var wg sync.WaitGroup
	ch := make(chan T)
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for it := range ch {
				fn(it)
			}
		}()
	}
	for _, it := range items {
		ch <- it
	}
	close(ch)
	wg.Wait()
}

// parseHumanSize converts "52.3 MiB", "1,024 KB", "512 MB" style strings to KB.
func parseHumanSize(s string) int64 {
	s = strings.TrimSpace(s)
	// "1,024" is a thousands separator; "52,3" is a decimal comma.
	if strings.Count(s, ",") == 1 {
		parts := strings.Split(s, ",")
		if len(parts[1]) > 1 {
			s = strings.ReplaceAll(s, ",", "")
		} else {
			s = strings.ReplaceAll(s, ",", ".")
		}
	}
	fields := strings.Fields(s)
	if len(fields) == 0 {
		return 0
	}
	num, err := strconv.ParseFloat(fields[0], 64)
	if err != nil {
		return 0
	}
	unit := strings.ToUpper(fields[0])
	if len(fields) > 1 {
		unit = strings.ToUpper(fields[1])
	}
	var kb int64
	switch {
	case strings.Contains(unit, "G"): // GB, GiB, GIO
		kb = int64(num * 1024 * 1024)
	case strings.Contains(unit, "M"): // MB, MiB, MIO
		kb = int64(num * 1024)
	case strings.Contains(unit, "K"): // KB, KiB, KIO
		kb = int64(num)
	default: // bytes
		kb = int64(num) / 1024
	}
	if kb == 0 && num > 0 {
		kb = 1
	}
	return kb
}

// parseDate converts common package-manager date strings to time.Time.
func parseDate(s string) time.Time {
	s = strings.TrimSpace(s)
	if s == "" || s == "-" || s == "None" || s == "unknown" {
		return time.Time{}
	}

	// Normalize spaces
	s = strings.Join(strings.Fields(s), " ")

	layouts := []string{
		"Mon 02 Jan 2006 03:04:05 PM MST",
		"Mon 02 Jan 2006 03:04:05 PM -0700",
		"Mon 02 Jan 2006 03:04:05 PM -07",
		"Mon 02 Jan 2006 03:04:05 PM +06",
		"Mon 02 Jan 2006 03:04:05 PM",
		"Mon 02 Jan 2006 01:04:05 PM MST",
		"Mon 02 Jan 2006 01:04:05 PM -0700",
		"Mon 02 Jan 2006 01:04:05 PM +06",
		"Mon 02 Jan 2006 01:04:05 PM",
		"Mon 02 Jan 2006 15:04:05 MST",
		"Mon 02 Jan 2006 15:04:05 -0700",
		"Mon 02 Jan 2006 15:04:05 -07",
		"Mon 02 Jan 2006 15:04:05 +06",
		"Mon 02 Jan 2006 15:04:05",
		"Mon Jan 2 15:04:05 2006",
		"Mon Jan _2 15:04:05 2006",
		"2006-01-02 15:04:05 -0700",
		"2006-01-02 15:04:05 MST",
		"2006-01-02 15:04:05",
		"2006-01-02 15:04",
		"2006-01-02T15:04:05Z07:00",
		"2006-01-02T15:04:05-0700",
		"2006-01-02T15:04:05",
		"2006-01-02",
		"02/01/2006 15:04:05",
		"02/01/2006",
		"01/02/2006 15:04:05",
		"01/02/2006",
		time.RFC1123,
		time.RFC1123Z,
		time.RFC822,
		time.RFC822Z,
		time.RFC3339,
		time.ANSIC,
		time.UnixDate,
		time.RubyDate,
	}

	for _, layout := range layouts {
		if t, err := time.Parse(layout, s); err == nil {
			return t
		}
	}
	return time.Time{}
}

// installedApps guards a slice of AppInfo against races (worker-pool fills).
type appSink struct {
	mu   sync.Mutex
	apps []model.AppInfo
}

func (s *appSink) add(a model.AppInfo) {
	s.mu.Lock()
	s.apps = append(s.apps, a)
	s.mu.Unlock()
}
