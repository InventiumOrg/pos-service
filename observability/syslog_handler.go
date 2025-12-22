package observability

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"log/syslog"
	"os"
)

// SyslogHandler implements slog.Handler to send logs to syslog
type SyslogHandler struct {
	writer   *syslog.Writer
	level    slog.Level
	fallback slog.Handler
}

// NewSyslogHandler creates a new syslog handler
func NewSyslogHandler(network, address, tag string, level slog.Level) (*SyslogHandler, error) {
	writer, err := syslog.Dial(network, address, syslog.LOG_INFO|syslog.LOG_USER, tag)
	if err != nil {
		return nil, err
	}

	return &SyslogHandler{
		writer:   writer,
		level:    level,
		fallback: slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level}),
	}, nil
}

// Enabled reports whether the handler handles records at the given level
func (h *SyslogHandler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= h.level
}

// Handle processes a log record
func (h *SyslogHandler) Handle(ctx context.Context, record slog.Record) error {
	if err := h.fallback.Handle(ctx, record); err != nil {
		return err
	}

	logEntry := map[string]interface{}{
		"level": record.Level.String(),
		"msg":   record.Message,
	}

	record.Attrs(func(attr slog.Attr) bool {
		logEntry[attr.Key] = attr.Value.Any()
		return true
	})

	logJSON, err := json.Marshal(logEntry)
	if err != nil {
		return err
	}

	switch record.Level {
	case slog.LevelDebug:
		return h.writer.Debug(string(logJSON))
	case slog.LevelInfo:
		return h.writer.Info(string(logJSON))
	case slog.LevelWarn:
		return h.writer.Warning(string(logJSON))
	case slog.LevelError:
		return h.writer.Err(string(logJSON))
	default:
		return h.writer.Info(string(logJSON))
	}
}

// WithAttrs returns a new handler with additional attributes
func (h *SyslogHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &SyslogHandler{
		writer:   h.writer,
		level:    h.level,
		fallback: h.fallback.WithAttrs(attrs),
	}
}

// WithGroup returns a new handler with a group
func (h *SyslogHandler) WithGroup(name string) slog.Handler {
	return &SyslogHandler{
		writer:   h.writer,
		level:    h.level,
		fallback: h.fallback.WithGroup(name),
	}
}

// SetupSyslogLogging configures slog to send logs to syslog
func SetupSyslogLogging(network, address, serviceName string) error {
	handler, err := NewSyslogHandler(network, address, serviceName, slog.LevelInfo)
	if err != nil {
		return fmt.Errorf("failed to create syslog handler: %w", err)
	}

	logger := slog.New(handler)
	slog.SetDefault(logger)

	slog.Info("Syslog logging configured",
		slog.String("network", network),
		slog.String("address", address),
		slog.String("service", serviceName))

	return nil
}
