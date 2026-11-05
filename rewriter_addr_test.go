package main

import (
	"context"
	"net"
	"testing"

	"github.com/armon/go-socks5"
)

func TestAddRule(t *testing.T) {
	tests := []struct {
		rule    string
		wantErr bool
		want    map[string]RewriteDest
	}{
		{
			rule: "example.com:80:10.0.0.1",
			want: map[string]RewriteDest{"example.com:80": {net.ParseIP("10.0.0.1"), 80}},
		},
		{
			rule: "1.2.3.4:80:10.0.0.1:8080",
			want: map[string]RewriteDest{"1.2.3.4:80": {net.ParseIP("10.0.0.1"), 8080}},
		},
		{
			rule: "host:100-102:10.0.0.1:200",
			want: map[string]RewriteDest{
				"host:100": {net.ParseIP("10.0.0.1"), 200},
				"host:101": {net.ParseIP("10.0.0.1"), 201},
				"host:102": {net.ParseIP("10.0.0.1"), 202},
			},
		},
		{
			// Range ending at the max port must terminate
			rule: "host:65534-65535:10.0.0.1",
			want: map[string]RewriteDest{
				"host:65534": {net.ParseIP("10.0.0.1"), 65534},
				"host:65535": {net.ParseIP("10.0.0.1"), 65535},
			},
		},
		{rule: "host:100-101:10.0.0.1:65535", wantErr: true}, // destination overflow
		{rule: "host:102-100:10.0.0.1", wantErr: true},       // reversed range
		{rule: "host:80", wantErr: true},
		{rule: "host:abc:10.0.0.1", wantErr: true},
		{rule: "host:70000:10.0.0.1", wantErr: true},
		{rule: "host:80:not-an-ip", wantErr: true},
		{rule: "host:1-2-3:10.0.0.1", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.rule, func(t *testing.T) {
			r := NewRewriterAddr()
			err := r.AddRule(tt.rule)
			if (err != nil) != tt.wantErr {
				t.Fatalf("AddRule(%q) err = %v, wantErr %v", tt.rule, err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if len(r.Rules) != len(tt.want) {
				t.Fatalf("got %d rules, want %d: %v", len(r.Rules), len(tt.want), r.Rules)
			}
			for k, want := range tt.want {
				got, ok := r.Rules[k]
				if !ok || !got.IP.Equal(want.IP) || got.Port != want.Port {
					t.Errorf("rule %q = %v (present: %v), want %v", k, got, ok, want)
				}
			}
		})
	}
}

func TestRewrite(t *testing.T) {
	r := NewRewriterAddr()
	if err := r.AddRule("example.com:443:10.0.0.1:8443"); err != nil {
		t.Fatal(err)
	}

	addr := &socks5.AddrSpec{FQDN: "example.com", Port: 443}
	got := r.Rewrite(context.Background(), &socks5.Request{DestAddr: addr}, addr)
	if !got.IP.Equal(net.ParseIP("10.0.0.1")) || got.Port != 8443 {
		t.Errorf("Rewrite = %s:%d, want 10.0.0.1:8443", got.IP, got.Port)
	}

	addr = &socks5.AddrSpec{FQDN: "other.com", Port: 443}
	got = r.Rewrite(context.Background(), &socks5.Request{DestAddr: addr}, addr)
	if got.IP != nil || got.Port != 443 {
		t.Errorf("unmatched address was rewritten: %s:%d", got.IP, got.Port)
	}
}
