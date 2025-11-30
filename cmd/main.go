package main

import (
	"log"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/m1kkY8/termftp/internal/config"
	"github.com/m1kkY8/termftp/internal/sftpclient"
	"github.com/m1kkY8/termftp/internal/ui"
)

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	client, err := sftpclient.New(cfg)
	if err != nil {
		log.Fatalf("init sftp client: %v", err)
	}
	defer client.Close()

	m := ui.New(ui.Options{
		LocalRoot:  localRoot(),
		RemoteRoot: cfg.Root,
		Client:     client.Client,
		Transfer: ui.TransferOptions{
			BufferSize:       cfg.BufferSizeBytes(),
			ParallelStreams:  cfg.ParallelStreams(),
			ProgressInterval: cfg.ProgressInterval(),
		},
	})
	if _, err := tea.NewProgram(m, tea.WithAltScreen()).Run(); err != nil {
		log.Fatalf("run ui: %v", err)
	}
}

func localRoot() string {
	if root := os.Getenv("TERMFTP_LOCAL_ROOT"); root != "" {
		if abs, err := filepath.Abs(root); err == nil {
			return abs
		}
		return root
	}
	if wd, err := os.Getwd(); err == nil {
		return wd
	}
	return "."
}
