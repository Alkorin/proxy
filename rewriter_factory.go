package main

import (
	"context"

	"github.com/armon/go-socks5"
)

type RewriterFactory struct {
	rewriters []Rewriter
}

type Rewriter interface {
	Rewrite(ctx context.Context, request *socks5.Request, addr *socks5.AddrSpec) *socks5.AddrSpec
}

var _ socks5.AddressRewriter = &RewriterFactory{}

func (f *RewriterFactory) AddRewriter(r Rewriter) {
	f.rewriters = append(f.rewriters, r)
}

func (f *RewriterFactory) Rewrite(ctx context.Context, request *socks5.Request) (context.Context, *socks5.AddrSpec) {
	addr := request.DestAddr

	for _, r := range f.rewriters {
		addr = r.Rewrite(ctx, request, addr)
	}

	// Noop
	return ctx, addr
}
