package ui

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

var (
	headerStyle = lipgloss.NewStyle().Bold(true)
	errorStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
)

func (m *model) View() string {
	if len(m.panes) == 0 {
		return ""
	}
	views := make([]string, 0, len(m.panes))
	for _, p := range m.panes {
		views = append(views, p.render())
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, views...)
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
