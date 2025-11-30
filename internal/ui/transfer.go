package ui

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/pkg/sftp"
)

type transferTickMsg struct{}
type transferDoneMsg struct {
	err error
}

type transferJob struct {
	reader      io.Reader
	writer      io.Writer
	readerAt    io.ReaderAt
	writerAt    io.WriterAt
	size        int64
	bufferSize  int
	streams     int
	closers     []io.Closer
	transferred atomic.Int64
}

func newTransferJob(reader io.Reader, writer io.Writer, size int64, closers []io.Closer, cfg transferConfig, readerAt io.ReaderAt, writerAt io.WriterAt) *transferJob {
	if cfg.bufferSize <= 0 {
		cfg.bufferSize = 8 * 1024 * 1024
	}
	streams := cfg.streams
	if streams <= 0 {
		streams = 1
	}
	return &transferJob{
		reader:     reader,
		writer:     writer,
		readerAt:   readerAt,
		writerAt:   writerAt,
		size:       size,
		bufferSize: cfg.bufferSize,
		streams:    streams,
		closers:    closers,
	}
}

func (j *transferJob) run() error {
	if j.shouldUseParallel() {
		return j.copyParallel()
	}
	return j.copySequential()
}

func (j *transferJob) shouldUseParallel() bool {
	return j.streams > 1 && j.readerAt != nil && j.writerAt != nil && j.size > int64(j.bufferSize)
}

func (j *transferJob) copySequential() error {
	buffer := make([]byte, j.bufferSize)
	writer := &countingWriter{dst: j.writer, job: j}
	_, err := io.CopyBuffer(writer, j.reader, buffer)
	return err
}

func (j *transferJob) copyParallel() error {
	streams := j.streams
	if streams <= 1 {
		return j.copySequential()
	}
	chunkSize := (j.size + int64(streams) - 1) / int64(streams)
	var wg sync.WaitGroup
	errCh := make(chan error, streams)
	for i := 0; i < streams; i++ {
		offset := int64(i) * chunkSize
		length := minInt64(chunkSize, j.size-offset)
		if length <= 0 {
			continue
		}
		wg.Add(1)
		go func(off, ln int64) {
			defer wg.Done()
			reader := io.NewSectionReader(j.readerAt, off, ln)
			writer := &writerAtSection{WriterAt: j.writerAt, offset: off}
			buffer := make([]byte, j.bufferSize)
			_, err := io.CopyBuffer(&countingWriter{dst: writer, job: j}, reader, buffer)
			if err != nil && err != io.EOF {
				errCh <- err
			}
		}(offset, length)
	}
	wg.Wait()
	close(errCh)
	for err := range errCh {
		if err != nil {
			return err
		}
	}
	return nil
}

func (j *transferJob) add(n int64) {
	if n > 0 {
		j.transferred.Add(n)
	}
}

func (j *transferJob) transferredBytes() int64 {
	return j.transferred.Load()
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

type countingWriter struct {
	dst io.Writer
	job *transferJob
}

func (w *countingWriter) Write(p []byte) (int, error) {
	n, err := w.dst.Write(p)
	if n > 0 {
		w.job.add(int64(n))
	}
	return n, err
}

type writerAtSection struct {
	io.WriterAt
	offset int64
}

func (w *writerAtSection) Write(p []byte) (int, error) {
	n, err := w.WriterAt.WriteAt(p, w.offset)
	w.offset += int64(n)
	return n, err
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
	adviseSequential(file)
	info, err := file.Stat()
	if err != nil {
		file.Close()
		return tea.Printf("stat local file: %v", err)
	}
	if err := ensureRemoteDir(m.client, filepath.Dir(remotePath)); err != nil {
		file.Close()
		return tea.Printf("prep remote dir: %v", err)
	}
	rc, err := m.client.OpenFile(remotePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC)
	if err != nil {
		file.Close()
		return tea.Printf("create remote file: %v", err)
	}
	_ = preallocateRemote(rc, info.Size())
	m.setupTransferJob(
		newTransferJob(file, rc, info.Size(), []io.Closer{file, rc}, m.transferCfg, file, rc),
		transferState{
			active:      true,
			direction:   "Upload",
			filename:    filepath.Base(localPath),
			total:       info.Size(),
			started:     time.Now(),
			lastUpdate:  time.Now(),
			refreshPane: paneRemote,
		},
	)
	return m.startTransfer()
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
	localFile, err := os.OpenFile(localPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		remoteFile.Close()
		return tea.Printf("create local file: %v", err)
	}
	adviseSequential(localFile)
	m.setupTransferJob(
		newTransferJob(remoteFile, localFile, info.Size(), []io.Closer{remoteFile, localFile}, m.transferCfg, remoteFile, localFile),
		transferState{
			active:      true,
			direction:   "Download",
			filename:    filepath.Base(remotePath),
			total:       info.Size(),
			started:     time.Now(),
			lastUpdate:  time.Now(),
			refreshPane: paneLocal,
		},
	)
	return m.startTransfer()
}

func (m *model) setupTransferJob(job *transferJob, state transferState) {
	m.job = job
	state.transferred = 0
	m.transfer = state
}

func (m *model) startTransfer() tea.Cmd {
	return tea.Batch(m.runTransferJob(), m.scheduleProgressTick())
}

func (m *model) runTransferJob() tea.Cmd {
	job := m.job
	if job == nil {
		return nil
	}
	return func() tea.Msg {
		err := job.run()
		return transferDoneMsg{err: err}
	}
}

func (m *model) scheduleProgressTick() tea.Cmd {
	if !m.transfer.active {
		return nil
	}
	interval := m.transferCfg.progressInterval
	return tea.Tick(interval, func(time.Time) tea.Msg {
		return transferTickMsg{}
	})
}

func (m *model) handleTransferTick() tea.Cmd {
	if !m.transfer.active || m.job == nil {
		return nil
	}
	total := m.job.transferredBytes()
	delta := total - m.transfer.transferred
	if delta > 0 {
		now := time.Now()
		elapsed := now.Sub(m.transfer.lastUpdate).Seconds()
		if elapsed <= 0 {
			elapsed = 1e-6
		}
		instant := float64(delta) / elapsed
		m.transfer.rate = smoothRate(m.transfer.rate, instant)
		m.transfer.lastUpdate = now
		m.transfer.transferred = total
	}
	return m.scheduleProgressTick()
}

func smoothRate(previous, instant float64) float64 {
	const alpha = 0.35
	if previous <= 0 {
		return instant
	}
	if instant <= 0 {
		return previous * (1 - alpha)
	}
	return previous*(1-alpha) + instant*alpha
}

func (m *model) finishTransfer(resultErr error) tea.Cmd {
	if m.job != nil {
		m.transfer.transferred = m.job.transferredBytes()
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

func preallocateRemote(f *sftp.File, size int64) error {
	if f == nil || size <= 0 {
		return nil
	}
	return f.Truncate(size)
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
	if t.total <= 0 {
		return 0
	}
	rate := t.currentSpeed()
	if rate <= 0 {
		return 0
	}
	remaining := float64(t.total - t.transferred)
	if remaining <= 0 {
		return 0
	}
	return time.Duration(remaining/rate) * time.Second
}

func minInt64(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}
