package main

import (
	"flag"
	"fmt"
	"github.com/elazarl/goproxy"
	"github.com/elazarl/goproxy/ext/auth"
	"net/http"
	"os"
)

func main() {
	var port = flag.String("port", "18080", "port")
	var verbose = flag.Bool("v", false, "verbose")
	var user = flag.String("user", "goixy", "user")
	var password = flag.String("password", "goixy-secret", "password")
	flag.Usage = goixyUsage
	flag.Parse()

	proxy := goproxy.NewProxyHttpServer()
	proxy.Verbose = *verbose
	auth_conn := auth.BasicConnect(
		"goixy v0.1.0",
		func(u, p string) bool {
			return u == *user && p == *password
	})
	proxy.OnRequest().HandleConnect(auth_conn)
	auth_do := auth.Basic(
		"my_realm", func(u, p string) bool {
			return u == *user && p == *password
	})
	proxy.OnRequest().Do(auth_do)
	fmt.Printf("Listen on port: %s... \n", *port)
	http.ListenAndServe(":" + *port, proxy)
}

func goixyUsage() {
     fmt.Printf("Usage: %s [OPTIONS] argument ...\n", os.Args[0])
     flag.PrintDefaults()
}
