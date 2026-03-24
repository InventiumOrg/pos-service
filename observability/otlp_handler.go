package observability

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"
)

// OTLPHandler implements slog.Handler to send logs via OTLP HTTP
type OTLPHandler struct {
	client      *http.Client
	otlpURL     string
	serviceName string
	level       slog.Level
	fallback    slog.Handler
	headers     map[string]string
}

// OTLPConfig holds configuration for OTLP handler
type OTLPConfig struct {
	Endpoint    string
	ServiceName string
	Level       slog.Level
	Headers     map[string]string
}

// NewOTLPHandler creates a new OTLP handler
func NewOTLPHandler(config OTLPConfig) *OTLPHandler {
	return &OTLPHandler{
		client:      &http.Client{Timeout: 5 * time.Second},
		otlpURL:     config.Endpoint + "/v1/logs",
		serviceName: config.ServiceName,
		level:       config.Level,
		fallback:    slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: config.Level}),
		headers:     config.Headers,
	}
}

// Enabled reports whether the handler handles records at the given level
func (h *OTLPHandler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= h.level
}

// Handle processes a log record
func (h *OTLPHandler) Handle(ctx context.Context, record slog.Record) error {
	if err := h.fallback.Handle(ctx, record); err != nil {
		return err
	}

	go h.sendToOTLP(record)
	return nil
}

// WithAttrs returns a new handler with additional attributes
func (h *OTLPHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &OTLPHandler{
		client:      h.client,
		otlpURL:     h.otlpURL,
		serviceName: h.serviceName,
		level:       h.level,
		fallback:    h.fallback.WithAttrs(attrs),
		headers:     h.headers,
	}
}

// WithGroup returns a new handler with a group
func (h *OTLPHandler) WithGroup(name string) slog.Handler {
	return &OTLPHandler{
		client:      h.client,
		otlpURL:     h.otlpURL,
		serviceName: h.serviceName,
		level:       h.level,
		fallback:    h.fallback.WithGroup(name),
		headers:     h.headers,
	}
}

// OTLP Log structures
type OTLPLogsPayload struct {
	ResourceLogs []ResourceLogs `json:"resourceLogs"`
}

type ResourceLogs struct {
	Resource  Resource    `json:"resource"`
	ScopeLogs []ScopeLogs `json:"scopeLogs"`
}

type Resource struct {
	Attributes []Attribute `json:"attributes"`
}

type ScopeLogs struct {
	Scope      Scope       `json:"scope"`
	LogRecords []LogRecord `json:"logRecords"`
}

type Scope struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type LogRecord struct {
	TimeUnixNano   string      `json:"timeUnixNano"`
	SeverityNumber int         `json:"severityNumber"`
	SeverityText   string      `json:"severityText"`
	Body           Body        `json:"body"`
	Attributes     []Attribute `json:"attributes"`
}

type Body struct {
	StringValue string `json:"stringValue"`
}

type Attribute struct {
	Key   string      `json:"key"`
	Value interface{} `json:"value"`
}

// sendToOTLP sends the log record via OTLP
func (h *OTLPHandler) sendToOTLP(record slog.Record) {
	severityNumber := h.slogLevelToOTLP(record.Level)

	var attributes []Attribute
	record.Attrs(func(attr slog.Attr) bool {
		attributes = append(attributes, Attribute{
			Key:   attr.Key,
			Value: map[string]interface{}{"stringValue": fmt.Sprintf("%v", attr.Value.Any())},
		})
		return true
	})

	payload := OTLPLogsPayload{
		ResourceLogs: []ResourceLogs{
			{
				Resource: Resource{
					Attributes: []Attribute{
						{
							Key:   "service.name",
							Value: map[string]interface{}{"stringValue": h.serviceName},
						},
					},
				},
				ScopeLogs: []ScopeLogs{
					{
						Scope: Scope{
							Name:    "go-slog",
							Version: "1.0.0",
						},
						LogRecords: []LogRecord{
							{
								TimeUnixNano:   fmt.Sprintf("%d", record.Time.UnixNano()),
								SeverityNumber: severityNumber,
								SeverityText:   record.Level.String(),
								Body: Body{
									StringValue: record.Message,
								},
								Attributes: attributes,
							},
						},
					},
				},
			},
		},
	}

	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return
	}

	req, err := http.NewRequest("POST", h.otlpURL, bytes.NewBuffer(payloadJSON))
	if err != nil {
		return
	}

	req.Header.Set("Content-Type", "application/json")

	// Add configured headers (like Authorization)
	for key, value := range h.headers {
		req.Header.Set(key, value)
	}

	resp, err := h.client.Do(req)
	if err != nil {
		slog.Error("Failed to send OTLP log", slog.String("error", err.Error()), slog.String("url", h.otlpURL))
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		slog.Error("OTLP server returned error",
			slog.Int("status_code", resp.StatusCode),
			slog.String("status", resp.Status),
			slog.String("url", h.otlpURL))
	}
}

// slogLevelToOTLP converts slog level to OTLP severity number
func (h *OTLPHandler) slogLevelToOTLP(level slog.Level) int {
	switch level {
	case slog.LevelDebug:
		return 5
	case slog.LevelInfo:
		return 9
	case slog.LevelWarn:
		return 13
	case slog.LevelError:
		return 17
	default:
		return 9
	}
}

// SetupOTLPLogging configures slog to send logs via OTLP
func SetupOTLPLogging(endpoint string, serviceName string) error {
	return SetupOTLPLoggingWithHeaders(endpoint, serviceName, "")
}

// SetupOTLPLoggingWithHeaders configures slog to send logs via OTLP with custom headers
func SetupOTLPLoggingWithHeaders(endpoint, serviceName, headersStr string) error {
	// Parse headers from environment variable format
	headers := make(map[string]string)
	if headersStr != "" {
		pairs := strings.Split(headersStr, ",")
		for _, pair := range pairs {
			if kv := strings.SplitN(pair, "=", 2); len(kv) == 2 {
				headers[strings.TrimSpace(kv[0])] = strings.TrimSpace(kv[1])
			}
		}
	}

	config := OTLPConfig{
		Endpoint:    "http://" + endpoint,
		ServiceName: serviceName,
		Level:       slog.LevelInfo,
		Headers:     headers,
	}

	handler := NewOTLPHandler(config)
	logger := slog.New(handler)
	slog.SetDefault(logger)

	slog.Info("OTLP logging configured",
		slog.String("endpoint", config.Endpoint),
		slog.String("service", serviceName),
		slog.Int("headers_count", len(headers)))

	return nil
}
