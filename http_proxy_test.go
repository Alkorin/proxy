package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func newTestProxy(t *testing.T, host, ip string) *httptest.Server {
	t.Helper()
	d := NewDNSMangler()
	if err := d.AddHost(host, ip); err != nil {
		t.Fatal(err)
	}
	p := httptest.NewServer(&httpProxy{Resolver: d, Logger: log.New(io.Discard, "", 0)})
	t.Cleanup(p.Close)
	return p
}

func TestHTTPProxyForward(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Backend", "yes")
		fmt.Fprintf(w, "hello %s", r.URL.Path)
	}))
	defer backend.Close()
	_, port, _ := net.SplitHostPort(backend.Listener.Addr().String())

	proxy := newTestProxy(t, "mangled.test", "127.0.0.1")
	proxyURL, _ := url.Parse(proxy.URL)
	client := &http.Client{Transport: &http.Transport{Proxy: http.ProxyURL(proxyURL)}}

	// Non-default port must be preserved
	resp, err := client.Get("http://mangled.test:" + port + "/path")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if string(body) != "hello /path" || resp.Header.Get("X-Backend") != "yes" {
		t.Errorf("unexpected response: %q, headers %v", body, resp.Header)
	}
}

func TestHTTPProxyConnect(t *testing.T) {
	// Simple echo server
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	go func() {
		c, err := ln.Accept()
		if err != nil {
			return
		}
		defer c.Close()
		io.Copy(c, c)
	}()
	_, port, _ := net.SplitHostPort(ln.Addr().String())

	proxy := newTestProxy(t, "mangled.test", "127.0.0.1")
	conn, err := (&net.Dialer{}).DialContext(context.Background(), "tcp", proxy.Listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	fmt.Fprintf(conn, "CONNECT mangled.test:%s HTTP/1.1\r\nHost: mangled.test:%s\r\n\r\n", port, port)
	br := bufio.NewReader(conn)
	resp, err := http.ReadResponse(br, &http.Request{Method: http.MethodConnect})
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("CONNECT status = %d", resp.StatusCode)
	}

	fmt.Fprint(conn, "ping\n")
	line, err := br.ReadString('\n')
	if err != nil || strings.TrimSpace(line) != "ping" {
		t.Fatalf("echo = %q, err %v", line, err)
	}
}
