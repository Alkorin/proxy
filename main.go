package main

import (
	"flag"
	"fmt"
	"log"
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
var verbose = false

func init() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage of %s (Version %s):\n", os.Args[0], buildVersion)
		flag.PrintDefaults()
	}
	flag.StringVar(&listen, "listen", listen, "Address on which the server will listen")
	flag.Var(&dnsManglingList, "resolve", "Provide a custom address for a specific host and port pair. This option can be used many times")
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

	var rewriter socks5.AddressRewriter
	if verbose {
		rewriter = NewRewriterLogger(log.New(os.Stderr, "", log.LstdFlags))
	}

	// Instanciate socks proxy
	conf := &socks5.Config{
		Resolver: dnsMangler,
		Rewriter: rewriter,
	}
	server, err := socks5.New(conf)
	if err != nil {
		log.Fatal(err)
	}

	// Start server
	log.Printf("Start listening on %q", listen)
	if err := server.ListenAndServe("tcp", listen); err != nil {
		log.Fatal(err)
	}
}
