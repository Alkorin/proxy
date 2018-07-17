package main

import (
	"net"

	"github.com/armon/go-socks5"
	"github.com/pkg/errors"
	"golang.org/x/net/context"
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
		return errors.Errorf("failed to parse ip: %q", ip)
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
	addr, err := net.ResolveIPAddr("ip", name)
	if err != nil {
		return ctx, nil, err
	}

	return ctx, addr.IP, nil
}
