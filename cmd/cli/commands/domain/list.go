package domain

import (
	"context"
	"fmt"

	"github.com/urfave/cli/v3"

	"github.com/dmitrymomot/runlite/cmd/cli/internal/ui"
)

func listCommand() *cli.Command {
	return &cli.Command{
		Name:   "list",
		Usage:  "List all domains",
		Action: listAction,
	}
}

func listAction(ctx context.Context, cmd *cli.Command) error {
	client := newClient()

	domains, err := client.listDomains(ctx)
	if err != nil {
		return cli.Exit(ui.ErrorString(fmt.Sprintf("Failed to list domains: %v", err)), 1)
	}

	if len(domains) == 0 {
		ui.PrintInfo("No domains registered")
		return nil
	}

	for _, domain := range domains {
		status := "✓"
		statusColor := ui.ColorSuccess

		if !domain.Verified {
			status = "⚠"
			statusColor = ui.ColorWarning
		}

		fmt.Printf("%s %s", statusColor(status), domain.Name)

		if domain.AppName != "" {
			fmt.Printf(" %s→ %s%s", ui.ColorGray(""), ui.ColorInfo(domain.AppName), ui.ColorGray(""))
		}

		fmt.Println()
	}

	return nil
}
