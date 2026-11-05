package main

import (
	"context"
	"fmt"
	"net"

	"github.com/armon/go-socks5"
)

type DNSMangler struct {
	hosts map[string]net.IP
}

var _ socks5.NameResolver = &DNSMangler{}

func NewDNSMangler() *DNSMangler {
	return &DNSMangler{
		hosts: make(map[string]net.IP),
	}
}

func (d *DNSMangler) AddHost(host string, ipString string) error {
	ip := net.ParseIP(ipString)
	if ip == nil {
		return fmt.Errorf("failed to parse ip: %q", ipString)
	}

	d.hosts[host] = ip

	return nil
}

func (d *DNSMangler) Resolve(ctx context.Context, name string) (context.Context, net.IP, error) {
	// Try our mangled results
	if ip, ok := d.hosts[name]; ok {
		return ctx, ip, nil
	}

	// Resolve name
	addr, err := net.DefaultResolver.LookupIPAddr(ctx, name)
	if err != nil {
		return ctx, nil, err
	}
	if len(addr) == 0 {
		return ctx, nil, fmt.Errorf("no address found for %q", name)
	}

	return ctx, addr[0].IP, nil
}
