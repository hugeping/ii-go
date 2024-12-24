// some code taken from https://github.com/yi-jiayu/secure
// secure is a super simple TLS termination proxy
package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path/filepath"
	"time"
	"golang.org/x/crypto/acme/autocert"
)

var (
	upstream string
	addr     string
)

func init() {
	flag.StringVar(&addr, "addr", ":443", "listen address")

	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(),
			"usage: %s [-addr host:port] upstream\n",
			filepath.Base(os.Args[0]))
		flag.PrintDefaults()
		fmt.Fprintln(flag.CommandLine.Output(), "  upstream string\n    \tupstream url")
	}
}

func main() {
	flag.Parse()

	if flag.NArg() == 1 {
		upstream = flag.Arg(0)
	} else {
		flag.Usage()
		os.Exit(2)
	}

	u, err := url.Parse(upstream)
	if err != nil {
		fmt.Printf("invalid upstream address: %s", err)
		os.Exit(1)
	}

	rp := httputil.NewSingleHostReverseProxy(u)

	certManager := autocert.Manager{
		Prompt:     autocert.AcceptTOS,
		Cache:      autocert.DirCache("certs"),
	}

	tlsConfig := certManager.TLSConfig()

	srv := http.Server{
		Handler: rp,
		TLSConfig: tlsConfig,
		Addr: addr,
		ReadTimeout:       5 * time.Second,
		WriteTimeout:      5 * time.Second,
		IdleTimeout:       8 * time.Second,
		ReadHeaderTimeout: 2 * time.Second,
	}

	log.Printf("listen-addr=%s upstream-url=%s", srv.Addr, u.String())

	srv.ListenAndServeTLS("", "")
}
