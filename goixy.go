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
	var verbose = flag.Bool("verbose", false, "verbose")
	var user = flag.String("user", "goixy", "user")
	var password = flag.String("password", "goixy-secret", "password")
	flag.Usage = goixyUsage
	flag.Parse()

	proxy := goproxy.NewProxyHttpServer()
	proxy.Verbose = *verbose
	auth_handler := auth.BasicConnect(
		"goixy v0.1.0",
		func(u, p string) bool {
			return u == *user && p == *password
	})
	proxy.OnRequest().HandleConnect(auth_handler)
	fmt.Printf("Listen on port: %s... \n", *port)
	http.ListenAndServe(":" + *port, proxy)
}

func goixyUsage() {
     fmt.Printf("Usage: %s [OPTIONS] argument ...\n", os.Args[0])
     flag.PrintDefaults()
}
