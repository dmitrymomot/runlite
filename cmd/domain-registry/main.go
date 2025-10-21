package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/dmitrymomot/runlite/internal/registry"
)

var version = "dev"

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	logger.Info("starting domain registry service", "version", version)

	ctx := context.Background()

	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}

	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "/var/lib/runlite/domains.db"
	}

	storage, err := registry.NewStorage(ctx, logger, dbPath)
	if err != nil {
		logger.Error("failed to initialize storage", "error", err)
		os.Exit(1)
	}
	defer func() {
		if err := storage.Close(); err != nil {
			logger.Error("failed to close storage", "error", err)
		}
	}()

	api := registry.NewAPI(storage, logger)

	addr := fmt.Sprintf("127.0.0.1:%s", port)
	if err := api.Start(ctx, addr); err != nil {
		logger.Error("server stopped with error", "error", err)
		os.Exit(1)
	}

	logger.Info("domain registry service stopped")
}
