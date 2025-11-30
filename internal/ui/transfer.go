package ui

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/pkg/sftp"
)

const transferChunkSize = 64 * 1024

type transferProgressMsg struct {
	delta int64
	done  bool
	err   error
}

type transferJob struct {
	reader  io.Reader
	writer  io.Writer
	closers []io.Closer
	buffer  []byte
}

func (j *transferJob) step() (int64, bool, error) {
	if j.buffer == nil {
		j.buffer = make([]byte, transferChunkSize)
	}
	read, readErr := j.reader.Read(j.buffer)
	if read > 0 {
		if _, writeErr := j.writer.Write(j.buffer[:read]); writeErr != nil {
			return int64(read), true, writeErr
		}
	}
	if readErr != nil {
		if errors.Is(readErr, io.EOF) {
			return int64(read), true, nil
		}
		return int64(read), true, readErr
	}
	return int64(read), false, nil
}

func (j *transferJob) close() error {
	var err error
	for _, c := range j.closers {
		if c == nil {
			continue
		}
		if cerr := c.Close(); err == nil {
			err = cerr
		}
	}
	j.closers = nil
	return err
}

func (m *model) uploadSelected() tea.Cmd {
	if len(m.panes) < 2 || m.client == nil {
		return tea.Printf("remote client unavailable")
	}
	if m.transfer.active {
		return tea.Printf("transfer already running")
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
	info, err := file.Stat()
	if err != nil {
		file.Close()
		return tea.Printf("stat local file: %v", err)
	}
	if err := ensureRemoteDir(m.client, filepath.Dir(remotePath)); err != nil {
		file.Close()
		return tea.Printf("prep remote dir: %v", err)
	}
	rc, err := m.client.Create(remotePath)
	if err != nil {
		file.Close()
		return tea.Printf("create remote file: %v", err)
	}
	m.job = &transferJob{
		reader:  file,
		writer:  rc,
		closers: []io.Closer{file, rc},
	}
	m.transfer = transferState{
		active:      true,
		direction:   "Upload",
		filename:    filepath.Base(localPath),
		total:       info.Size(),
		transferred: 0,
		started:     time.Now(),
		lastUpdate:  time.Now(),
		refreshPane: paneRemote,
	}
	return tea.Batch(m.processTransferChunk())
}

func (m *model) downloadSelected() tea.Cmd {
	if len(m.panes) < 2 || m.client == nil {
		return tea.Printf("remote client unavailable")
	}
	if m.transfer.active {
		return tea.Printf("transfer already running")
	}
	remotePane := m.panes[paneRemote]
	localPane := m.panes[paneLocal]
	row := remotePane.table.SelectedRow()
	if row == nil {
		return tea.Printf("no file selected")
	}
	if row[colType] == rowTypeDir {
		return tea.Printf("directories not supported yet")
	}
	remotePath := filepath.Join(remotePane.cwd, row[colName])
	localPath := filepath.Join(localPane.cwd, row[colName])
	if err := os.MkdirAll(filepath.Dir(localPath), 0o755); err != nil {
		return tea.Printf("prepare local dir: %v", err)
	}
	remoteFile, err := m.client.Open(remotePath)
	if err != nil {
		return tea.Printf("open remote file: %v", err)
	}
	info, err := remoteFile.Stat()
	if err != nil {
		remoteFile.Close()
		return tea.Printf("stat remote file: %v", err)
	}
	localFile, err := os.Create(localPath)
	if err != nil {
		remoteFile.Close()
		return tea.Printf("create local file: %v", err)
	}
	m.job = &transferJob{
		reader:  remoteFile,
		writer:  localFile,
		closers: []io.Closer{remoteFile, localFile},
	}
	m.transfer = transferState{
		active:      true,
		direction:   "Download",
		filename:    filepath.Base(remotePath),
		total:       info.Size(),
		transferred: 0,
		started:     time.Now(),
		lastUpdate:  time.Now(),
		refreshPane: paneLocal,
	}
	return tea.Batch(m.processTransferChunk())
}

func (m *model) processTransferChunk() tea.Cmd {
	job := m.job
	if job == nil {
		return nil
	}
	return func() tea.Msg {
		delta, done, err := job.step()
		return transferProgressMsg{delta: delta, done: done, err: err}
	}
}

func (m *model) handleTransferProgress(msg transferProgressMsg) tea.Cmd {
	if !m.transfer.active {
		return nil
	}
	if msg.delta > 0 {
		m.transfer.transferred += msg.delta
		now := time.Now()
		elapsed := now.Sub(m.transfer.lastUpdate).Seconds()
		if elapsed <= 0 {
			elapsed = 1
		}
		m.transfer.rate = float64(msg.delta) / elapsed
		m.transfer.lastUpdate = now
	}
	if msg.err != nil {
		return m.finishTransfer(msg.err)
	}
	if msg.done {
		return m.finishTransfer(nil)
	}
	return m.processTransferChunk()
}

func (m *model) finishTransfer(resultErr error) tea.Cmd {
	if m.job != nil {
		if cerr := m.job.close(); resultErr == nil {
			resultErr = cerr
		}
		m.job = nil
	}
	m.transfer.active = false
	m.transfer.err = resultErr
	refreshPane := m.transfer.refreshPane
	m.transfer.refreshPane = 0

	var cmds []tea.Cmd
	if resultErr == nil {
		if refreshPane >= 0 && refreshPane < len(m.panes) {
			_ = m.panes[refreshPane].changeDirectory(m.panes[refreshPane].cwd)
		}
		cmds = append(cmds, tea.Printf("%s complete: %s", strings.ToLower(m.transfer.direction), m.transfer.filename))
	} else {
		cmds = append(cmds, tea.Printf("%s failed: %v", strings.ToLower(m.transfer.direction), resultErr))
	}
	return tea.Batch(cmds...)
}

func ensureRemoteDir(client *sftp.Client, path string) error {
	if path == "." || path == "" {
		return nil
	}
	return client.MkdirAll(path)
}

func (t transferState) percent() float64 {
	if t.total == 0 {
		return 0
	}
	return float64(t.transferred) / float64(t.total)
}

func (t transferState) elapsed() time.Duration {
	if t.started.IsZero() {
		return 0
	}
	return time.Since(t.started)
}

func (t transferState) speed() float64 {
	elapsed := t.elapsed().Seconds()
	if elapsed <= 0 {
		return 0
	}
	return float64(t.transferred) / elapsed
}

func (t transferState) currentSpeed() float64 {
	if t.rate > 0 {
		return t.rate
	}
	return t.speed()
}

func (t transferState) eta() time.Duration {
	speed := t.currentSpeed()
	if speed <= 0 || t.total == 0 {
		return 0
	}
	remaining := float64(t.total - t.transferred)
	if remaining <= 0 {
		return 0
	}
	seconds := remaining / speed
	return time.Duration(seconds * float64(time.Second))
}
