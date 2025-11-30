package ui

import (
	"os"
	"path/filepath"

	"github.com/pkg/sftp"
)

type localProvider struct{}

func (localProvider) ReadDir(path string) ([]entry, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}
	result := make([]entry, 0, len(entries))
	for _, e := range entries {
		info, err := e.Info()
		if err != nil {
			return nil, err
		}
		result = append(result, entry{
			name:  e.Name(),
			isDir: e.IsDir(),
			size:  info.Size(),
		})
	}
	return result, nil
}

type sftpProvider struct {
	client *sftp.Client
}

func (p *sftpProvider) ReadDir(path string) ([]entry, error) {
	files, err := p.client.ReadDir(path)
	if err != nil {
		return nil, err
	}
	result := make([]entry, 0, len(files))
	for _, f := range files {
		result = append(result, entry{
			name:  f.Name(),
			isDir: f.IsDir(),
			size:  f.Size(),
		})
	}
	return result, nil
}

func defaultLocalRoot(path string) string {
	if path != "" {
		if abs, err := filepath.Abs(path); err == nil {
			return abs
		}
		return path
	}
	if wd, err := os.Getwd(); err == nil {
		return wd
	}
	return "."
}

func defaultRemoteRoot(path string) string {
	if path == "" {
		return "/"
	}
	return path
}
