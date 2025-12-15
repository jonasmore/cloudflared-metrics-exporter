package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/rs/zerolog"
	"github.com/urfave/cli/v2"
)

var (
	Version   = "dev"
	BuildTime = "unknown"
)

func main() {
	app := &cli.App{
		Name:    "cloudflared-metrics-exporter",
		Usage:   "Export Cloudflare Tunnel metrics to JSONL format",
		Version: Version,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "metrics",
				Usage:    "Metrics endpoint address (e.g., localhost:2000 or http://localhost:2000)",
				Required: true,
				EnvVars:  []string{"METRICS_ENDPOINT"},
			},
			&cli.StringFlag{
				Name:     "metricsfile",
				Usage:    "Path to the JSONL file where metrics will be saved",
				Required: true,
				EnvVars:  []string{"METRICS_FILE"},
			},
			&cli.DurationFlag{
				Name:    "metricsinterval",
				Usage:   "How frequently to export metrics",
				Value:   60 * time.Second,
				EnvVars: []string{"METRICS_INTERVAL"},
			},
			&cli.StringFlag{
				Name:    "metricsfilter",
				Usage:   "Comma-separated list of metric name patterns to export. Supports wildcards (*). If not set, all metrics are exported.",
				EnvVars: []string{"METRICS_FILTER"},
			},
			&cli.BoolFlag{
				Name:    "metricscompress",
				Usage:   "Enable change-only export mode. Only writes metrics when their value changes, significantly reducing file size.",
				EnvVars: []string{"METRICS_COMPRESS"},
			},
			&cli.StringFlag{
				Name:    "log-level",
				Usage:   "Log level (debug, info, warn, error)",
				Value:   "info",
				EnvVars: []string{"LOG_LEVEL"},
			},
			&cli.StringFlag{
				Name:    "cf-access-client-id",
				Usage:   "Cloudflare Access service token Client ID for authenticated metrics endpoints",
				EnvVars: []string{"CF_ACCESS_CLIENT_ID"},
			},
			&cli.StringFlag{
				Name:    "cf-access-client-secret",
				Usage:   "Cloudflare Access service token Client Secret for authenticated metrics endpoints",
				EnvVars: []string{"CF_ACCESS_CLIENT_SECRET"},
			},
		},
		Action: run,
	}

	if err := app.Run(os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run(c *cli.Context) error {
	// Setup logger
	log := setupLogger(c.String("log-level"))

	log.Info().
		Str("version", Version).
		Str("buildTime", BuildTime).
		Msg("Starting cloudflared-metrics-exporter")

	// Parse metrics endpoint
	metricsEndpoint := c.String("metrics")
	if !strings.HasPrefix(metricsEndpoint, "http://") && !strings.HasPrefix(metricsEndpoint, "https://") {
		metricsEndpoint = "http://" + metricsEndpoint
	}
	// Ensure /metrics path
	if !strings.HasSuffix(metricsEndpoint, "/metrics") {
		metricsEndpoint = strings.TrimSuffix(metricsEndpoint, "/") + "/metrics"
	}

	// Parse filter patterns
	var filterPatterns []string
	if filterStr := c.String("metricsfilter"); filterStr != "" {
		for _, pattern := range strings.Split(filterStr, ",") {
			pattern = strings.TrimSpace(pattern)
			if pattern != "" {
				filterPatterns = append(filterPatterns, pattern)
			}
		}
	}

	// Create exporter
	exporter, err := NewJSONLExporter(
		metricsEndpoint,
		c.String("metricsfile"),
		c.Duration("metricsinterval"),
		filterPatterns,
		c.Bool("metricscompress"),
		c.String("cf-access-client-id"),
		c.String("cf-access-client-secret"),
		log,
	)
	if err != nil {
		return fmt.Errorf("failed to create exporter: %w", err)
	}

	// Setup context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Setup signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start exporter in goroutine
	errChan := make(chan error, 1)
	go func() {
		errChan <- exporter.Run(ctx)
	}()

	// Log configuration
	logEvent := log.Info().
		Str("endpoint", metricsEndpoint).
		Str("file", c.String("metricsfile")).
		Dur("interval", c.Duration("metricsinterval")).
		Bool("compress", c.Bool("metricscompress"))
	if len(filterPatterns) > 0 {
		logEvent.Strs("filters", filterPatterns)
	}
	logEvent.Msg("JSONL metrics export started")

	// Wait for signal or error
	select {
	case sig := <-sigChan:
		log.Info().Str("signal", sig.String()).Msg("Received shutdown signal")
		cancel()
		// Wait for exporter to finish
		<-errChan
	case err := <-errChan:
		if err != nil {
			return fmt.Errorf("exporter error: %w", err)
		}
	}

	log.Info().Msg("Shutdown complete")
	return nil
}

func setupLogger(level string) *zerolog.Logger {
	zerolog.TimeFieldFormat = time.RFC3339

	var logLevel zerolog.Level
	switch strings.ToLower(level) {
	case "debug":
		logLevel = zerolog.DebugLevel
	case "info":
		logLevel = zerolog.InfoLevel
	case "warn":
		logLevel = zerolog.WarnLevel
	case "error":
		logLevel = zerolog.ErrorLevel
	default:
		logLevel = zerolog.InfoLevel
	}

	logger := zerolog.New(os.Stdout).
		Level(logLevel).
		With().
		Timestamp().
		Logger()

	return &logger
}
