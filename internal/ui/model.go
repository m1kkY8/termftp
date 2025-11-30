package ui

import (
	"time"

	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
)

func New(opts Options) *model {
	transferCfg := normalizeTransferOptions(opts.Transfer)
	local := newPane("Local", defaultLocalRoot(opts.LocalRoot), localProvider{}, false)
	remoteProvider := dirProvider(localProvider{})
	readonly := true
	if opts.Client != nil {
		remoteProvider = &sftpProvider{client: opts.Client}
		readonly = false
	}
	remote := newPane("Remote", defaultRemoteRoot(opts.RemoteRoot), remoteProvider, readonly)

	local.focus(true)
	remote.focus(false)

	bar := progress.New(progress.WithDefaultGradient())
	bar.Width = 40

	return &model{
		panes:       []*pane{local, remote},
		focused:     paneLocal,
		client:      opts.Client,
		progress:    bar,
		transferCfg: transferCfg,
	}
}

func (m *model) Init() tea.Cmd { return nil }

func (m *model) activePane() *pane {
	if len(m.panes) == 0 {
		return nil
	}
	return m.panes[m.focused]
}

func (m *model) focusPane(idx int) {
	if idx < 0 || idx >= len(m.panes) || idx == m.focused {
		return
	}
	m.panes[m.focused].focus(false)
	m.focused = idx
	m.panes[m.focused].focus(true)
}

func (m *model) toggleFocus() {
	if len(m.panes) < 2 {
		return
	}
	m.focusPane((m.focused + 1) % len(m.panes))
}

func (m *model) resize(width, height int) {
	m.width = width
	m.height = height
	if len(m.panes) == 0 {
		return
	}
	gap := 4
	paneWidth := max(20, (width-gap)/len(m.panes))
	availableHeight := height
	if availableHeight <= 0 {
		availableHeight = 20
	}
	transferReserve := max(6, availableHeight/4)
	paneHeight := max(7, int(float64(availableHeight-transferReserve)*0.6))
	for _, p := range m.panes {
		p.setSize(paneWidth, paneHeight)
	}
	if width > 0 {
		m.progress.Width = max(10, width-4)
	}
}

func normalizeTransferOptions(opts TransferOptions) transferConfig {
	bufferSize := opts.BufferSize
	if bufferSize <= 0 {
		bufferSize = 8 * 1024 * 1024
	}
	// Clamp to avoid excessive memory use.
	if bufferSize < 1024*1024 {
		bufferSize = 1024 * 1024
	}
	if bufferSize > 64*1024*1024 {
		bufferSize = 64 * 1024 * 1024
	}
	streams := opts.ParallelStreams
	if streams <= 0 {
		streams = 4
	}
	if streams > 32 {
		streams = 32
	}
	interval := opts.ProgressInterval
	if interval <= 0 {
		interval = 75 * time.Millisecond
	}
	if interval < 25*time.Millisecond {
		interval = 25 * time.Millisecond
	}
	if interval > time.Second {
		interval = time.Second
	}
	return transferConfig{
		bufferSize:       bufferSize,
		streams:          streams,
		progressInterval: interval,
	}
}
