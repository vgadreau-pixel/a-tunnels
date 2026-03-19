package ssh

import (
	"crypto/rand"
	"crypto/rsa"
	"fmt"
	"io"
	"log"
	"net"

	"github.com/a-tunnels/a-tunnels/internal/config"
	"github.com/a-tunnels/a-tunnels/internal/tunnel"
	"golang.org/x/crypto/ssh"
)

type Server struct {
	addr      string
	config    *config.ServerConfig
	tunnelMgr tunnel.Manager
}

func NewServer(addr string, cfg *config.ServerConfig, mgr tunnel.Manager) *Server {
	return &Server{
		addr:      addr,
		config:    cfg,
		tunnelMgr: mgr,
	}
}

func (s *Server) Start() error {
	config := &ssh.ServerConfig{
		PasswordCallback: s.authenticate,
	}

	privateKey, err := generateHostKey()
	if err != nil {
		return err
	}
	config.AddHostKey(privateKey)

	ln, err := net.Listen("tcp", s.addr)
	if err != nil {
		return err
	}

	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				log.Printf("SSH accept error: %v", err)
				continue
			}
			go s.handleConnection(conn, config)
		}
	}()

	log.Printf("SSH server started on %s", s.addr)
	return nil
}

func (s *Server) authenticate(c ssh.ConnMetadata, password []byte) (*ssh.Permissions, error) {
	for _, key := range s.config.Auth.APIKeys {
		if string(password) == key {
			return &ssh.Permissions{}, nil
		}
	}
	return nil, fmt.Errorf("invalid credentials")
}

func (s *Server) handleConnection(conn net.Conn, config *ssh.ServerConfig) {
	defer conn.Close()

	sshConn, chans, reqs, err := ssh.NewServerConn(conn, config)
	if err != nil {
		log.Printf("SSH handshake failed: %v", err)
		return
	}
	defer sshConn.Close()

	go ssh.DiscardRequests(reqs)

	for newChannel := range chans {
		if newChannel.ChannelType() != "session" {
			newChannel.Reject(ssh.UnknownChannelType, "unknown channel type")
			continue
		}

		channel, requests, err := newChannel.Accept()
		if err != nil {
			continue
		}

		go s.handleSession(channel, requests)
	}
}

func (s *Server) handleSession(channel ssh.Channel, requests <-chan *ssh.Request) {
	defer channel.Close()

	for req := range requests {
		switch req.Type {
		case "shell":
			channel.SendRequest("exit-status", false, []byte("0"))
			s.interactiveSession(channel)

		case "exec":
			s.execCommand(channel, string(req.Payload))

		default:
			req.Reply(false, nil)
		}
	}
}

func (s *Server) interactiveSession(channel ssh.Channel) {
	tunnels := s.tunnelMgr.List()
	welcome := "A-Tunnels CLI\n"
	welcome += fmt.Sprintf("Connected tunnels: %d\n\n", len(tunnels))
	welcome += "Commands: list, create, delete, stats, logs, restart, exit\n"
	channel.Write([]byte(welcome))

	buf := make([]byte, 1024)
	for {
		channel.Write([]byte("> "))
		n, err := channel.Read(buf)
		if err != nil {
			return
		}

		cmd := string(buf[:n])
		s.handleCommand(channel, cmd)
	}
}

func (s *Server) execCommand(channel ssh.Channel, cmd string) {
	s.handleCommand(channel, cmd)
	channel.SendRequest("exit-status", false, []byte("0"))
}

func (s *Server) handleCommand(channel ssh.Channel, cmd string) {
	tunnels := s.tunnelMgr.List()

	switch {
	case cmd == "list" || cmd == "list\n":
		for _, t := range tunnels {
			fmt.Fprintf(channel, "%s\t%s\t%s\t%s\n", t.ID, t.Name, t.Protocol, t.Status)
		}

	case cmd == "exit" || cmd == "exit\n":
		channel.Close()

	default:
		channel.Write([]byte("Unknown command\n"))
	}
}

func generateHostKey() (ssh.Signer, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return nil, fmt.Errorf("failed to generate RSA key: %w", err)
	}

	return ssh.NewSignerFromKey(privateKey)
}

func init() {
	_ = io.ReadFull
}
