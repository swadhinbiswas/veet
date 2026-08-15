<div align="center">

<img src="assets/logo.svg" alt="VEET — Universal Linux App Uninstaller" width="620"/>

<br/><br/>

[![Go Version](https://img.shields.io/badge/Go-1.22+-00ADD8?style=flat-square&logo=go)](https://go.dev)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg?style=flat-square)](LICENSE)
[![Platform](https://img.shields.io/badge/Platform-Linux-FCC624?style=flat-square&logo=linux&logoColor=black)](#)
[![PRs Welcome](https://img.shields.io/badge/PRs-welcome-brightgreen.svg?style=flat-square)](CONTRIBUTING.md)

<p align="center">
  <b>Universal Linux App Uninstaller &amp; Deep-Clean Residual Purger</b><br/>
  <i>Completely remove applications, orphaned configs, hidden caches, system logs, and leftover dotfiles in one confirmed action.</i>
</p>

</div>

---

## 💡 Why VEET?

When you remove software using traditional Linux package managers (`pacman -R`, `apt remove`, `dnf remove`, `flatpak uninstall`, etc.):
- **Only the binaries are deleted** (`/usr/bin`, `/usr/share`).
- **User data is left behind forever**: `~/.config/<app>`, `~/.cache/<app>` *(often multiple gigabytes from Chromium/Electron apps)*, `~/.local/share/<app>`, `/var/log/<app>`, and lingering dotfiles.
- **Package managers are fragmented**: Your system software lives across system repos, AUR, Flatpak, Snap, AppImages, npm global, pipx, and Cargo.

**VEET bridges this gap** — giving Linux users a fast, unified, macOS *AppCleaner*-like experience built entirely in the terminal.

---

## ✨ Features

- **🌐 Multi-Source Detection**: Concurrently scans system package managers (`apt`, `dnf`, `pacman`, AUR via `yay`/`paru`, `zypper`), universal formats (`Flatpak`, `Snap`), portable bundles (`AppImage`), global toolchains (`npm -g`, `pipx`, `cargo`, `gem`, `go install`), and system services.
- **🍂 Orphan & Unneeded Dependencies**: Automatically discovers forgotten orphaned dependencies (`pacman -Qdt`, `apt autoremove`, `dnf repoquery --unneeded`) ready to be pruned.
- **📋 Systemd Journal & Log Vacuum**: Safely detects bloated `/var/log/journal` systemd logs and old `.gz` log archives to reclaim storage.
- **🔍 Deep Clean Removal**: Stages every candidate directory before execution (`~/.config`, `~/.cache`, `~/.local/share`, `/etc`, `/var/log`).
- **📊 1-Click Interactive Stat Cards**: Jump between `All Apps`, `Flatpak`, `Snap`, `Tools`, `Cache`, and `Reclaimable` with keys <kbd>1</kbd>–<kbd>6</kbd> or <kbd>[</kbd> / <kbd>]</kbd>.
- **⚡ Instant Bloat Sorting (<kbd>o</kbd>)**: Cycle sorting order (`Size ↓`, `Name A-Z`, `Source`, `Date ↓`) to find massive disk-hogging applications immediately.
- **💾 Real-Time Disk Gauge**: Live disk utilization monitoring displayed in the header.
- **🔒 Sudo Batching & Safety First**: Home-directory paths are removed as the current user. System paths are batched into **one single sudo prompt** per uninstall. Core system packages (`glibc`, `linux`, `systemd`, current shell) are protected and refused by default.
- **📄 Audit Trail & JSON Export**: Every removal is logged to `~/.local/share/veet/history.log`. Press <kbd>e</kbd> to export an inventory report to `apps-report.json`.

---

## 🚀 Installation

### Option 1: One-Line Bash Installer *(Recommended)*
Automatically compiles, installs to `~/.local/bin/veet`, and sets up shell completions:

```bash
curl -sSL https://raw.githubusercontent.com/swadhinbiswas/veet/main/install.sh | bash
```

*Or from local clone:*
```bash
./install.sh
```

---

### Option 2: Go Install
```bash
go install github.com/swadhinbiswas/veet@latest
# or from local clone:
go install .
```

---

### Option 3: Build from Source
```bash
git clone https://github.com/swadhinbiswas/veet.git
cd veet
make build
# or install directly
make install
```

---

### 🗑️ Uninstallation
To remove VEET and its shell completions:
```bash
./uninstall.sh
```

---

## 🎮 Usage

### Interactive TUI Mode *(Default)*
Launch the interactive terminal dashboard:
```bash
veet
```

### CLI & Scripting Modes
```bash
# Plain-text table of all installed applications across sources
veet scan

# Machine-readable JSON output (great for scripting and audits)
veet scan --json

# Non-interactive preview & deep-clean removal
veet clean <app-name>
veet clean <app-name> --source flatpak --yes
```

---

## ⌨️ TUI Keybindings Reference

| Key | Action |
|---|---|
| <kbd>1</kbd> – <kbd>6</kbd> | Jump to stat cards (`All`, `Flatpak`, `Snap`, `Tools`, `Cache`, `Reclaim`) |
| <kbd>[</kbd> / <kbd>]</kbd> or <kbd>←</kbd> / <kbd>→</kbd> | Cycle through top stat cards |
| <kbd>↑</kbd> <kbd>↓</kbd> or <kbd>j</kbd> <kbd>k</kbd> | Navigate the application list |
| <kbd>Space</kbd> | Toggle multi-select checkbox on current row |
| <kbd>a</kbd> | Select / deselect all visible filtered rows |
| <kbd>o</kbd> / <kbd>O</kbd> | Cycle sort order (`Size ↓`, `Name A-Z`, `Source`, `Date ↓`) |
| <kbd>p</kbd> / <kbd>P</kbd> | Toggle staged filesystem paths inspector in details panel |
| <kbd>/</kbd> | Focus fuzzy search-as-you-type filter |
| <kbd>Tab</kbd> / <kbd>Shift+Tab</kbd> | Cycle specific source filter (`pacman`, `flatpak`, `aur`, etc.) |
| <kbd>Enter</kbd> | Stage selected apps and open confirmation preview |
| <kbd>c</kbd> | 1-key quick purge for cache & residual dotfiles |
| <kbd>e</kbd> | Export inventory report to JSON (`~/.local/share/veet/apps-report.json`) |
| <kbd>r</kbd> | Rescan all detectors and package managers |
| <kbd>l</kbd> | View historical audit logs |
| <kbd>s</kbd> | Settings & protected components |
| <kbd>h</kbd> / <kbd>?</kbd> | Toggle full help modal |
| <kbd>Esc</kbd> | Back out of modal / unfocus search |
| <kbd>q</kbd> / <kbd>Ctrl+C</kbd> | Quit VEET |

---

## ⚙️ Configuration

Create an optional configuration file at `~/.config/veet/config.yaml`:

```yaml
protected_packages:
  - custom-kernel-module
  - my-critical-database
  - docker-engine
```

---

## 🏗️ Architecture

```
main.go               Single entrypoint & Cobra CLI commands (scan, clean, tui)
internal/model/       AppInfo, RemovableFiles, Config, SourceMeta
internal/detector/    Concurrent source detectors (apt, pacman, flatpak, appimage, orphans, ...)
internal/uninstaller/ Staging engine, sudo batching & filesystem cleanup (Afero)
internal/history/     Append-only audit logger (~/.local/share/veet/history.log)
internal/ui/          Bubble Tea & Lip Gloss TUI (table, details, stat cards, progress, styles)
```

---

## 🤝 Contributing

Contributions are welcome! Please check out [CONTRIBUTING.md](CONTRIBUTING.md) for instructions on setting up your development environment and adding support for new package managers or features.

---

## 📜 License

This project is open source and available under the [MIT License](LICENSE).
