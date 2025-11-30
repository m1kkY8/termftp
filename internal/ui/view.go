package ui

import (
	"fmt"
	"math"
	"time"

	"github.com/charmbracelet/lipgloss"
)

var (
	headerStyle       = lipgloss.NewStyle().Bold(true)
	errorStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
	transferPaneStyle = lipgloss.NewStyle().
				BorderStyle(lipgloss.NormalBorder()).
				BorderForeground(lipgloss.Color("68")).
				Padding(0, 1)
)

func (m *model) View() string {
	if len(m.panes) == 0 {
		return ""
	}
	views := make([]string, 0, len(m.panes))
	for _, p := range m.panes {
		views = append(views, p.render())
	}
	panes := lipgloss.JoinHorizontal(lipgloss.Top, views...)
	width := lipgloss.Width(panes)
	if width == 0 {
		width = m.width
	}
	transfer := m.renderTransferPane(width)
	return lipgloss.JoinVertical(lipgloss.Left, panes, "", transfer)
}

func (p *pane) render() string {
	border := blurredBorder
	if p.focused {
		border = focusedBorder
	}
	panelStyle := baseStyle.
		Width(p.width).
		BorderForeground(border)

	title := fmt.Sprintf("%s: %s", p.title, p.cwd)
	body := panelStyle.Render(p.table.View())
	if p.err != nil {
		body += "\n" + errorStyle.Render(p.err.Error())
	}
	return headerStyle.Render(title) + "\n" + body
}

func (m *model) renderTransferPane(width int) string {
	if width <= 0 {
		width = m.width
	}
	if width <= 0 {
		for _, p := range m.panes {
			width += p.width
		}
	}
	width = max(20, width)
	body := "No active transfers"
	if m.transfer.active {
		percent := math.Max(0, math.Min(1, m.transfer.percent()))
		m.progress.Width = max(10, width-4)
		bar := m.progress.ViewAs(percent)
		stats := fmt.Sprintf(
			"%s %s %s/%s (%s) • %s • ETA %s • Elapsed %s",
			m.transfer.direction,
			formatFilename(m.transfer.filename),
			formatBytes(m.transfer.transferred),
			formatBytes(m.transfer.total),
			formatPercent(percent, m.transfer.total > 0),
			formatSpeed(m.transfer),
			formatETA(m.transfer),
			formatElapsed(m.transfer),
		)
		body = stats + "\n" + bar
	} else if m.transfer.err != nil {
		body = errorStyle.Render(m.transfer.err.Error())
	}
	panel := transferPaneStyle.MaxWidth(width).Render(body)
	return lipgloss.JoinVertical(lipgloss.Left, headerStyle.Render("Transfer"), panel)
}

func formatSpeed(t transferState) string {
	speed := t.currentSpeed()
	if speed <= 0 {
		return "--"
	}
	return fmt.Sprintf("%s/s", humanSize(int64(speed)))
}

func formatETA(t transferState) string {
	eta := t.eta()
	if eta <= 0 || eta > 24*time.Hour {
		return "--"
	}
	minutes := eta / time.Minute
	seconds := (eta % time.Minute) / time.Second
	if eta >= time.Hour {
		hours := eta / time.Hour
		minutes = (eta % time.Hour) / time.Minute
		return fmt.Sprintf("%02d:%02d:%02d", hours, minutes, seconds)
	}
	return fmt.Sprintf("%02d:%02d", minutes, seconds)
}

func formatElapsed(t transferState) string {
	elapsed := t.elapsed()
	if elapsed <= 0 {
		return "--"
	}
	minutes := elapsed / time.Minute
	seconds := (elapsed % time.Minute) / time.Second
	if elapsed >= time.Hour {
		hours := elapsed / time.Hour
		minutes = (elapsed % time.Hour) / time.Minute
		return fmt.Sprintf("%02d:%02d:%02d", hours, minutes, seconds)
	}
	return fmt.Sprintf("%02d:%02d", minutes, seconds)
}

func formatBytes(value int64) string {
	if value <= 0 {
		return "0B"
	}
	return humanSize(value)
}

func formatPercent(percent float64, known bool) string {
	if !known {
		return "--"
	}
	return fmt.Sprintf("%3.0f%%", percent*100)
}

func formatFilename(name string) string {
	if name == "" {
		return ""
	}
	return fmt.Sprintf("[%s]", name)
}
