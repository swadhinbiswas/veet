package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"github.com/swadhinbiswas/veet/internal/detector"
	"github.com/swadhinbiswas/veet/internal/model"
	"github.com/swadhinbiswas/veet/internal/ui"
	"github.com/swadhinbiswas/veet/internal/uninstaller"
)

var rootCmd = &cobra.Command{
	Use:   "veet",
	Short: "Universal Linux App Uninstaller & Deep-Clean Residual Purger",
	Long: `VEET finds and completely uninstalls applications from every
common source — apt, dnf, pacman/AUR, zypper, flatpak, snap, appimage,
npm, pipx, cargo, gem and go install — removing configs, caches, logs
and residuals in one confirmed action.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runTUI()
	},
}

var scanCmd = &cobra.Command{
	Use:   "scan",
	Short: "List installed applications",
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
	Use:   "clean <app>",
	Short: "Deep-clean uninstall an application",
	Long: `Uninstall <app> and remove its configs, caches, logs and residuals.
Always shows a preview first; pass --yes to proceed. The package-manager
step and any root-owned paths are run via sudo.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		source, _ := cmd.Flags().GetString("source")
		yes, _ := cmd.Flags().GetBool("yes")
		return runClean(name, source, yes)
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

func main() {
	rootCmd.AddCommand(scanCmd, cleanCmd, tuiCmd)
	scanCmd.Flags().Bool("json", false, "output as JSON")
	cleanCmd.Flags().String("source", "", "only match apps from this source")
	cleanCmd.Flags().Bool("yes", false, "proceed with removal after preview")
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

// runClean implements the non-interactive deep-clean path.
func runClean(name, source string, yes bool) error {
	cfg := model.DefaultConfig()
	if err := cfg.Load(); err != nil {
		return err
	}
	apps, skipped, _ := scanApps()
	for _, s := range skipped {
		fmt.Fprintln(os.Stderr, "skipped:", s)
	}
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

	un := uninstaller.New(cfg.Home, nil, cfg.ProtectedSet())
	if err := un.StageRemoval(match); err != nil {
		return err
	}
	fmt.Printf("\n%s (%s)\n", match.Name, match.Source)
	fmt.Printf("would free %s across %d path(s)\n",
		model.HumanSize(match.Removable.TotalKB()), match.Removable.FileCount())
	for _, p := range match.Removable.Paths {
		tag := ""
		for _, e := range match.Removable.Elevated {
			if e == p {
				tag = "  [requires sudo]"
			}
		}
		fmt.Println("  " + p + tag)
	}
	fmt.Println()
	if !yes {
		return fmt.Errorf("preview above — re-run with --yes to uninstall")
	}
	fmt.Printf("running: %s\n", strings.Join(uninstaller.RemoveCommand(*match, true), " "))
	freed, files, err := un.Remove(context.Background(), match, func(s uninstaller.Step) {
		icon := "✓"
		if s.Err != nil {
			icon = "✗"
		}
		fmt.Printf("  %s %s\n", icon, s.Text)
	})
	if err != nil {
		return err
	}
	fmt.Printf("\nDone: freed %s, removed %d path(s)\n", model.HumanSize(freed), files)
	return nil
}
