package main

import (
	"fmt"
	"log"

	"github.com/m1kkY8/termftp/internal/config"
	"github.com/m1kkY8/termftp/internal/sftpclient"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
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
