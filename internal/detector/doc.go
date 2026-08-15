// Package detector provides concurrent discovery engines for installed Linux applications.
//
// It queries system package managers (pacman, AUR, apt, dnf, zypper), universal application
// sandboxes (Flatpak, Snap), portable application bundles (AppImage), orphaned package queries,
// system logs / journal storage, and global language toolchains (npm, pipx, cargo, gem, go install).
package detector
