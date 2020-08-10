package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"

	"github.com/armon/go-socks5"
)

var ErrInvalidErrorRule = errors.New("Invalid format, should be (IP|FQDN):PORT[-PORT]:IP[:PORT]")

type RewriterAddr struct {
	Rules map[string]RewriteDest
}

type RewriteDest struct {
	IP   net.IP
	Port int
}

func NewRewriterAddr() *RewriterAddr {
	return &RewriterAddr{
		Rules: make(map[string]RewriteDest),
	}
}

// AddRule parse and import a rule into the map
// Format: (IP|FQDN):PORT[-PORT]:IP[:PORT]
func (r *RewriterAddr) AddRule(rule string) error {
	splittedRule := strings.Split(rule, ":")
	if len(splittedRule) != 3 && len(splittedRule) != 4 {
		return ErrInvalidErrorRule
	}

	// Parse source port (PORT[-PORT])
	splittedSrcPort := strings.Split(splittedRule[1], "-")
	if len(splittedSrcPort) != 1 && len(splittedSrcPort) != 2 {
		return ErrInvalidErrorRule
	}

	srcPortBegin, err := ParseUint16(splittedSrcPort[0])
	if err != nil {
		return fmt.Errorf("Invalid source port %q: %s", splittedRule[1], err.Error())
	}

	srcPortEnd := srcPortBegin
	if len(splittedSrcPort) == 2 {
		port, err := ParseUint16(splittedSrcPort[1])
		if err != nil {
			return fmt.Errorf("Invalid end source port %q: %s", splittedRule[1], err.Error())
		}
		if srcPortBegin > port {
			return fmt.Errorf("Invalid end source port %q: should be lower than %d", splittedRule[1], srcPortBegin)
		}
		srcPortEnd = uint16(port)
	}

	// Parse Destination
	ip := net.ParseIP(splittedRule[2])
	if ip == nil {
		return fmt.Errorf("Invalid destination IP %q", splittedRule[2])
	}
	dstPortBegin := srcPortBegin
	if len(splittedRule) == 4 {
		port, err := ParseUint16(splittedRule[3])
		if err != nil {
			return fmt.Errorf("Invalid destination port %q: %s", splittedRule[3], err.Error())
		}
		dstPortBegin = port
	}

	for srcPort, dstPort := srcPortBegin, dstPortBegin; srcPort <= srcPortEnd; srcPort, dstPort = srcPort+1, dstPort+1 {
		r.Rules[fmt.Sprintf("%s:%d", splittedRule[0], srcPort)] = RewriteDest{
			IP:   ip,
			Port: int(dstPort),
		}
	}

	return nil
}

func (r *RewriterAddr) Rewrite(ctx context.Context, request *socks5.Request, addr *socks5.AddrSpec) *socks5.AddrSpec {
	var key string

	if addr.FQDN != "" {
		key = fmt.Sprintf("%s:%d", addr.FQDN, addr.Port)
	} else {
		key = fmt.Sprintf("%s:%d", addr.IP.String(), addr.Port)
	}

	if dest, ok := r.Rules[key]; ok {
		addr.IP = dest.IP
		addr.Port = dest.Port
	}

	return addr
}
