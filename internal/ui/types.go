package ui

import (
	"time"

	"github.com/charmbracelet/bubbles/progress"
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
	panes       []*pane
	focused     int
	width       int
	height      int
	client      *sftp.Client
	progress    progress.Model
	transfer    transferState
	job         *transferJob
	transferCfg transferConfig
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

type transferState struct {
	active      bool
	direction   string
	filename    string
	total       int64
	transferred int64
	started     time.Time
	err         error
	lastUpdate  time.Time
	rate        float64
	refreshPane int
}

type transferConfig struct {
	bufferSize       int
	streams          int
	progressInterval time.Duration
}

type Options struct {
	LocalRoot  string
	RemoteRoot string
	Client     *sftp.Client
	Transfer   TransferOptions
}

type TransferOptions struct {
	BufferSize       int
	ParallelStreams  int
	ProgressInterval time.Duration
}
