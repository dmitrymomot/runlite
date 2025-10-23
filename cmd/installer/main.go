package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/charmbracelet/fang"

	"github.com/dmitrymomot/runlite/cmd/installer/commands"
)

func main() {
	// Setup context with signal handling
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	// Execute root command with fang
	if err := fang.Execute(ctx, commands.Root()); err != nil {
		os.Exit(1)
	}
}
