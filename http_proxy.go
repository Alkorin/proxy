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

func (p *httpProxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	_, ip, err := p.Resolver.Resolve(r.Context(), r.URL.Hostname())
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
	p.Logger.Printf("Command: %s, FQDN: %q, IP: %q, port: %s", r.Method, r.URL.Hostname(), ip.String(), r.URL.Port())

	if r.Method == http.MethodConnect {
		dest_conn, err := net.DialTimeout("tcp", ip.String()+":"+r.URL.Port(), 10*time.Second)
		if err != nil {
			log.Printf("Faild to dial: %s", err.Error())
			http.Error(w, err.Error(), http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
		hijacker, ok := w.(http.Hijacker)
		if !ok {
			http.Error(w, "Hijacking not supported", http.StatusInternalServerError)
			return
		}
		client_conn, _, err := hijacker.Hijack()
		if err != nil {
			http.Error(w, err.Error(), http.StatusServiceUnavailable)
		}
		go func() {
			defer dest_conn.Close()
			io.Copy(dest_conn, client_conn)
		}()
		go func() {
			defer client_conn.Close()
			io.Copy(client_conn, dest_conn)
		}()
	} else {
		r.URL.Host = ip.String()
		resp, err := http.DefaultTransport.RoundTrip(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusServiceUnavailable)
			return
		}
		defer resp.Body.Close()
		for k, vv := range resp.Header {
			for _, v := range vv {
				w.Header().Add(k, v)
			}
		}
		w.WriteHeader(resp.StatusCode)
		io.Copy(w, resp.Body)
	}
}
