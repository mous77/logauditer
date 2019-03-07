package main

import (
	"flag"
	"fmt"
	"os"

	"logauditer/api"
	cli "logauditer/client"
)

//Values populated by the Go linker.
var (
	version = "v0.0.1"
	commit  = "v0.0.1"
	date    = "20181217"
	client  api.LogAuditerClient
)

var hosts = flag.String("host", "127.0.0.1:9992", "Host to connect to a server.")

var showVersion = flag.Bool("version", false, "Show logAuditer version.")

func main() {
	flag.Parse()

	if *showVersion {
		fmt.Printf("version: %s\ncommit: %s\nbuildtime: %s", version, commit, date)
		os.Exit(0)
	}

	if err := cli.Run(*hosts); err != nil {
		fmt.Fprintf(os.Stderr, "could not run CLI: %v", err)
	}
}
