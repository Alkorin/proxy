package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/armon/go-socks5"
)

// Build details
var buildVersion = "dev"
var buildCommit = "unknown"
var buildDate = "unknown"

// Config
var listen = "127.0.0.1:8000"
var dnsManglingList stringArray
var rewriteList stringArray
var verbose = false
var mode string

func init() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage of %s (Version %s):\n", os.Args[0], buildVersion)
		flag.PrintDefaults()
	}
	flag.StringVar(&listen, "listen", listen, "Address on which the server will listen")
	flag.Var(&dnsManglingList, "resolve", "Provide a custom IP address for a specific host. This option can be used many times")
	flag.Var(&rewriteList, "rewrite", "Provide a custom IP:Port for a specific IP/FQDN:port. Format: (IP|FQDN):PORT:IP[:PORT]")
	flag.StringVar(&mode, "mode", "socks5", "Proxy mode, allowed values: http, socks5")
	flag.BoolVar(&verbose, "verbose", verbose, "Display access logs")
	flag.Parse()

	if flag.NArg() > 0 {
		if flag.Arg(0) == "version" {
			fmt.Fprintf(os.Stderr, "DNSMangler Proxy version %s (%s - %s)\n", buildVersion, buildCommit, buildDate)
			os.Exit(0)
		} else if flag.Arg(0) == "help" {
			flag.Usage()
			os.Exit(0)
		} else {
			fmt.Printf("Invalid command %q\n", flag.Arg(0))
			flag.Usage()
			os.Exit(-1)
		}
	}
}

func main() {
	log.Printf("DNSMangler Proxy version %s", buildVersion)

	// Instanciate our DNS mangler
	dnsMangler := NewDNSMangler()

	for _, v := range dnsManglingList {
		hostAndIP := strings.Split(v, ":")
		if len(hostAndIP) != 2 {
			log.Fatalf("Invalid resolve value %q, should be in %q format", v, "host:ip")
		}
		err := dnsMangler.AddHost(hostAndIP[0], hostAndIP[1])
		if err != nil {
			log.Fatalf(err.Error())
		}
		log.Printf(" - Will resolve %q to %q", hostAndIP[0], hostAndIP[1])
	}

	logger := log.New(os.Stderr, "", log.LstdFlags)

	if mode == "socks5" {
		rewriterFactory := &RewriterFactory{}

		if len(rewriteList) > 0 {
			rewriter := NewRewriterAddr()
			for _, v := range rewriteList {
				err := rewriter.AddRule(v)
				if err != nil {
					log.Fatalf("Failed to parse rewrite %q: %s", v, err.Error())
				}
			}

			for k, v := range rewriter.Rules {
				log.Printf(" - Will rewrite %q to %s:%d", k, v.IP, v.Port)
			}
			rewriterFactory.AddRewriter(rewriter)
		}

		if verbose {
			rewriterFactory.AddRewriter(NewRewriterLogger(logger))
		}

		// Instanciate socks proxy
		conf := &socks5.Config{
			Resolver: dnsMangler,
			Rewriter: rewriterFactory,
		}
		server, err := socks5.New(conf)
		if err != nil {
			log.Fatal(err)
		}

		// Start server
		log.Printf("Start listening on %q - mode: SOCKS5", listen)
		if err := server.ListenAndServe("tcp", listen); err != nil {
			log.Fatal(err)
		}
	} else if mode == "http" {
		server := &http.Server{
			Addr: listen,
			Handler: &httpProxy{
				Resolver: dnsMangler,
				Verbose:  verbose,
				Logger:   logger,
			},
			TLSNextProto: make(map[string]func(*http.Server, *tls.Conn, http.Handler)),
		}
		log.Printf("Start listening on %q - mode: HTTP", listen)
		if err := server.ListenAndServe(); err != nil {
			log.Fatal(err)
		}
	} else {
		log.Fatalf("Wrong mode: %s", mode)
	}
}
