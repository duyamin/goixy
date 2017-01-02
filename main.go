package main

import (
	"fmt"
	"flag"
	"os"
)

func main() {
	if len(os.Args) == 1 {
		fmt.Println("usage: goixy <local|remote> [<args>]")
		return
	}

	localCommand := flag.NewFlagSet("local", flag.ExitOnError)
	hostLocal := localCommand.String("host", "127.0.0.1", "host")
	portLocal := localCommand.String("port", "11080", "port")
	rhost := localCommand.String("rhost", "", "remote host")
	rport := localCommand.String("rport", "", "remote port")
	debugLocal := localCommand.Bool("debug", false, "debug")

	remoteCommand := flag.NewFlagSet("remote", flag.ExitOnError)
	hostRemote := remoteCommand.String("host", "", "host")
	portRemote := remoteCommand.String("port", "", "port")
	debugRemote := remoteCommand.Bool("debug", false, "debug")

	switch os.Args[1] {
	case "local":
		localCommand.Parse(os.Args[2:])
		runLocal(*hostLocal, *portLocal, *rhost, *rport, *debugLocal)
	case "remote":
		localCommand.Parse(os.Args[2:])
		runRemote(*hostRemote, *portRemote, *debugRemote)
	default:
		fmt.Printf("%q is not valid command.\n", os.Args[1])
		os.Exit(1)
	}
}
