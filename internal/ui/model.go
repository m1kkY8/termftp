package ui

import (
	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/pkg/sftp"
)

func New(localRoot, remoteRoot string, remoteClient *sftp.Client) *model {
	local := newPane("Local", defaultLocalRoot(localRoot), localProvider{}, false)
	remoteProvider := dirProvider(localProvider{})
	readonly := true
	if remoteClient != nil {
		remoteProvider = &sftpProvider{client: remoteClient}
		readonly = false
	}
	remote := newPane("Remote", defaultRemoteRoot(remoteRoot), remoteProvider, readonly)

	local.focus(true)
	remote.focus(false)

	bar := progress.New(progress.WithDefaultGradient())
	bar.Width = 40

	return &model{
		panes:    []*pane{local, remote},
		focused:  paneLocal,
		client:   remoteClient,
		progress: bar,
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
