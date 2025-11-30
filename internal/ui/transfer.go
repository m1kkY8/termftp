package ui

import (
	"errors"
	"io"
	"os"
	"path/filepath"
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
	local  *os.File
	remote *sftp.File
	buffer []byte
}

func (j *transferJob) step() (int64, bool, error) {
	if j.buffer == nil {
		j.buffer = make([]byte, transferChunkSize)
	}
	read, readErr := j.local.Read(j.buffer)
	if read > 0 {
		if _, writeErr := j.remote.Write(j.buffer[:read]); writeErr != nil {
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
	if j.local != nil {
		err = j.local.Close()
		j.local = nil
	}
	if j.remote != nil {
		if cerr := j.remote.Close(); err == nil {
			err = cerr
		}
		j.remote = nil
	}
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
	m.job = &transferJob{local: file, remote: rc}
	m.transfer = transferState{
		active:      true,
		direction:   "Upload",
		filename:    filepath.Base(localPath),
		total:       info.Size(),
		transferred: 0,
		started:     time.Now(),
		lastUpdate:  time.Now(),
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

	var cmds []tea.Cmd
	if resultErr == nil {
		if len(m.panes) > paneRemote {
			_ = m.panes[paneRemote].changeDirectory(m.panes[paneRemote].cwd)
		}
		cmds = append(cmds, tea.Printf("upload complete: %s", m.transfer.filename))
	} else {
		cmds = append(cmds, tea.Printf("upload failed: %v", resultErr))
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
