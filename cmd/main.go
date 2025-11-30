package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/m1kkY8/termftp/internal/config"
	"github.com/m1kkY8/termftp/internal/sftpclient"
	"github.com/m1kkY8/termftp/internal/ui"
)

func main() {
	m := ui.New()
	if _, err := tea.NewProgram(m).Run(); err != nil {
		fmt.Println("Error running program:", err)
		os.Exit(1)
	}

	// if err := run(); err != nil {
	// 	log.Fatal(err)
	// }
}

func run() error {
	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	client, err := sftpclient.New(cfg)
	if err != nil {
		return fmt.Errorf("init sftp client: %w", err)
	}
	defer client.Close()

	files, err := client.ReadDir(cfg.Root)
	if err != nil {
		return fmt.Errorf("read remote dir %q: %w", cfg.Root, err)
	}

	for _, file := range files {
		fmt.Printf("%s\tSize: %dB\n", file.Name(), file.Size())
	}

	return nil
}
