//go:build testserver

// Command adt-testserver is a standalone HTTP server that emulates the SAP ADT REST API
// for testing purposes. Routes and fixture data are loaded from YAML fixture files
// specified as mandatory positional glob patterns.
//
// Usage:
//
//	go run ./testserver [flags] <glob> [<glob>...]
//
// Flags:
//
//	--sys-id    SAP System ID reported by the server (default: TST)
//	--client    SAP client number (default: 001)
//	--user      Expected Basic Auth username (default: developer)
//	--password  Expected Basic Auth password (default: secret)
//	--port      TCP port to listen on (default: 8080)
package main

import (
	"flag"
	"log"

	"github.com/oisee/vibing-steampunk/internal/testserver"
)

func main() {
	sysID := flag.String("sys-id", "TST", "SAP System ID")
	client := flag.String("client", "001", "SAP Client")
	user := flag.String("user", "developer", "SAP Username")
	password := flag.String("password", "secret", "SAP Password")
	port := flag.Int("port", 3000, "HTTP port to listen on")
	flag.Parse()

	globs := flag.Args()
	if len(globs) == 0 {
		log.Fatal("usage: adt-testserver [flags] <glob> [<glob>...]\n  at least one fixture glob pattern is required")
	}

	testserver.StartTestServer(sysID, client, user, password, globs, port)
}
