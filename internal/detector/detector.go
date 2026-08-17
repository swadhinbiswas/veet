package detector

import (
	"bytes"
	"context"
	"os/exec"
	"sync"

	"github.com/swadhinbiswas/veet/internal/model"
)

// Cmd abstracts an exec.Cmd for output capture.
type Cmd interface {
	SetStdout(b *bytes.Buffer)
	SetStderr(b *bytes.Buffer)
	Run() error
}

// Execer abstracts process execution so detectors are testable.
type Execer interface {
	LookPath(name string) (string, error)
	CommandContext(ctx context.Context, name string, args ...string) Cmd
}

// RealExec wraps os/exec.
type RealExec struct{}

func (RealExec) LookPath(name string) (string, error) { return exec.LookPath(name) }

func (RealExec) CommandContext(ctx context.Context, name string, args ...string) Cmd {
	return &realCmd{cmd: exec.CommandContext(ctx, name, args...)}
}

type realCmd struct {
	cmd    *exec.Cmd
	stdout *bytes.Buffer
	stderr *bytes.Buffer
}

func (c *realCmd) SetStdout(b *bytes.Buffer) { c.cmd.Stdout = b }
func (c *realCmd) SetStderr(b *bytes.Buffer) { c.cmd.Stderr = b }
func (c *realCmd) Run() error                { return c.cmd.Run() }

// Detector discovers installed applications from one source.
type Detector interface {
	// Source returns the canonical source id ("apt", "pacman", ...).
	Source() string
	// Available reports whether the backing tool is installed.
	Available() bool
	// Detect returns installed apps from this source.
	Detect(ctx context.Context) ([]model.AppInfo, error)
}

// Runner reports a detector's outcome.
type Runner struct {
	Apps    []model.AppInfo
	Source  string
	Skipped bool   // tool not present, nothing detected
	Reason  string // why it was skipped
	Err     error
}

// Exec runs a command synchronously, returning trimmed stdout.
func Exec(ctx context.Context, e Execer, name string, args ...string) (string, error) {
	cmd := e.CommandContext(ctx, name, args...)
	var out bytes.Buffer
	cmd.SetStdout(&out)
	cmd.SetStderr(&bytes.Buffer{})
	if err := cmd.Run(); err != nil {
		return "", err
	}
	return out.String(), nil
}

// ProgressEvent describes a detector's state change during scan.
type ProgressEvent struct {
	Index  int
	Source string
	Status string // "starting", "done", "skipped", "error"
	Count  int
	Total  int
	Err    error
}

// RunAll executes every detector concurrently and reports results.
func RunAll(ctx context.Context, e Execer, detectors []Detector) []Runner {
	return RunAllStreaming(ctx, e, detectors, nil)
}

// RunAllStreaming executes every detector concurrently and streams ProgressEvents to onProgress.
func RunAllStreaming(ctx context.Context, e Execer, detectors []Detector, onProgress func(ProgressEvent)) []Runner {
	results := make([]Runner, len(detectors))
	var wg sync.WaitGroup
	total := len(detectors)
	for i, d := range detectors {
		wg.Add(1)
		go func(i int, d Detector) {
			defer wg.Done()
			src := d.Source()
			if onProgress != nil {
				onProgress(ProgressEvent{Index: i, Source: src, Status: "starting", Total: total})
			}
			res := Runner{Source: src}
			if !d.Available() {
				res.Skipped = true
				res.Reason = "not installed"
				results[i] = res
				if onProgress != nil {
					onProgress(ProgressEvent{Index: i, Source: src, Status: "skipped", Total: total})
				}
				return
			}
			apps, err := d.Detect(ctx)
			res.Apps = apps
			res.Err = err
			results[i] = res
			if onProgress != nil {
				status := "done"
				if err != nil {
					status = "error"
				}
				onProgress(ProgressEvent{Index: i, Source: src, Status: status, Count: len(apps), Total: total, Err: err})
			}
		}(i, d)
	}
	wg.Wait()
	return results
}
