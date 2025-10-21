package domain

import (
	"context"
	"fmt"

	"github.com/urfave/cli/v3"

	"github.com/dmitrymomot/runlite/cmd/cli/internal/ui"
)

func verifyCommand() *cli.Command {
	return &cli.Command{
		Name:      "verify",
		Usage:     "Verify a domain",
		ArgsUsage: "DOMAIN",
		Action:    verifyAction,
	}
}

func verifyAction(ctx context.Context, cmd *cli.Command) error {
	if cmd.Args().Len() != 1 {
		return cli.Exit(ui.ErrorString("Usage: domain verify DOMAIN"), 1)
	}

	domain := cmd.Args().Get(0)

	client := newClient()
	verified, err := client.verifyDomain(ctx, domain)
	if err != nil {
		return cli.Exit(ui.ErrorString(fmt.Sprintf("Failed to verify domain: %v", err)), 1)
	}

	if verified {
		ui.PrintSuccess(fmt.Sprintf("Domain %s verified successfully", domain))
	} else {
		ui.PrintWarning(fmt.Sprintf("Domain %s verification failed", domain))
	}

	return nil
}
