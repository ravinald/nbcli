// Command nbcli is the modern Netbox CLI + TUI.
//
// Build with -ldflags to embed version metadata:
//
//	go build -ldflags "-X github.com/ravinald/nbcli/internal/version.Version=v0.1.0 \
//	                   -X github.com/ravinald/nbcli/internal/version.Commit=$(git rev-parse --short HEAD) \
//	                   -X github.com/ravinald/nbcli/internal/version.Date=$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
//	    -o bin/nbcli ./cmd/nbcli
package main

import (
	"os"

	"github.com/ravinald/nbcli/internal/cmd"
)

func main() {
	os.Exit(cmd.Execute(os.Args[1:], cmd.StdIO()))
}
