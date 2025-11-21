package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
	"yeti/internal/config"
	"yeti/internal/logger"
	"yeti/pkg/logging"
)

var (
	configFile string
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "filtering-service",
		Short: "Filtering Service for data pipeline",
		Long:  "Filtering Service processes messages and applies filtering rules",
		RunE:  serveCmd().RunE,
	}

	rootCmd.PersistentFlags().StringVar(&configFile, "config", "", "Path to config file (required)")

	rootCmd.AddCommand(serveCmd())

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func serveCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "serve",
		Short: "Start the filtering service",
		RunE: func(cmd *cobra.Command, args []string) error {
			earlyLog := logging.NewEarlyLog()

			if configFile == "" {
				configFile = os.Getenv("CONFIG_FILE")
				if configFile == "" {
					earlyLog.Error("Config file is required. Use --config flag or CONFIG_FILE environment variable")
					return fmt.Errorf("config file is required")
				}
			}

			cfg, err := config.Load(configFile)
			if err != nil {
				earlyLog.Error("Failed to load config: %v", err)
				return err
			}

			log, err := logger.New(cfg.Logging.Level)
			if err != nil {
				earlyLog.Error("Failed to init logger: %v", err)
				return err
			}
			defer log.Sync()

			ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
			defer cancel()

			log.InfowCtx(ctx, "Starting Filtering Service")

			app := NewApp(cfg, log)
			if err := app.Initialize(ctx); err != nil {
				log.Fatalf("Failed to initialize application: %v", err)
			}

			log.InfowCtx(ctx, "Service running")
			if err := app.Run(ctx); err != nil && err != context.Canceled {
				log.ErrorwCtx(ctx, "Service stopped with error", "error", err)
				return err
			}
			log.InfowCtx(ctx, "Service shutdown complete")
			return nil
		},
	}
}
