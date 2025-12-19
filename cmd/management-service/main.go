package main

import (
	"context"
	"fmt"
	"github.com/spf13/cobra"
	"os"
	"os/signal"
	"syscall"
	_ "yeti/cmd/management-service/docs"

	"yeti/internal/config"
	"yeti/internal/logger"
	"yeti/pkg/logging"
)

var (
	configFile string
)

// @title           Yeti Management Service API
// @version         1.0
// @description     REST API for managing filtering rules, enrichment rules, and deduplication configuration
// @termsOfService  http://swagger.io/terms/

// @contact.name   API Support
// @contact.url    http://www.example.com/support
// @contact.email  support@example.com

// @license.name  Apache 2.0
// @license.url   http://www.apache.org/licenses/LICENSE-2.0.html

// @host      localhost:8080
// @BasePath  /api/v1

// @schemes   http https

func main() {
	rootCmd := &cobra.Command{
		Use:   "management-service",
		Short: "Management Service for data pipeline",
		Long:  "Management Service provides REST API for managing rules and configuration",
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
		Short: "Start the management service",
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

			log.InfowCtx(ctx, "Starting Management Service")

			app := NewApp(cfg, log)
			if err := app.Initialize(ctx); err != nil {
				log.Fatalf("Failed to initialize application: %v", err)
			}

			if err := app.Run(ctx); err != nil {
				log.ErrorwCtx(ctx, "Application error", "error", err)
				return err
			}
			return nil
		},
	}
}
