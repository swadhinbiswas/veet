package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"

	"github.com/swadhinbiswas/veet/internal/detector"
	"github.com/swadhinbiswas/veet/internal/history"
	"github.com/swadhinbiswas/veet/internal/model"
	"github.com/swadhinbiswas/veet/internal/ui"
	"github.com/swadhinbiswas/veet/internal/uninstaller"
)

var rootCmd = &cobra.Command{
	Use:   "veet",
	Short: "Universal Linux App Uninstaller & Deep-Clean Residual Purger",
	Long: `VEET finds and completely uninstalls applications from every
common source — apt, dnf, pacman/AUR, zypper, flatpak, snap, appimage,
brew, nix, npm, pipx, cargo, gem and go install — removing configs, caches, logs
and residuals in one confirmed action.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runTUI()
	},
}

var scanCmd = &cobra.Command{
	Use:   "scan",
	Short: "List installed applications across all package managers",
	Long:  "Scan all package managers and print a table (or JSON with --json).",
	RunE: func(cmd *cobra.Command, args []string) error {
		asJSON, _ := cmd.Flags().GetBool("json")
		apps, skipped, _ := scanApps()
		for _, s := range skipped {
			fmt.Fprintln(os.Stderr, "skipped:", s)
		}
		if asJSON {
			return printJSON(apps)
		}
		printTable(apps)
		return nil
	},
}

var cleanCmd = &cobra.Command{
	Use:   "clean <app> [app2...]",
	Short: "Deep-clean uninstall one or more applications",
	Long: `Uninstall <app> and remove its configs, caches, logs and residuals.
Shows a preview of all staged paths before execution.
Pass --yes (-y) to proceed non-interactively, or --dry-run to preview only.`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		source, _ := cmd.Flags().GetString("source")
		yes, _ := cmd.Flags().GetBool("yes")
		dryRun, _ := cmd.Flags().GetBool("dry-run")
		return runClean(args, source, yes, dryRun)
	},
}

var orphansCmd = &cobra.Command{
	Use:   "orphans",
	Short: "Detect and clean orphaned packages and unneeded dependencies",
	RunE: func(cmd *cobra.Command, args []string) error {
		asJSON, _ := cmd.Flags().GetBool("json")
		clean, _ := cmd.Flags().GetBool("clean")
		yes, _ := cmd.Flags().GetBool("yes")

		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()
		orphanDet := detector.NewOrphanDetector(detector.RealExec{})
		apps, err := orphanDet.Detect(ctx)
		if err != nil {
			return err
		}
		if asJSON {
			return printJSON(apps)
		}
		if len(apps) == 0 {
			fmt.Println("No orphaned packages or unneeded dependencies found.")
			return nil
		}
		fmt.Printf("Found %d orphaned package(s):\n", len(apps))
		printTable(apps)
		if !clean {
			fmt.Println("\nTip: Run with --clean (or veet clean <app>) to remove these orphaned dependencies.")
			return nil
		}
		var names []string
		for _, a := range apps {
			names = append(names, a.Name)
		}
		return runClean(names, "orphan", yes, false)
	},
}

var cacheCmd = &cobra.Command{
	Use:   "cache",
	Short: "Inspect and clean reclaimable user caches and leftover directories",
	RunE: func(cmd *cobra.Command, args []string) error {
		asJSON, _ := cmd.Flags().GetBool("json")
		clean, _ := cmd.Flags().GetBool("clean")
		yes, _ := cmd.Flags().GetBool("yes")

		cfg := model.DefaultConfig()
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()
		cacheDet := detector.NewCacheDetector(cfg.Home)
		apps, err := cacheDet.Detect(ctx)
		if err != nil {
			return err
		}
		if asJSON {
			return printJSON(apps)
		}
		if len(apps) == 0 {
			fmt.Println("No reclaimable cache or leftover directories found.")
			return nil
		}
		var totalKB int64
		for _, a := range apps {
			totalKB += a.InstallSizeKB
		}
		fmt.Printf("Found %d cache/leftover targets (%s total):\n", len(apps), model.HumanSize(totalKB))
		printTable(apps)
		if !clean {
			fmt.Println("\nTip: Run with --clean to remove these cache entries.")
			return nil
		}
		var names []string
		for _, a := range apps {
			names = append(names, a.Name)
		}
		return runClean(names, "cache", yes, false)
	},
}

var historyCmd = &cobra.Command{
	Use:   "history",
	Short: "View or clear uninstallation audit log history",
	RunE: func(cmd *cobra.Command, args []string) error {
		clear, _ := cmd.Flags().GetBool("clear")
		asJSON, _ := cmd.Flags().GetBool("json")
		limit, _ := cmd.Flags().GetInt("limit")

		cfg := model.DefaultConfig()
		hlog := history.New(afero.NewOsFs(), history.DefaultPath(cfg.DataDir))
		if clear {
			if err := hlog.Clear(); err != nil {
				return err
			}
			fmt.Println("Audit history cleared.")
			return nil
		}
		entries, err := hlog.Read()
		if err != nil {
			return err
		}
		if limit > 0 && len(entries) > limit {
			entries = entries[:limit]
		}
		if asJSON {
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(entries)
		}
		if len(entries) == 0 {
			fmt.Println("No uninstallation history found.")
			return nil
		}
		fmt.Printf("%-20s %-24s %-10s %-8s %12s %s\n", "DATE", "APP", "SOURCE", "STATUS", "FREED", "DETAIL")
		for _, e := range entries {
			fmt.Printf("%-20s %-24s %-10s %-8s %12s %s\n",
				e.Time.Format("2006-01-02 15:04:05"),
				e.App, e.Source, e.Status,
				model.HumanSize(e.FreedKB), e.Detail)
		}
		return nil
	},
}

var tuiCmd = &cobra.Command{
	Use:   "tui",
	Short: "Launch the interactive terminal UI (default)",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runTUI()
	},
}

func init() {
	rootCmd.AddCommand(scanCmd, cleanCmd, orphansCmd, cacheCmd, historyCmd, tuiCmd)
	scanCmd.Flags().Bool("json", false, "output as JSON")

	cleanCmd.Flags().String("source", "", "only match apps from this source")
	cleanCmd.Flags().BoolP("yes", "y", false, "proceed with removal non-interactively")
	cleanCmd.Flags().Bool("dry-run", false, "inspect and stage paths without deleting")

	orphansCmd.Flags().Bool("json", false, "output as JSON")
	orphansCmd.Flags().Bool("clean", false, "clean all discovered orphans")
	orphansCmd.Flags().BoolP("yes", "y", false, "proceed without confirmation prompt")

	cacheCmd.Flags().Bool("json", false, "output as JSON")
	cacheCmd.Flags().Bool("clean", false, "clean all discovered cache/residual items")
	cacheCmd.Flags().BoolP("yes", "y", false, "proceed without confirmation prompt")

	historyCmd.Flags().Bool("json", false, "output as JSON")
	historyCmd.Flags().Bool("clear", false, "clear the audit log file")
	historyCmd.Flags().IntP("limit", "n", 50, "limit output to N most recent entries")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "veet:", err)
		os.Exit(1)
	}
}

// runTUI launches the Bubble Tea program; refuses to run as root.
func runTUI() error {
	if os.Geteuid() == 0 {
		return fmt.Errorf("refusing to run the TUI as root — elevation happens per-command via sudo instead")
	}
	cfg := model.DefaultConfig()
	if err := cfg.Load(); err != nil {
		return fmt.Errorf("config: %w", err)
	}
	program := tea.NewProgram(ui.New(cfg), tea.WithAltScreen())
	if _, err := program.Run(); err != nil {
		return err
	}
	return nil
}

func scanApps() ([]model.AppInfo, []string, []error) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	cfg := model.DefaultConfig()
	return detector.Scan(ctx, detector.RealExec{}, cfg.Home)
}

func printTable(apps []model.AppInfo) {
	if len(apps) == 0 {
		fmt.Println("no applications found")
		return
	}
	fmt.Printf("%-28s %-12s %-10s %12s\n", "APP/PACKAGE", "VERSION", "SOURCE", "SIZE")
	for _, a := range apps {
		size := model.HumanSize(a.InstallSizeKB)
		if a.InstallSizeKB == 0 {
			size = "-"
		}
		fmt.Printf("%-28s %-12s %-10s %12s\n", a.Name, a.Version, a.Source, size)
	}
}

type jsonApp struct {
	Name      string `json:"name"`
	Version   string `json:"version"`
	Source    string `json:"source"`
	Installed string `json:"installed,omitempty"`
	SizeKB    int64  `json:"size_kb"`
	Protected bool   `json:"protected"`
}

func printJSON(apps []model.AppInfo) error {
	out := make([]jsonApp, len(apps))
	for i, a := range apps {
		inst := ""
		if !a.InstalledOn.IsZero() {
			inst = a.InstalledOn.Format("2006-01-02")
		}
		out[i] = jsonApp{
			Name: a.Name, Version: a.Version, Source: a.Source,
			Installed: inst, SizeKB: a.InstallSizeKB, Protected: a.Protected,
		}
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}

// runClean implements multi-app cleaning with interactive prompts and dry-run support.
func runClean(names []string, source string, yes, dryRun bool) error {
	cfg := model.DefaultConfig()
	if err := cfg.Load(); err != nil {
		return err
	}
	apps, skipped, _ := scanApps()
	for _, s := range skipped {
		fmt.Fprintln(os.Stderr, "skipped:", s)
	}

	var matches []*model.AppInfo
	for _, name := range names {
		var match *model.AppInfo
		for i := range apps {
			a := &apps[i]
			if a.Name == name || strings.EqualFold(a.Name, name) {
				if source != "" && a.Source != source {
					continue
				}
				match = a
				break
			}
		}
		if match == nil {
			if source != "" {
				return fmt.Errorf("no installed application named %q from source %q", name, source)
			}
			return fmt.Errorf("no installed application named %q", name)
		}
		if cfg.ProtectedSet()[match.Name] || match.Protected {
			return fmt.Errorf("refusing: %s is a protected system component", match.Name)
		}
		matches = append(matches, match)
	}

	hlog := history.New(afero.NewOsFs(), history.DefaultPath(cfg.DataDir))
	un := uninstaller.New(cfg.Home, hlog, cfg.ProtectedSet())

	var totalKB int64
	var totalFiles int
	for _, m := range matches {
		if err := un.StageRemoval(m); err != nil {
			return err
		}
		totalKB += m.Removable.TotalKB()
		totalFiles += m.Removable.FileCount()
	}

	fmt.Println()
	for _, m := range matches {
		fmt.Printf("• %s (%s)\n", m.Name, m.Source)
		fmt.Printf("  would free %s across %d path(s)\n",
			model.HumanSize(m.Removable.TotalKB()), m.Removable.FileCount())
		for _, p := range m.Removable.Paths {
			tag := ""
			for _, e := range m.Removable.Elevated {
				if e == p {
					tag = "  [requires sudo]"
				}
			}
			fmt.Println("    " + p + tag)
		}
	}
	fmt.Printf("\nTotal reclaimable: %s across %d path(s) (%d app(s))\n\n",
		model.HumanSize(totalKB), totalFiles, len(matches))

	if dryRun {
		fmt.Println("Dry-run mode: no files or packages were removed.")
		return nil
	}

	if !yes {
		fmt.Print("Proceed with uninstallation and residual removal? [y/N]: ")
		reader := bufio.NewReader(os.Stdin)
		input, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("aborted")
		}
		input = strings.TrimSpace(strings.ToLower(input))
		if input != "y" && input != "yes" {
			fmt.Println("Aborted by user.")
			return nil
		}
	}

	for _, m := range matches {
		cmdArgs := uninstaller.RemoveCommand(*m, true)
		if len(cmdArgs) > 0 {
			fmt.Printf("\nrunning: %s\n", strings.Join(cmdArgs, " "))
		}
		freed, files, err := un.Remove(context.Background(), m, func(s uninstaller.Step) {
			icon := "✓"
			if s.Err != nil {
				icon = "✗"
			}
			fmt.Printf("  %s %s\n", icon, s.Text)
		})
		if err != nil {
			return fmt.Errorf("failed removing %s: %w", m.Name, err)
		}
		fmt.Printf("Done: freed %s, removed %d path(s)\n", model.HumanSize(freed), files)
	}
	return nil
}
