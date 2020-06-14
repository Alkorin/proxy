package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/armon/go-socks5"
)

var ErrInvalidErrorRule = errors.New("Invalid format, should be (IP|FQDN):PORT:IP[:PORT]")

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

func (r *RewriterAddr) AddRule(rule string) error {
	splittedRule := strings.Split(rule, ":")
	if len(splittedRule) != 3 && len(splittedRule) != 4 {
		return ErrInvalidErrorRule
	}

	srcPort, err := strconv.ParseUint(splittedRule[1], 10, 16)
	if err != nil {
		return fmt.Errorf("Invalid source port %q: %s", splittedRule[1], err.Error())
	}

	// Parse Destination
	var dst RewriteDest
	ip := net.ParseIP(splittedRule[2])
	if ip == nil {
		return fmt.Errorf("Invalid destination IP %q", splittedRule[2])
	}
	dst.IP = ip
	if len(splittedRule) == 4 {
		dstPort, err := strconv.ParseUint(splittedRule[3], 10, 16)
		if err != nil {
			return fmt.Errorf("Invalid destination ort %q: %s", splittedRule[3], err.Error())
		}
		dst.Port = int(dstPort)
	} else {
		dst.Port = int(srcPort)
	}

	r.Rules[fmt.Sprintf("%s:%d", splittedRule[0], srcPort)] = dst

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
