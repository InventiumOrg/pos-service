package routes

import (
	handlers "pos-service/handlers"
	"pos-service/observability"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
)

type Route struct {
	db       *pgx.Conn
	handlers *handlers.Handlers
}

func NewRoute(db *pgx.Conn, prometheusMetrics *observability.PrometheusMetrics) *Route {
	return &Route{
		db:       db,
		handlers: handlers.NewHandlers(db, prometheusMetrics),
	}
}

func (r *Route) AddPOSRoutes(router *gin.Engine) {
	v1 := router.Group("/v1")
	{
		pos := v1.Group("/pos")
		{
			pos.GET("/:id", r.handlers.GetPOS)
			pos.GET("/list", r.handlers.ListPOS)
			pos.POST("/create", r.handlers.CreatePOS)
			pos.PUT("/:id", r.handlers.UpdatePOS)
			pos.DELETE("/:id", r.handlers.DeletePOS)
		}
	}
}

func (r *Route) AddHealthRoutes(router *gin.Engine) {
	// Health check endpoints (no authentication required)
	router.GET("/healthz", r.handlers.HealthzHandler)
	router.GET("/readyz", r.handlers.ReadyzHandler)
}
