<div align="center">

<img src="assets/logo.svg" alt="VEET — Universal Linux App Uninstaller" width="600"/>

<br/>

[![CI](https://github.com/swadhinbiswas/veet/actions/workflows/ci.yml/badge.svg)](https://github.com/swadhinbiswas/veet/actions/workflows/ci.yml)
[![Go Version](https://img.shields.io/badge/Go-1.22+-00ADD8?style=flat-square&logo=go)](https://go.dev)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg?style=flat-square)](LICENSE)
[![Platform](https://img.shields.io/badge/Platform-Linux-FCC624?style=flat-square&logo=linux&logoColor=black)](#)
[![PRs Welcome](https://img.shields.io/badge/PRs-welcome-brightgreen.svg?style=flat-square)](CONTRIBUTING.md)

<p align="center">
  <b>Universal Linux Application Uninstaller &amp; Deep-Clean Residual Purger</b><br/>
  <i>An interactive terminal system utility that detects installed software across package managers and automates complete removal of binaries, user caches, configuration files, and residual logs in one confirmed action.</i>
</p>

</div>

---

## Technical Overview

When software is uninstalled using conventional Linux package managers (`pacman -R`, `apt remove`, `dnf remove`, `flatpak uninstall`), only package-managed system files (`/usr/bin`, `/usr/share`) are modified. User-space directories remain unmanaged:

- `~/.config/<app>` *(user configuration state)*
- `~/.cache/<app>` *(accumulated runtime caches, shader caches, webview stores)*
- `~/.local/share/<app>` *(local application data and databases)*
- `~/.local/state/<app>` *(session and state histories)*
- `/var/log/<app>` and systemd journal archives

Over extended usage, unmanaged leftovers consume significant disk storage. Modern Linux distributions also feature package manager fragmentation across native repositories, universal formats, and language toolchains.

**VEET** bridges this gap by scanning all installed targets concurrently, staging candidate residual paths for user verification, and executing atomic uninstalls with batched privilege elevation.

---

## Comparison Matrix

| Feature | `pacman -R` / `apt remove` | `flatpak uninstall` | `bleachbit` | `veet` |
|:---|:---:|:---:|:---:|:---:|
| **Binary Removal** | Yes | Yes | No | **Yes** |
| **User Configuration Cleanup (`~/.config`)** | No | No | Partial | **Yes (Inspected &amp; Staged)** |
| **User Cache Cleanup (`~/.cache`)** | No | No | Yes | **Yes (Inspected &amp; Staged)** |
| **Universal Multi-Source Aggregator** | No | No | No | **Yes (15+ package managers)** |
| **Orphaned Dependencies Detection** | Manual query | No | No | **Yes (Live detection)** |
| **AppImage &amp; Portable Binary Scanner** | No | No | No | **Yes (Auto-discovered)** |
| **Interactive Terminal UI (TUI)** | No | No | GTK Only | **Yes (Zero-latency Bubble Tea)** |
| **Pre-execution Dry-run Inspection** | Partial | Partial | Yes | **Yes (Full path inspection)** |
| **Batched Sudo Elevation** | Per command | N/A | Full root required | **Yes (Single prompt per batch)** |

---

## Core Capabilities

<table>
  <tr>
    <td width="50%" valign="top">
      <h4><img src="assets/icons/sources.svg" width="18" height="18" align="center" /> Multi-Source Parallel Detection</h4>
      <p>Concurrently queries system package managers (<code>pacman</code>/AUR, <code>apt</code>, <code>dnf</code>, <code>zypper</code>), sandboxed formats (<code>Flatpak</code>, <code>Snap</code>), portable bundles (<code>AppImage</code>), and language toolchains (<code>npm -g</code>, <code>pipx</code>, <code>cargo</code>, <code>gem</code>, <code>go install</code>) with zero-cost skipping for absent tools.</p>
    </td>
    <td width="50%" valign="top">
      <h4><img src="assets/icons/purge.svg" width="18" height="18" align="center" /> Deep Residual Cleanup</h4>
      <p>Automates residual discovery across user configurations, cached runtime stores, desktop integration entries, icons, and system paths, staging each candidate before disk execution.</p>
    </td>
  </tr>
  <tr>
    <td width="50%" valign="top">
      <h4><img src="assets/icons/orphan.svg" width="18" height="18" align="center" /> Orphaned Dependency Pruning</h4>
      <p>Live detection of unneeded dependency packages across native package managers (<code>pacman -Qdt</code>, <code>apt autoremove</code>, <code>dnf repoquery --unneeded</code>) ready for safe reclamation.</p>
    </td>
    <td width="50%" valign="top">
      <h4><img src="assets/icons/journal.svg" width="18" height="18" align="center" /> System Log &amp; Journal Vacuum</h4>
      <p>Identifies bloated systemd journal storage (<code>/var/log/journal</code>) and rotated compressed log archives (<code>/var/log/*.gz</code>) with integrated elevated vacuum routines.</p>
    </td>
  </tr>
  <tr>
    <td width="50%" valign="top">
      <h4><img src="assets/icons/terminal.svg" width="18" height="18" align="center" /> Interactive Navigation &amp; Telemetry</h4>
      <p>Instant category switching (<kbd>1</kbd>–<kbd>6</kbd>), four-mode list sorting (<kbd>o</kbd>) for rapid disk bloat identification, and in-place staged path inspection (<kbd>p</kbd>).</p>
    </td>
    <td width="50%" valign="top">
      <h4><img src="assets/icons/security.svg" width="18" height="18" align="center" /> Sudo Batching &amp; Anti-Brick Safety</h4>
      <p>Core system components (<code>glibc</code>, <code>linux</code>, <code>systemd</code>, active shell) are protected from selection. System paths are batched into a single elevated transaction.</p>
    </td>
  </tr>
</table>

---

## Installation

### Option 1: Automated Script Installation (Recommended)

Downloads, compiles with binary optimizations (`-ldflags="-s -w"`), installs to `~/.local/bin/veet`, and configures shell autocompletions:

```bash
curl -sSL https://raw.githubusercontent.com/swadhinbiswas/veet/main/install.sh | bash
```

*Or from local clone:*
```bash
./install.sh
```

---

### Option 2: Go Package Manager

Install directly via the standard Go toolchain (requires Go 1.22+):

```bash
go install github.com/swadhinbiswas/veet@latest
```

*Or install from local source:*
```bash
go install .
```

---

### Option 3: Manual Compilation via Makefile

```bash
git clone https://github.com/swadhinbiswas/veet.git
cd veet
make build
make install
```

> [!NOTE]
> Ensure `~/.local/bin` is present in your environment `$PATH`. If not, add `export PATH="$PATH:$HOME/.local/bin"` to your `~/.bashrc` or `~/.zshrc`.

---

## Usage

### Interactive Terminal Dashboard

Launch the zero-latency terminal user interface:

```bash
veet
```

---

### Command Line Interface

VEET exposes non-interactive headless commands for automation, scripting, and CI auditing:

```bash
# Output installed application inventory table to stdout
veet scan

# Export machine-readable JSON inventory
veet scan --json

# Non-interactive deep-clean removal with preview
veet clean <package-name>

# Non-interactive removal scoped to specific source with confirmation
veet clean <package-name> --source flatpak --yes
```

---

## Keybinding Reference

<table width="100%">
  <thead>
    <tr>
      <th width="25%">Keybinding</th>
      <th width="75%">Action</th>
    </tr>
  </thead>
  <tbody>
    <tr>
      <td><kbd>1</kbd> – <kbd>6</kbd></td>
      <td>Jump to stat card category (<code>All</code>, <code>Flatpak</code>, <code>Snap</code>, <code>Tools</code>, <code>Cache</code>, <code>Reclaim</code>)</td>
    </tr>
    <tr>
      <td><kbd>[</kbd> / <kbd>]</kbd> or <kbd>&larr;</kbd> / <kbd>&rarr;</kbd></td>
      <td>Cycle horizontally through top stat categories</td>
    </tr>
    <tr>
      <td><kbd>&uarr;</kbd> <kbd>&darr;</kbd> / <kbd>j</kbd> <kbd>k</kbd></td>
      <td>Navigate table selection cursor</td>
    </tr>
    <tr>
      <td><kbd>Space</kbd></td>
      <td>Toggle multi-select checkbox on active row</td>
    </tr>
    <tr>
      <td><kbd>a</kbd></td>
      <td>Select / deselect all visible filtered rows (skips protected components)</td>
    </tr>
    <tr>
      <td><kbd>o</kbd> / <kbd>O</kbd></td>
      <td>Cycle sort order (<code>Size &darr;</code>, <code>Name A-Z</code>, <code>Source</code>, <code>Date &darr;</code>)</td>
    </tr>
    <tr>
      <td><kbd>p</kbd> / <kbd>P</kbd></td>
      <td>Toggle staged filesystem paths inspector in details panel</td>
    </tr>
    <tr>
      <td><kbd>/</kbd></td>
      <td>Focus fuzzy search-as-you-type filter input</td>
    </tr>
    <tr>
      <td><kbd>Tab</kbd> / <kbd>Shift+Tab</kbd></td>
      <td>Cycle specific source filter dropdown (<code>pacman</code>, <code>aur</code>, <code>flatpak</code>, etc.)</td>
    </tr>
    <tr>
      <td><kbd>Enter</kbd></td>
      <td>Stage selected applications and open confirmation preview modal</td>
    </tr>
    <tr>
      <td><kbd>c</kbd></td>
      <td>1-key quick purge for user cache &amp; residual files</td>
    </tr>
    <tr>
      <td><kbd>e</kbd></td>
      <td>Export audit inventory report to <code>~/.local/share/veet/apps-report.json</code></td>
    </tr>
    <tr>
      <td><kbd>r</kbd></td>
      <td>Trigger background rescan across all detectors</td>
    </tr>
    <tr>
      <td><kbd>l</kbd></td>
      <td>Open history audit log viewport</td>
    </tr>
    <tr>
      <td><kbd>s</kbd></td>
      <td>Open settings and protected components registry</td>
    </tr>
    <tr>
      <td><kbd>h</kbd> / <kbd>?</kbd></td>
      <td>Toggle keybinding reference overlay</td>
    </tr>
    <tr>
      <td><kbd>Esc</kbd></td>
      <td>Dismiss active modal / blur search input</td>
    </tr>
    <tr>
      <td><kbd>q</kbd> / <kbd>Ctrl+C</kbd></td>
      <td>Quit VEET</td>
    </tr>
  </tbody>
</table>

---

## Configuration

<details>
<summary><b>Custom Configuration (<code>~/.config/veet/config.yaml</code>)</b></summary>
<br/>

VEET supports optional user configuration via YAML. Create `~/.config/veet/config.yaml` to declare custom protected packages:

```yaml
# Packages that VEET will refuse to stage or select for deletion
protected_packages:
  - custom-kernel-module
  - my-critical-service
  - proprietary-driver
```

</details>

<details>
<summary><b>System Architecture &amp; Module Map</b></summary>
<br/>

```
veet/
├── main.go               # Cobra CLI entrypoint (scan, clean, tui)
├── internal/
│   ├── model/            # Data types: AppInfo, RemovableFiles, Config, SourceMeta
│   ├── detector/         # Parallel detector engine (15+ source handlers)
│   ├── uninstaller/      # Staging engine, sudo batching & filesystem removal (Afero)
│   ├── history/          # Append-only audit logger (~/.local/share/veet/history.log)
│   └── ui/               # Bubble Tea & Lip Gloss TUI components
├── assets/               # Brand assets & custom vector icons
├── Makefile              # Build automation targets
├── install.sh            # Automated Bash installer
└── uninstall.sh          # Uninstaller script
```

</details>

---

## Uninstallation

To remove the compiled binary, shell autocompletions, and optionally purge audit logs:

```bash
./uninstall.sh
```

---

## Contributing

Contributions are welcome. Please refer to [CONTRIBUTING.md](CONTRIBUTING.md) for local development setup, detector implementation guides, and testing protocols.

---

## License

VEET is open-source software licensed under the [MIT License](LICENSE).
