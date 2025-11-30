package ui

import (
	"io"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/pkg/sftp"
)

func (m *model) uploadSelected() tea.Cmd {
	if len(m.panes) < 2 || m.client == nil {
		return tea.Printf("remote client unavailable")
	}
	src := m.panes[paneLocal]
	dst := m.panes[paneRemote]
	row := src.table.SelectedRow()
	if row == nil {
		return tea.Printf("no file selected")
	}
	if row[colType] == rowTypeDir {
		return tea.Printf("directories not supported yet")
	}
	localPath := filepath.Join(src.cwd, row[colName])
	remotePath := filepath.Join(dst.cwd, row[colName])
	file, err := os.Open(localPath)
	if err != nil {
		return tea.Printf("open local file: %v", err)
	}
	defer file.Close()
	if err := ensureRemoteDir(m.client, filepath.Dir(remotePath)); err != nil {
		return tea.Printf("prep remote dir: %v", err)
	}
	rc, err := m.client.Create(remotePath)
	if err != nil {
		return tea.Printf("create remote file: %v", err)
	}
	if _, err := io.Copy(rc, file); err != nil {
		rc.Close()
		return tea.Printf("upload failed: %v", err)
	}
	if err := rc.Close(); err != nil {
		return tea.Printf("finalize upload: %v", err)
	}
	_ = dst.changeDirectory(dst.cwd)
	return tea.Printf("uploaded %s -> %s", localPath, remotePath)
}

func ensureRemoteDir(client *sftp.Client, path string) error {
	if path == "." || path == "" {
		return nil
	}
	return client.MkdirAll(path)
}
