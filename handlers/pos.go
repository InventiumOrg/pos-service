package handlers

import (
	"log/slog"
	"net/http"
	models "pos-service/models/sqlc"
	"pos-service/observability"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

const posIDAttr = "pos.id"
const statusAttr = "operation.status"

type Handlers struct {
	db                *pgx.Conn
	queries           *models.Queries
	tracer            trace.Tracer
	prometheusMetrics *observability.PrometheusMetrics
}

func NewHandlers(db *pgx.Conn, prometheusMetrics *observability.PrometheusMetrics) *Handlers {
	return &Handlers{
		db:                db,
		queries:           models.New(db),
		tracer:            otel.Tracer("pos-service/handlers"),
		prometheusMetrics: prometheusMetrics,
	}
}

func (h *Handlers) GetPOS(ctx *gin.Context) {
	spanCtx, span := h.tracer.Start(ctx.Request.Context(), "GetPOS")
	defer span.End()

	idStr := ctx.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		span.RecordError(err)
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid POS ID format",
		})
		return
	}

	span.SetAttributes(attribute.Int64(posIDAttr, id))

	dbStart := time.Now()
	POS, err := h.queries.GetPOS(spanCtx, id)
	dbDuration := time.Since(dbStart)

	if h.prometheusMetrics != nil {
		h.prometheusMetrics.RecordDBOperation("get", "pos", dbDuration, err)
	}

	if err != nil {
		span.RecordError(err)
		slog.Error("Got an error while getting POS", slog.Any("err", err.Error()))
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get POS",
		})
		return
	}

	// Record successful retrieval (Prometheus)
	if h.prometheusMetrics != nil {
		h.prometheusMetrics.RecordPOSOperation("get", "pos", POS.Location)
	}

	span.SetAttributes(
		attribute.String(statusAttr, "success"),
		attribute.String("pos.name", POS.Name),
	)
	ctx.JSON(200, gin.H{
		"message": "Get POS Successfully",
		"data":    POS,
	})
}

func (h *Handlers) ListPOS(ctx *gin.Context) {
	spanCtx, span := h.tracer.Start(ctx.Request.Context(), "ListPOS")
	defer span.End()

	span.SetAttributes(
		attribute.Int("pos.limit", 10),
		attribute.Int("pos.offset", 0),
	)

	dbStart := time.Now()
	POSs, err := h.queries.ListPOS(spanCtx, models.ListPOSParams{
		Limit:  10,
		Offset: 0,
	})
	dbDuration := time.Since(dbStart).Seconds()

	if h.prometheusMetrics != nil {
		h.prometheusMetrics.RecordDBOperation("list", "pos", time.Duration(dbDuration), err)
	}

	if err != nil {
		span.RecordError(err)
		slog.Error("Got an error while listing POS", slog.Any("err", err.Error()))
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to list POS",
		})
		return
	}

	if h.prometheusMetrics != nil {
		h.prometheusMetrics.RecordPOSOperation("list", "pos", "all")
	}

	span.SetAttributes(
		attribute.Int("pos.count", len(POSs)),
		attribute.String(statusAttr, "success"),
	)

	ctx.JSON(200, gin.H{
		"message": "List POS Successfully",
		"data":    POSs,
	})
}

func (h *Handlers) UpdatePOS(ctx *gin.Context) {
	spanCtx, span := h.tracer.Start(ctx.Request.Context(), "UpdatePOS")
	defer span.End()

	idStr := ctx.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		span.RecordError(err)
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid POS ID",
		})
		return
	}

	span.SetAttributes(attribute.Int64("pos.id", id))

	tx, err := h.db.Begin(spanCtx)
	if err != nil {
		span.RecordError(err)
		slog.Error("Failed to start transaction", slog.Any("err", err.Error()))
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to start transaction",
		})
		return
	}
	defer tx.Rollback(spanCtx)

	// Create query with transaction
	qtx := h.queries.WithTx(tx)

	// Check if pos exists before updating
	dbStart := time.Now()
	existingPOS, err := qtx.GetPOS(spanCtx, int64(id))
	dbDuration := time.Since(dbStart)

	// Record database operation duration for existence check (Prometheus)
	if h.prometheusMetrics != nil {
		h.prometheusMetrics.RecordDBOperation("get_for_update", "inventory", dbDuration, err)
	}

	if err != nil {
		span.RecordError(err)
		slog.Error("POS not found", slog.Any("err", err.Error()))
		ctx.JSON(http.StatusNotFound, gin.H{
			"error": "POS not found",
		})
		return
	}
	totalSaleUnit := int64(12434232)

	// Update POS within transaction
	param := models.UpdatePOSParams{
		ID:            int64(id),
		Name:          ctx.PostForm("Name"),
		Location:      ctx.PostForm("Location"),
		Description:   ctx.PostForm("Description"),
		TotalSaleUnit: totalSaleUnit,
	}

	dbStart = time.Now()
	POS, err := qtx.UpdatePOS(spanCtx, param)
	dbDuration = time.Since(dbStart)

	// Record database operation duration for update (Prometheus)
	if h.prometheusMetrics != nil {
		h.prometheusMetrics.RecordDBOperation("update", "pos", dbDuration, err)
	}

	if err != nil {
		slog.Error("Could not update pos", slog.Any("err", err.Error()))
		span.RecordError(err)
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to update pos",
		})
		return
	}
	// Commit transaction
	if err := tx.Commit(ctx); err != nil {
		slog.Error("Failed to commit transaction", slog.Any("err", err.Error()))
		span.RecordError(err)
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to commit transaction",
		})
		return
	}

	// Record successful update (Prometheus)
	if h.prometheusMetrics != nil {
		h.prometheusMetrics.RecordPOSOperation("update", POS.Name, POS.Location)

		// Track location changes if different
		if existingPOS.Location != POS.Location {
			h.prometheusMetrics.RecordPOSOperation("location_change", POS.Name, POS.Location)
		}

		// Track name changes if different
		if existingPOS.Name != POS.Name {
			h.prometheusMetrics.RecordPOSOperation("name_change", POS.Name, POS.Location)
		}
	}

	span.SetAttributes(attribute.String(statusAttr, "success"))

	ctx.JSON(200, gin.H{
		"message": "Update POS Successfully",
		"data":    POS,
	})
}

func (h *Handlers) CreatePOS(ctx *gin.Context) {
	spanCtx, span := h.tracer.Start(ctx.Request.Context(), "CreatePOS")
	defer span.End()

	totalSaleUnitStr := ctx.PostForm("Total Sale Unit")
	totalSaleUnit, err := strconv.ParseInt(totalSaleUnitStr, 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "Error parsing Total Sale Unit",
		})
	}
	param := models.CreatePOSParams{
		Name:          ctx.PostForm("Name"),
		Location:      ctx.PostForm("Location"),
		Description:   ctx.PostForm("Description"),
		TotalSaleUnit: totalSaleUnit,
	}

	// Add attributes to the span
	span.SetAttributes(
		attribute.String("pos.name", param.Name),
		attribute.String("pos.location", param.Location),
		attribute.String("pos.description", param.Description),
		attribute.Int("pos.total_sale_unit", int(param.TotalSaleUnit)),
	)

	dbStart := time.Now()
	POS, err := h.queries.CreatePOS(spanCtx, param)
	dbDuration := time.Since(dbStart)

	// Record database operation duration (Prometheus)
	if h.prometheusMetrics != nil {
		h.prometheusMetrics.RecordDBOperation("create", "pos", dbDuration, err)
	}

	if err != nil {
		slog.Error("Could not create pos: ", slog.Any("err", err.Error()))
		span.RecordError(err)
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to create pos",
		})
		return
	}

	// Record successful creation (Prometheus)
	if h.prometheusMetrics != nil {
		h.prometheusMetrics.RecordPOSOperation("create", POS.Name, POS.Location)
		// Update active inventory count (you'd need to query the total count or maintain it)
		// For now, we'll increment by 1 (in a real app, you'd want to track the actual count)
		h.prometheusMetrics.UpdatePOSCount(1) // This should be the actual total count
	}

	span.SetAttributes(attribute.String(statusAttr, "success"))
	ctx.JSON(200, gin.H{
		"message": "Create POS Successfully",
		"data":    POS,
	})
}

func (h *Handlers) DeletePOS(ctx *gin.Context) {
	spanCtx, span := h.tracer.Start(ctx.Request.Context(), "DeletePOS")
	defer span.End()

	idStr := ctx.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		span.RecordError(err)
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid POS ID format",
		})
		return
	}

	span.SetAttributes(attribute.Int64(posIDAttr, id))

	dbStart := time.Now()
	err = h.queries.DeletePOS(spanCtx, id)
	dbDuration := time.Since(dbStart)

	// Record database operation duration (Prometheus)
	if h.prometheusMetrics != nil {
		h.prometheusMetrics.RecordDBOperation("delete", "inventory", dbDuration, err)
	}

	if err != nil {
		span.RecordError(err)
		slog.Error("Failed to delete POS", slog.Any("err", err.Error()))
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to delete POS",
		})
		return
	}

	// Record successful deletion (Prometheus)
	if h.prometheusMetrics != nil {
		h.prometheusMetrics.RecordPOSOperation("delete", idStr, "unknown")
	}

	span.SetAttributes(attribute.String("operation.status", "success"))
	ctx.JSON(200, gin.H{"message": "Delete POS Successfully"})
}
