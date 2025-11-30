package ui

import (
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/lipgloss"
	"github.com/pkg/sftp"
)

const (
	colName = iota
	colType
	colSize
)

const (
	rowTypeDir  = "dir"
	rowTypeFile = "file"
)

const (
	paneLocal = iota
	paneRemote
)

var (
	baseStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("240"))
	focusedBorder = lipgloss.Color("57")
	blurredBorder = lipgloss.Color("240")
)

type dirProvider interface {
	ReadDir(path string) ([]entry, error)
}

// entry represents minimal file metadata used by the UI tables.
type entry struct {
	name  string
	isDir bool
	size  int64
}

type model struct {
	panes   []*pane
	focused int
	width   int
	height  int
	client  *sftp.Client
}

type pane struct {
	title    string
	provider dirProvider
	table    table.Model
	cwd      string
	err      error
	width    int
	height   int
	focused  bool
	readonly bool
}
