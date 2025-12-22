# Observability Implementation

This document describes the OpenTelemetry observability implementation for the POS service, following patterns from the warehouse-service.

## Overview

The service implements comprehensive observability with:
- **Traces**: Distributed tracing with OpenTelemetry
- **Metrics**: Business and application metrics
- **Logs**: Structured logging with multiple backends

## Configuration

Configure observability through environment variables in `app.env`:

```env
SERVICE_NAME="pos-service"
OTEL_EXPORTER_OTLP_ENDPOINT="localhost:4318"
OTEL_EXPORTER_OTLP_HEADERS=""
LOKI_URL=""
SYSLOG_ADDRESS=""
SYSLOG_NETWORK="udp"
LOG_FILE_PATH="/logs/otel-logs.json"
```

## Logging

### Priority Order
The service tries logging backends in this order:
1. **OTLP** - Direct to OpenTelemetry Collector
2. **Loki** - Direct to Grafana Loki
3. **Syslog** - Traditional syslog
4. **File** - JSON file logging
5. **Stdout** - Console JSON logging (fallback)

### Log Handlers

#### OTLP Handler (`observability/otlp_handler.go`)
Sends logs directly to OTLP endpoint in OpenTelemetry format.

#### Loki Handler (`observability/loki_handler.go`)
Sends logs to Grafana Loki via HTTP API with labels.

#### Syslog Handler (`observability/syslog_handler.go`)
Sends logs to syslog server (UDP/TCP).

## Metrics

### Application Metrics
- `http_requests_total` - Total HTTP requests
- `http_request_duration_seconds` - Request duration histogram
- `db_connections_active` - Active database connections

### Business Metrics
- `pos_operations_total` - Total POS operations
- `pos_created_total` - POS creation count
- `pos_retrievals_total` - POS retrieval count
- `pos_list_requests_total` - List requests count
- `pos_deletes_total` - POS deletion count
- `pos_updates_total` - POS update count
- `database_operation_duration_seconds` - DB operation duration
- `database_operation_errors_total` - DB error count
- `authentication_attempts_total` - Auth attempts
- `active_pos_count` - Current active POS count

## Tracing

All handler operations create spans with:
- Operation name
- Attributes (IDs, counts, status)
- Error recording
- Duration tracking

### Example Span Attributes
```go
span.SetAttributes(
    attribute.Int64("pos.id", id),
    attribute.String("operation.status", "success"),
)
```

## Usage in Handlers

Handlers automatically record:
1. **Traces** - Span per operation
2. **Metrics** - Business metrics per operation
3. **Logs** - Structured logs with context

### Example Pattern
```go
func (h *Handlers) GetPOS(ctx *gin.Context) {
    spanCtx, span := h.tracer.Start(ctx.Request.Context(), "GetPOS")
    defer span.End()
    
    start := time.Now()
    result, err := h.queries.GetPOS(spanCtx, id)
    duration := time.Since(start).Seconds()
    
    if h.businessMetrics != nil {
        h.businessMetrics.DBOperationDuration.Record(spanCtx, duration,
            metric.WithAttributes(attribute.String("operation", "get_pos")))
    }
    
    if err != nil {
        span.RecordError(err)
        h.businessMetrics.DBOperationErrors.Add(spanCtx, 1)
        return
    }
    
    h.businessMetrics.POSRetrievals.Add(spanCtx, 1)
}
```

## Integration with Grafana Stack

### Grafana Cloud Setup
1. Set `OTEL_EXPORTER_OTLP_ENDPOINT` to your Grafana Cloud OTLP endpoint
2. Set `OTEL_EXPORTER_OTLP_HEADERS` with authentication token
3. Optionally set `LOKI_URL` for direct Loki integration

### Local Development
1. Run OpenTelemetry Collector locally
2. Configure collector to export to Jaeger/Prometheus/Loki
3. Set `OTEL_EXPORTER_OTLP_ENDPOINT=localhost:4318`

## Error Handling

All observability operations:
- Never block the main request flow
- Fail gracefully with fallbacks
- Log errors without crashing
- Continue with reduced functionality if backends unavailable

## Best Practices

1. **Always use context** - Pass span context through call chain
2. **Record errors** - Use `span.RecordError(err)` for all errors
3. **Add attributes** - Include relevant IDs and status
4. **Measure duration** - Track operation timing
5. **Increment counters** - Update business metrics
6. **Structured logging** - Use slog with key-value pairs
