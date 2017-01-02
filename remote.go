package main

import (
	"net"
	"fmt"
)

func runRemote(host, port string, debug bool) {
	server, err := net.Listen("tcp", host + ":" + port)
	check(err)
	defer server.Close()
	fmt.Printf("listen on port %s:%s\n", host, port)

	for {
        local, err := server.Accept()
        if err != nil {
            continue
        }
        go handleLocal(local)
    }
}

func handleLocal(local net.Conn) {
    defer local.Close()
}
