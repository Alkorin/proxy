package main

import (
	"context"
	"log"

	"github.com/armon/go-socks5"
)

type RewriterLogger struct {
	logger *log.Logger
}

func NewRewriterLogger(logger *log.Logger) *RewriterLogger {
	return &RewriterLogger{
		logger: logger,
	}
}

func (l *RewriterLogger) Rewrite(ctx context.Context, request *socks5.Request, addr *socks5.AddrSpec) *socks5.AddrSpec {
	// Log
	var cmdName string
	switch request.Command {
	case socks5.ConnectCommand:
		cmdName = "Connect"
	case socks5.BindCommand:
		cmdName = "Bind"
	case socks5.AssociateCommand:
		cmdName = "Associate"
	default:
		cmdName = "unknown"
	}
	l.logger.Printf("Command: %s, FQDN: %q, IP: %q, port: %d", cmdName, request.DestAddr.FQDN, request.DestAddr.IP, request.DestAddr.Port)

	// Noop
	return addr
}
