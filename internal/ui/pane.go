package ui

import (
	"path/filepath"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func newPane(title, start string, provider dirProvider, readonly bool) *pane {
	t := table.New(
		table.WithColumns([]table.Column{
			{Title: "Name", Width: 30},
			{Title: "Type", Width: 6},
			{Title: "Size", Width: 10},
		}),
		table.WithHeight(15),
	)

	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		BorderBottom(true).
		Bold(false)
	s.Selected = s.Selected.
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("57")).
		Bold(false)
	t.SetStyles(s)

	p := &pane{
		title:    title,
		provider: provider,
		table:    t,
		readonly: readonly,
	}
	p.setSize(40, 20)
	if err := p.changeDirectory(start); err != nil {
		p.err = err
	}
	return p
}

func (p *pane) openSelection() tea.Cmd {
	row := p.table.SelectedRow()
	if row == nil {
		return nil
	}

	name := row[colName]
	if row[colType] != rowTypeDir {
		return tea.Printf("[%s] selected file: %s", p.title, filepath.Join(p.cwd, name))
	}

	if name == ".." {
		p.navigateUp()
		return nil
	}

	next := filepath.Join(p.cwd, name)
	if err := p.changeDirectory(next); err != nil {
		return tea.Printf("[%s] open %s: %v", p.title, next, err)
	}
	return nil
}

func (p *pane) navigateUp() {
	parent := filepath.Dir(p.cwd)
	if parent == p.cwd {
		return
	}
	_ = p.changeDirectory(parent)
}

func (p *pane) changeDirectory(path string) error {
	clean := filepath.Clean(path)
	entries, err := p.provider.ReadDir(clean)
	if err != nil {
		p.err = err
		return err
	}
	rows := make([]table.Row, 0, len(entries)+1)
	rows = append(rows, table.Row{"..", rowTypeDir, ""})
	for _, entry := range entries {
		rows = append(rows, table.Row{
			entry.name,
			map[bool]string{true: rowTypeDir, false: rowTypeFile}[entry.isDir],
			formatSize(entry),
		})
	}
	sortRows(rows[1:])
	p.cwd = clean
	p.table.SetRows(rows)
	p.table.GotoTop()
	p.err = nil
	return nil
}

func (p *pane) focus(active bool) {
	p.focused = active
	if active {
		p.table.Focus()
	} else {
		p.table.Blur()
	}
}

func (p *pane) setSize(width, height int) {
	if width <= 0 || height <= 0 {
		return
	}
	p.width = width
	p.height = height
	nameWidth := max(10, width-20)
	cols := p.table.Columns()
	if len(cols) >= 3 {
		cols[colName].Width = nameWidth
		cols[colType].Width = 6
		cols[colSize].Width = 12
		p.table.SetColumns(cols)
	}
	innerHeight := height - 3
	if innerHeight < 5 {
		innerHeight = height
	}
	p.table.SetHeight(innerHeight)
}
