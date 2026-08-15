# Contributing to VEET

First off, thank you for considering contributing to **VEET**! 

VEET is an open-source, community-driven tool dedicated to making software uninstallation and residual cleanup on Linux completely thorough, fast, and transparent.

Whether you want to add support for a new package manager, improve the TUI design, optimize detectors, or fix bugs, your help is welcome!

---

## Development Setup

### Prerequisites
- **Go 1.24+** installed ([golang.org](https://go.dev/dl/))
- Linux operating system (or Linux container/WSL)
- `git` and `make` (optional)

### Getting Started

1. **Fork and clone the repository**:
   ```bash
   git clone https://github.com/swadhinbiswas/veet.git
   cd veet
   ```

2. **Run tests**:
   ```bash
   go test -v ./...
   ```

3. **Build the single binary**:
   ```bash
   go build -o veet .
   # or
   make build
   ```

4. **Run the interactive TUI locally**:
   ```bash
   ./veet
   # or
   make run
   ```

---

## 🏗️ Codebase Architecture

```
veet/
├── main.go               # Single entrypoint & Cobra CLI commands (scan, clean, tui)
├── internal/
│   ├── model/            # Core data types: AppInfo, RemovableFiles, Config, SourceMeta
│   ├── detector/         # Parallel detection engine (1 file per package manager/source)
│   ├── uninstaller/      # Staging engine, sudo batching & filesystem cleanup (Afero)
│   ├── history/          # Append-only audit logger (~/.local/share/veet/history.log)
│   └── ui/               # Bubble Tea & Lip Gloss TUI (table, details, stat cards, styles)
├── Makefile              # Helper build & test targets
└── CONTRIBUTING.md       # Contribution guidelines
```

---

## 🔌 How to Add a New Package Manager or Source

Adding support for a new package manager or tool (e.g. `nix`, `guix`, `brew`, `gem`, `asdf`, etc.) is straightforward and modular.

### Step 1: Implement the `Detector` Interface

Create a new file under `internal/detector/<your-source>.go`:

```go
package detector

import (
	"context"
	"veet/internal/model"
)

type myToolDetector struct {
	ex Execer
}

func NewMyToolDetector(ex Execer) Detector {
	return &myToolDetector{ex: ex}
}

// Source returns the unique identifier string
func (d *myToolDetector) Source() string {
	return "mytool"
}

// Available checks if the CLI tool exists on the system
func (d *myToolDetector) Available() bool {
	_, err := d.ex.LookPath("mytool")
	return err == nil
}

// Detect scans installed packages and returns them
func (d *myToolDetector) Detect(ctx context.Context) ([]model.AppInfo, error) {
	out, err := Exec(ctx, d.ex, "mytool", "list")
	if err != nil {
		return nil, err
	}
	
	var apps []model.AppInfo
	// Parse output lines into model.AppInfo structs...
	return apps, nil
}
```

### Step 2: Register in `internal/detector/registry.go`

Add your detector to `Registry()`:

```go
func Registry(ex Execer, home string) []Detector {
	return []Detector{
		// ...
		NewMyToolDetector(ex),
		// ...
	}
}
```

### Step 3: Add Display Metadata in `internal/model/app.go`

Define your source's badge label, icon, and brand color in `SourceMetaByID`:

```go
var SourceMetaByID = map[string]SourceMeta{
	// ...
	"mytool": {Label: "MyTool", Icon: "❄️", Color: "#00E5FF"},
}
```

### Step 4: Add Uninstall Command in `internal/uninstaller/uninstaller.go`

Define how the package manager removes the application in `RemoveCommand()`:

```go
switch app.Source {
// ...
case "mytool":
	return []string{"mytool", "uninstall", "-y", app.Name}
}
```

---

## 🧪 Testing Guidelines

We prioritize high test coverage and rock-solid safety to ensure no user files are deleted accidentally.

- Run all unit tests:
  ```bash
  go test -v ./...
  ```
- File deletion runs through [afero](https://github.com/spf13/afero), allowing mock testing against `afero.NewMemMapFs()` without touching your real disk.
- Command execution is abstracted through the `Execer` interface for mock testing.

---

## 📝 Pull Request Checklist

Before submitting your pull request:
- [ ] All tests pass (`go test ./...`).
- [ ] Code is formatted (`go fmt ./...`).
- [ ] Added unit tests for new features or bug fixes.
- [ ] Kept commit messages clear and descriptive.
