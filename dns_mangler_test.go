package main

import (
	"context"
	"net"
	"testing"
)

func TestDNSMangler(t *testing.T) {
	d := NewDNSMangler()
	if err := d.AddHost("example.com", "10.0.0.1"); err != nil {
		t.Fatal(err)
	}
	if err := d.AddHost("v6.example.com", "::1"); err != nil {
		t.Fatal(err)
	}
	if err := d.AddHost("bad.example.com", "not-an-ip"); err == nil {
		t.Error("expected error for invalid IP")
	}

	for host, want := range map[string]string{
		"example.com":    "10.0.0.1",
		"v6.example.com": "::1",
		"127.0.0.1":      "127.0.0.1",
	} {
		_, ip, err := d.Resolve(context.Background(), host)
		if err != nil {
			t.Fatalf("Resolve(%q): %v", host, err)
		}
		if !ip.Equal(net.ParseIP(want)) {
			t.Errorf("Resolve(%q) = %s, want %s", host, ip, want)
		}
	}
}
