package main

import (
	"context"
	"log/slog"
	"os"
	"pos-service/api"
	"pos-service/config"
	"pos-service/observability"
	"time"

	"github.com/clerk/clerk-sdk-go/v2"
	"github.com/jackc/pgx/v5"
)

var conn *pgx.Conn

const attemptThreshold = 5

// setupLogging configures logging based on environment variables
func setupLogging(cfg config.Config) error {
	// Priority order: OTLP > Loki > Syslog > File > Stdout

	// Option 1: Direct OTLP Logs
	if cfg.OTELExporterOTLPEndpoint != "" {
		if err := observability.SetupOTLPLoggingWithHeaders(cfg.OTELExporterOTLPEndpoint, cfg.ServiceName, cfg.OTELExporterOTLPHeaders); err == nil {
			slog.Info("Using OTLP logging", slog.String("endpoint", cfg.OTELExporterOTLPEndpoint))
			return nil
		}
		slog.Warn("OTLP logging failed, trying next option")
	}

	// Option 2: Direct Loki HTTP
	if cfg.LokiURL != "" {
		if err := observability.SetupDirectLokiLogging(cfg.LokiURL, cfg.ServiceName); err == nil {
			slog.Info("Using direct Loki logging", slog.String("url", cfg.LokiURL))
			return nil
		}
		slog.Warn("Direct Loki logging failed, trying next option")
	}

	// Option 3: Syslog
	if cfg.SyslogAddress != "" {
		network := cfg.SyslogNetwork
		if network == "" {
			network = "udp"
		}
		if err := observability.SetupSyslogLogging(network, cfg.SyslogAddress, cfg.ServiceName); err == nil {
			slog.Info("Using syslog logging", slog.String("address", cfg.SyslogAddress))
			return nil
		}
		slog.Warn("Syslog logging failed, trying next option")
	}

	// Option 4: File logging
	if cfg.LogFilePath != "" {
		logConfig := observability.LogConfig{
			FilePath:   cfg.LogFilePath,
			MaxSizeMB:  100,
			MaxBackups: 5,
			MaxAgeDays: 30,
			Compress:   true,
		}
		if err := observability.SetupAdvancedFileLogger(logConfig); err == nil {
			slog.Info("Using file logging", slog.String("path", cfg.LogFilePath))
			return nil
		}
		slog.Warn("File logging failed, using stdout")
	}

	// Option 5: Default stdout JSON logging
	slog.Info("Using default stdout logging")
	return nil
}

func main() {
	config, err := config.LoadConfig(".")
	if err != nil {
		slog.Error("Failed to load config: ", slog.Any("ERROR", err))
		os.Exit(1)
	}

	slog.Info("Set Up Logging.....")
	if err := setupLogging(config); err != nil {
		slog.Error("Failed to setup logging", slog.Any("error", err))
	}

	clerk.SetKey(config.ClerKKey)
	slog.Info("Connecting to database", slog.String("db_source", config.DBSource))
	attempt := 1
	for attempt <= attemptThreshold {
		conn, err = pgx.Connect(context.Background(), config.DBSource)
		if err == nil {
			slog.Info("Connected to database successfully")
			break
		}
		slog.Error("Failed to connect to database",
			slog.Int("attempt", attempt),
			slog.Int("maxAttempts", attemptThreshold),
			slog.Any("error", err),
		)

		if attempt == attemptThreshold {
			slog.Error("Max connection attempts reached, exiting", slog.Any("ERROR", err))
			os.Exit(1)
		}

		backoffDuration := time.Duration(1<<(attempt-1)) * time.Second
		slog.Info("Retrying connection",
			slog.Int("attempt", attempt+1),
			slog.Duration("backoff", backoffDuration),
		)

		time.Sleep(backoffDuration)
		attempt++
	}

	router := api.NewServer(conn, config.ServiceName, "1.0.0", config.OTELExporterOTLPEndpoint, config.OTELExporterOTLPHeaders)

	err = router.Run(":11890", config.ServiceName)
	if err != nil {
		os.Exit(1)
	}
}
