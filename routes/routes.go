package routes

import (
	handlers "pos-service/handlers"
	"pos-service/middlewares"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
)

type Route struct {
	db       *pgx.Conn
	handlers *handlers.Handlers
}

func NewRoute(db *pgx.Conn) *Route {
	return &Route{
		db:       db,
		handlers: handlers.NewHandlers(db),
	}
}

func (r *Route) AddPOSRoutes(router *gin.Engine) {
	v1 := router.Group("/v1")
	{
		inventory := v1.Group("/pos")
		inventory.Use(middlewares.ClerkAuth(r.db))
		{
			inventory.GET("/:id", r.handlers.GetPOS)
			inventory.GET("/list", r.handlers.ListPOS)
			inventory.POST("/create", r.handlers.CreatePOS)
			inventory.PUT("/:id", r.handlers.UpdatePOS)
			inventory.DELETE("/:id", r.handlers.DeletePOS)
		}
	}
}

func (r *Route) AddHealthRoutes(router *gin.Engine) {
	// Health check endpoints (no authentication required)
	router.GET("/healthz", r.handlers.HealthzHandler)
	router.GET("/readyz", r.handlers.ReadyzHandler)
}
