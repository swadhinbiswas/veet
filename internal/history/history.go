package history

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/spf13/afero"
)

// Entry is one audit-trail line.
type Entry struct {
	Time    time.Time
	App     string
	Source  string
	Status  string // "ok", "failed"
	FreedKB int64
	Files   int
	Detail  string
}

// Log appends and reads ~/.local/share/veet/history.log via afero.
type Log struct {
	fs   afero.Fs
	path string
}

// New creates a history log rooted at the given path.
func New(fs afero.Fs, path string) *Log {
	return &Log{fs: fs, path: path}
}

// Path returns the history file location.
func (l *Log) Path() string { return l.path }

// DefaultPath returns the standard history file location.
func DefaultPath(dataDir string) string {
	return dataDir + "/history.log"
}

// Append writes one timestamped line. Never fails silently on the caller:
// errors are returned and surfaced in the UI activity log.
func (l *Log) Append(e Entry) error {
	line := fmt.Sprintf("%s\t%s\t%s\t%s\t%d\t%d\t%s\n",
		e.Time.Format(time.RFC3339), e.App, e.Source, e.Status,
		e.FreedKB, e.Files, strings.ReplaceAll(e.Detail, "\t", " "))
	f, err := l.fs.OpenFile(l.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.WriteString(line)
	return err
}

// Read returns all entries, most recent first.
func (l *Log) Read() ([]Entry, error) {
	data, err := afero.ReadFile(l.fs, l.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var entries []Entry
	for _, line := range strings.Split(string(data), "\n") {
		parts := strings.Split(line, "\t")
		if len(parts) < 6 {
			continue
		}
		t, err := time.Parse(time.RFC3339, parts[0])
		if err != nil {
			continue
		}
		var freed int64
		var files int
		fmt.Sscanf(parts[4], "%d", &freed)
		fmt.Sscanf(parts[5], "%d", &files)
		e := Entry{
			Time: t, App: parts[1], Source: parts[2], Status: parts[3],
			Files: files, FreedKB: freed,
		}
		if len(parts) > 6 {
			e.Detail = parts[6]
		}
		entries = append(entries, e)
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Time.After(entries[j].Time) })
	return entries, nil
}
