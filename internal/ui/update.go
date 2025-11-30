package ui

import (
	tea "github.com/charmbracelet/bubbletea"
)

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.resize(msg.Width, msg.Height)
		return m, nil
	case tea.KeyMsg:
		if cmd := m.handleKey(msg); cmd != nil {
			return m, cmd
		}
	case transferTickMsg:
		if cmd := m.handleTransferTick(); cmd != nil {
			return m, cmd
		}
	case transferDoneMsg:
		return m, m.finishTransfer(msg.err)
	}

	cmds := make([]tea.Cmd, 0, len(m.panes))
	for _, p := range m.panes {
		var cmd tea.Cmd
		p.table, cmd = p.table.Update(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	}
	return m, tea.Batch(cmds...)
}

func (m *model) handleKey(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "q", "ctrl+c":
		return tea.Quit
	case "tab":
		m.toggleFocus()
	case "ctrl+l":
		m.focusPane(paneLocal)
	case "ctrl+r":
		m.focusPane(paneRemote)
	case "enter":
		if pane := m.activePane(); pane != nil {
			return pane.openSelection()
		}
	case "backspace":
		if pane := m.activePane(); pane != nil {
			pane.navigateUp()
		}
	case "l":
		if len(m.panes) > paneLocal {
			return m.panes[paneLocal].openSelection()
		}
	case "L":
		if len(m.panes) > paneLocal {
			m.panes[paneLocal].navigateUp()
		}
	case "r":
		if len(m.panes) > paneRemote {
			return m.panes[paneRemote].openSelection()
		}
	case "R":
		if len(m.panes) > paneRemote {
			m.panes[paneRemote].navigateUp()
		}
	case "p":
		return m.uploadSelected()
	case "g":
		return m.downloadSelected()
	}
	return nil
}
