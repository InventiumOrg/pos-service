package api

import (
	"context"
	"log/slog"
	"pos-service/observability"
	routes "pos-service/routes"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

type Server struct {
	router            *gin.Engine
	routes            *routes.Route
	db                *pgx.Conn
	otelShutdown      func(context.Context) error
	metrics           *observability.AppMetrics
	prometheusMetrics *observability.PrometheusMetrics
}

func NewServer(db *pgx.Conn, serviceName, serviceVersion, otelEndpoint, otelHeaders string) *Server {
	// Setup OpenTelemetry
	// ctx := context.Background()
	// otelShutdown, err := observability.SetupOTelSDK(ctx, serviceName, serviceVersion, otelEndpoint, otelHeaders)
	// if err != nil {
	// 	slog.Error("Failed to setup OpenTelemetry", slog.Any("error", err))
	// 	otelShutdown = func(context.Context) error { return nil }
	// }

	// Create metrics
	metrics, err := observability.CreateMetrics()
	if err != nil {
		slog.Error("Failed to create metrics", slog.Any("error", err))
	}

	// Create business metrics
	prometheusMetrics := observability.NewPrometheusMetrics(serviceName)

	router := gin.Default()

	server := &Server{
		router: router,
		db:     db,
		// otelShutdown:      otelShutdown,
		metrics:           metrics,
		prometheusMetrics: prometheusMetrics,
	}

	router.Use(prometheusMetrics.PrometheusMiddleware())
	observability.SetupPrometheusEndpoint(router)
	// Add metrics middleware
	// router.Use(server.metricsMiddleware())
	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:8000"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"},
		AllowHeaders:     []string{"Origins", "Content-Type", "Authorization", "Bearer"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// Setup routes
	server.routes = routes.NewRoute(db, prometheusMetrics)

	// Add health check routes (no auth required)
	server.routes.AddHealthRoutes(server.router)

	// Add business logic routes
	server.routes.AddPOSRoutes(server.router)

	return server
}

func (s *Server) Run(addr string, serviceName string) error {
	slog.Info("Starting POS service server",
		slog.String("address", addr),
		slog.String("service", serviceName))

	return s.router.Run(addr)
}

// Shutdown gracefully shuts down the server and OpenTelemetry
func (s *Server) Shutdown(ctx context.Context) error {
	slog.Info("Shutting down POS service server")

	if s.otelShutdown != nil {
		if err := s.otelShutdown(ctx); err != nil {
			slog.Error("Failed to shutdown OpenTelemetry", slog.Any("error", err))
			return err
		}
	}

	if s.db != nil {
		s.db.Close(ctx)
	}

	return nil
}

// metricsMiddleware records HTTP request metrics
func (s *Server) metricsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		c.Next()

		if s.metrics != nil {
			duration := time.Since(start).Seconds()

			s.metrics.RequestCounter.Add(c.Request.Context(), 1,
				metric.WithAttributes(
					attribute.String("method", c.Request.Method),
					attribute.String("route", c.FullPath()),
					attribute.Int("status_code", c.Writer.Status()),
				))

			s.metrics.RequestDuration.Record(c.Request.Context(), duration,
				metric.WithAttributes(
					attribute.String("method", c.Request.Method),
					attribute.String("route", c.FullPath()),
					attribute.Int("status_code", c.Writer.Status()),
				))
		}
	}
}
