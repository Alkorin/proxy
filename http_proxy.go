package main

import (
	"context"
	"io"
	"log"
	"net"
	"net/http"
	"time"
)

type NameResolver interface {
	Resolve(ctx context.Context, name string) (context.Context, net.IP, error)
}

type httpProxy struct {
	Resolver NameResolver
	Verbose  bool
	Logger   *log.Logger
}

// Hop-by-hop headers that must not be forwarded by a proxy (RFC 9110 §7.6.1)
var hopHeaders = []string{
	"Connection",
	"Proxy-Connection",
	"Keep-Alive",
	"Proxy-Authenticate",
	"Proxy-Authorization",
	"Te",
	"Trailer",
	"Transfer-Encoding",
	"Upgrade",
}

func (p *httpProxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	_, ip, err := p.Resolver.Resolve(r.Context(), r.URL.Hostname())
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
	if p.Verbose {
		p.Logger.Printf("Command: %s, FQDN: %q, IP: %q, port: %s", r.Method, r.URL.Hostname(), ip.String(), r.URL.Port())
	}

	if r.Method == http.MethodConnect {
		p.handleConnect(w, r, ip)
	} else {
		p.handleHTTP(w, r, ip)
	}
}

func (p *httpProxy) handleConnect(w http.ResponseWriter, r *http.Request, ip net.IP) {
	destConn, err := net.DialTimeout("tcp", net.JoinHostPort(ip.String(), r.URL.Port()), 10*time.Second)
	if err != nil {
		p.Logger.Printf("Failed to dial: %s", err)
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}

	hijacker, ok := w.(http.Hijacker)
	if !ok {
		destConn.Close()
		http.Error(w, "Hijacking not supported", http.StatusInternalServerError)
		return
	}
	clientConn, _, err := hijacker.Hijack()
	if err != nil {
		destConn.Close()
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}

	if _, err := io.WriteString(clientConn, "HTTP/1.1 200 Connection established\r\n\r\n"); err != nil {
		destConn.Close()
		clientConn.Close()
		return
	}

	go func() {
		defer destConn.Close()
		io.Copy(destConn, clientConn)
	}()
	go func() {
		defer clientConn.Close()
		io.Copy(clientConn, destConn)
	}()
}

func (p *httpProxy) handleHTTP(w http.ResponseWriter, r *http.Request, ip net.IP) {
	outReq := r.Clone(r.Context())
	outReq.RequestURI = ""
	if port := r.URL.Port(); port != "" {
		outReq.URL.Host = net.JoinHostPort(ip.String(), port)
	} else if ip.To4() == nil {
		outReq.URL.Host = "[" + ip.String() + "]"
	} else {
		outReq.URL.Host = ip.String()
	}
	for _, h := range hopHeaders {
		outReq.Header.Del(h)
	}

	resp, err := http.DefaultTransport.RoundTrip(outReq)
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
	defer resp.Body.Close()

	for _, h := range hopHeaders {
		resp.Header.Del(h)
	}
	for k, vv := range resp.Header {
		for _, v := range vv {
			w.Header().Add(k, v)
		}
	}
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}
