package sftpclient

import (
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"

	"github.com/m1kkY8/termftp/internal/config"
)

const defaultSSHPort = 22
const (
	maxPacketBytes               = 1 << 20 // 1 MiB packets keep queues full
	fallbackPacketBytes          = 32 * 1024
	maxConcurrentRequestsPerFile = 64
)

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

	sftpConn, err := dialSFTP(sshConn)
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

func dialSFTP(sshConn *ssh.Client) (*sftp.Client, error) {
	primaryOpts := []sftp.ClientOption{
		sftp.MaxPacket(maxPacketBytes),
		sftp.MaxConcurrentRequestsPerFile(maxConcurrentRequestsPerFile),
		sftp.UseConcurrentReads(true),
		sftp.UseConcurrentWrites(true),
	}
	client, err := sftp.NewClient(sshConn, primaryOpts...)
	if err == nil {
		return client, nil
	}
	if !shouldFallback(err) {
		return nil, err
	}
	fallbackOpts := []sftp.ClientOption{
		sftp.MaxPacket(fallbackPacketBytes),
		sftp.MaxConcurrentRequestsPerFile(maxConcurrentRequestsPerFile / 2),
		sftp.UseConcurrentReads(true),
		sftp.UseConcurrentWrites(true),
	}
	return sftp.NewClient(sshConn, fallbackOpts...)
}

func shouldFallback(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "sizes larger than 32KB") || strings.Contains(strings.ToLower(msg), "maxpacket")
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
