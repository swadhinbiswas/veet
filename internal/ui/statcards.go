package ui

import (
	"strconv"

	"github.com/charmbracelet/lipgloss"

	"github.com/swadhinbiswas/veet/internal/model"
)

// Stats aggregates the numbers shown in the stat-card row.
type Stats struct {
	Total           int
	Flatpak         int
	Snap            int
	GlobalTools     int
	CacheKB         int64
	Reclaimable     int64
	PackageManagers int
}

// ComputeStats derives card numbers from the app list.
func ComputeStats(apps []model.AppInfo) Stats {
	s := Stats{Total: len(apps)}
	var cacheKB, reclaim int64
	toolSrcs := map[string]bool{"npm": true, "pipx": true, "cargo": true, "gem": true, "go": true}
	pmSrcs := map[string]bool{"apt": true, "dnf": true, "pacman": true, "aur": true, "zypper": true, "flatpak": true, "snap": true}
	for _, a := range apps {
		if toolSrcs[a.Source] {
			s.GlobalTools++
		}
		if pmSrcs[a.Source] {
			s.PackageManagers++
		}
		if a.Source == "flatpak" {
			s.Flatpak++
		}
		if a.Source == "snap" {
			s.Snap++
		}
		if a.Source == "cache" {
			cacheKB += a.InstallSizeKB
			reclaim += a.InstallSizeKB
		}
		if a.Removable.Staged() {
			reclaim += a.Removable.TotalKB()
		}
	}
	s.CacheKB = cacheKB
	s.Reclaimable = reclaim
	return s
}

type card struct {
	icon  string
	value string
	label string
	color string
}

// RenderStatCards lays out the six cards across the available width with active card indicator.
func RenderStatCards(s Stats, width int, activeIdx int) string {
	cards := []card{
		{"▣", strconv.Itoa(s.Total), "1: All Apps", Primary},
		{"◆", strconv.Itoa(s.Flatpak), "2: Flatpak", "#06B6D4"},
		{"▦", strconv.Itoa(s.Snap), "3: Snap", Danger},
		{"⚒", strconv.Itoa(s.GlobalTools), "4: Tools", Secondary},
		{"🧹", model.HumanSize(s.CacheKB), "5: Cache", Highlight},
		{"♻", model.HumanSize(s.Reclaimable), "6: Reclaim", Highlight},
	}
	if width < 80 {
		width = 80
	}
	n := len(cards)
	totalAvail := width - n
	if totalAvail < n*12 {
		totalAvail = n * 12
	}
	baseW := totalAvail / n
	rem := totalAvail % n

	var cells []string
	for i, c := range cards {
		cardW := baseW
		if i < rem {
			cardW++
		}
		innerW := cardW - 4
		if innerW < 8 {
			innerW = 8
		}

		isActive := i == activeIdx
		borderClr := panelBorder
		if isActive {
			borderClr = primary
		} else if c.color != "" {
			borderClr = lipgloss.Color(c.color)
		}

		tag := c.label
		if innerW < 12 {
			switch i {
			case 0:
				tag = "1:All"
			case 1:
				tag = "2:Flat"
			case 2:
				tag = "3:Snap"
			case 3:
				tag = "4:Tool"
			case 4:
				tag = "5:Cache"
			case 5:
				tag = "6:Recl"
			}
		}
		if isActive {
			tag = "●" + tag
		}
		tag = Truncate(tag, innerW)
		valText := Truncate(c.icon+" "+c.value, innerW)

		valStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(c.color))
		if isActive {
			valStyle = valStyle.Underline(true)
		}

		lblStyle := lipgloss.NewStyle().Foreground(mutedColor)
		if isActive {
			lblStyle = lipgloss.NewStyle().Bold(true).Foreground(primary)
		}

		cell := lipgloss.NewStyle().
			Width(innerW).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(borderClr).
			Padding(0, 1).
			Render(lipgloss.JoinVertical(lipgloss.Left,
				valStyle.Render(valText),
				lblStyle.Render(tag),
			))
		cells = append(cells, cell)
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, cells...)
}
