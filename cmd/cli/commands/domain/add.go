package domain

import (
	"context"
	"fmt"

	"github.com/urfave/cli/v3"

	"github.com/dmitrymomot/runlite/cmd/cli/internal/ui"
)

func addCommand() *cli.Command {
	return &cli.Command{
		Name:      "add",
		Usage:     "Add a new domain",
		ArgsUsage: "DOMAIN",
		Action:    addAction,
	}
}

func addAction(ctx context.Context, cmd *cli.Command) error {
	if cmd.Args().Len() != 1 {
		return cli.Exit(ui.ErrorString("Usage: domain add DOMAIN"), 1)
	}

	domain := cmd.Args().Get(0)

	client := newClient()
	if err := client.addDomain(ctx, domain); err != nil {
		return cli.Exit(ui.ErrorString(fmt.Sprintf("Failed to add domain: %v", err)), 1)
	}

	ui.PrintSuccess(fmt.Sprintf("Domain %s added successfully", domain))
	return nil
}
