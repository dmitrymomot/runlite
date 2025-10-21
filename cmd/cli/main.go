package main

import (
	"context"
	"os"

	"github.com/urfave/cli/v3"

	"github.com/dmitrymomot/runlite/cmd/cli/commands/app"
	"github.com/dmitrymomot/runlite/cmd/cli/commands/deploy"
	"github.com/dmitrymomot/runlite/cmd/cli/commands/domain"
	"github.com/dmitrymomot/runlite/cmd/cli/commands/env"
	"github.com/dmitrymomot/runlite/cmd/cli/commands/logs"
)

// version can be set at build time using:
// go build -ldflags "-X main.version=v1.0.0" -o runlite cmd/cli/main.go
var version = "dev"

func main() {
	cmd := &cli.Command{
		Name:    "runlite",
		Version: version,
		Usage:   "Lightweight deployment platform for Go applications",
		Commands: []*cli.Command{
			app.Command(),
			env.Command(),
			deploy.Command(),
			domain.Command(),
			logs.Command(),
		},
	}

	// Commands print their own error messages, so just exit on error
	if err := cmd.Run(context.Background(), os.Args); err != nil {
		os.Exit(1)
	}
}
