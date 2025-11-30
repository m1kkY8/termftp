package ui

import (
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

	return &model{
		panes:   []*pane{local, remote},
		focused: paneLocal,
		client:  remoteClient,
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
	paneHeight := max(7, height-2)
	for _, p := range m.panes {
		p.setSize(paneWidth, paneHeight)
	}
}
