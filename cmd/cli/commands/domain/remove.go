package domain

import (
	"context"
	"fmt"

	"github.com/urfave/cli/v3"

	"github.com/dmitrymomot/runlite/cmd/cli/internal/ui"
)

func removeCommand() *cli.Command {
	return &cli.Command{
		Name:      "remove",
		Usage:     "Remove a domain",
		ArgsUsage: "DOMAIN",
		Action:    removeAction,
	}
}

func removeAction(ctx context.Context, cmd *cli.Command) error {
	if cmd.Args().Len() != 1 {
		return cli.Exit(ui.ErrorString("Usage: domain remove DOMAIN"), 1)
	}

	domain := cmd.Args().Get(0)

	client := newClient()
	if err := client.removeDomain(ctx, domain); err != nil {
		return cli.Exit(ui.ErrorString(fmt.Sprintf("Failed to remove domain: %v", err)), 1)
	}

	ui.PrintSuccess(fmt.Sprintf("Domain %s removed successfully", domain))
	return nil
}
