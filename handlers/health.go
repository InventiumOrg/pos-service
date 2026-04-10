package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel/attribute"
)

// HealthzHandler handles the /healthz endpoint for basic liveness check
func (h *Handlers) HealthzHandler(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, gin.H{
		"status":    "ok",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"service":   "pos-service",
	})
}

// ReadyzHandler handles the /readyz endpoint for readiness check
func (h *Handlers) ReadyzHandler(ctx *gin.Context) {
	_, span := h.tracer.Start(ctx.Request.Context(), "Health Check")
	defer span.End()

	dbStart := time.Now()
	err := h.db.Ping(context.Background())
	dbDuration := time.Since(dbStart)

	if h.prometheusMetrics != nil {
		h.prometheusMetrics.RecordDBOperation("ping", "connection", dbDuration, err)
	}

	if err != nil {
		span.RecordError(err)
		span.SetAttributes(attribute.String("error", "database_ping_failed"))

		ctx.JSON(http.StatusServiceUnavailable, gin.H{
			"status":    "not ready",
			"timestamp": time.Now().UTC().Format(time.RFC3339),
			"service":   "pos-service",
			"error":     "database connection failed",
			"details":   err.Error(),
		})
		return
	}

	span.SetAttributes(attribute.String("health.status", "ready"))

	ctx.JSON(http.StatusOK, gin.H{
		"status":    "ready",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"service":   "pos-service",
		"checks": gin.H{
			"database": "ok",
		},
	})
}
