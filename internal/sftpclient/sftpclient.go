package sftpclient

import (
	"errors"
	"fmt"
	"net"
	"strconv"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"

	"github.com/m1kkY8/termftp/internal/config"
)

const defaultSSHPort = 22

type Client struct {
	*sftp.Client
	sshConn *ssh.Client
}

func New(cfg *config.Config) (*Client, error) {
	if cfg == nil {
		return nil, errors.New("config is nil")
	}

	sshConfig := &ssh.ClientConfig{
		User: cfg.User,
		Auth: []ssh.AuthMethod{
			ssh.Password(cfg.Password),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	addr := address(cfg)

	sshConn, err := ssh.Dial("tcp", addr, sshConfig)
	if err != nil {
		return nil, fmt.Errorf("dial ssh: %w", err)
	}

	sftpConn, err := sftp.NewClient(sshConn)
	if err != nil {
		sshConn.Close()
		return nil, fmt.Errorf("create sftp client: %w", err)
	}

	return &Client{Client: sftpConn, sshConn: sshConn}, nil
}

func (c *Client) Close() error {
	if c == nil {
		return nil
	}

	var err error
	if c.Client != nil {
		err = c.Client.Close()
	}

	if closeErr := c.closeSSH(); err == nil {
		err = closeErr
	}

	return err
}

func (c *Client) closeSSH() error {
	if c.sshConn == nil {
		return nil
	}
	if err := c.sshConn.Close(); err != nil {
		return err
	}
	c.sshConn = nil
	return nil
}

func address(cfg *config.Config) string {
	port := cfg.Port
	if port == 0 {
		port = defaultSSHPort
	}
	return net.JoinHostPort(cfg.Host, strconv.Itoa(port))
}
