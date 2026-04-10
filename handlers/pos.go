package handlers

import (
	"log/slog"
	"net/http"
	models "pos-service/models/sqlc"
	"pos-service/observability"
	"pos-service/utils"
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

	id, ok := utils.PathPOSID(ctx, "get pos rejected")
	if !ok {
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
		slog.Error("failed to get pos", slog.Int64("pos.id", id), slog.Any("err", err))
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get POS",
		})
		return
	}

	if h.prometheusMetrics != nil {
		h.prometheusMetrics.RecordPOSOperation("get", "pos", POS.Location)
	}

	span.SetAttributes(
		attribute.String(statusAttr, "success"),
		attribute.String("pos.name", POS.Name),
	)

	slog.Info("pos retrieved",
		slog.Int64("pos.id", id),
		slog.String("pos.name", POS.Name),
		slog.String("pos.location", POS.Location),
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
	dbDuration := time.Since(dbStart)

	if h.prometheusMetrics != nil {
		h.prometheusMetrics.RecordDBOperation("list", "pos", dbDuration, err)
	}

	if err != nil {
		span.RecordError(err)
		slog.Error("failed to list pos", slog.Any("err", err))
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

	slog.Info("pos listed", slog.Int("count", len(POSs)))
	ctx.JSON(200, gin.H{
		"message": "List POS Successfully",
		"data":    POSs,
	})
}

func (h *Handlers) UpdatePOS(ctx *gin.Context) {
	spanCtx, span := h.tracer.Start(ctx.Request.Context(), "UpdatePOS")
	defer span.End()

	id, ok := utils.PathPOSID(ctx, "update pos rejected")
	if !ok {
		return
	}

	span.SetAttributes(attribute.Int64("pos.id", id))

	tx, err := h.db.Begin(spanCtx)
	if err != nil {
		span.RecordError(err)
		slog.Error("failed to start transaction", slog.Any("err", err))
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to start transaction",
		})
		return
	}
	defer tx.Rollback(spanCtx)

	qtx := h.queries.WithTx(tx)

	dbStart := time.Now()
	existingPOS, err := qtx.GetPOS(spanCtx, id)
	dbDuration := time.Since(dbStart)

	if h.prometheusMetrics != nil {
		h.prometheusMetrics.RecordDBOperation("get_for_update", "pos", dbDuration, err)
	}

	if err != nil {
		span.RecordError(err)
		slog.Error("pos not found", slog.Int64("pos.id", id), slog.Any("err", err))
		ctx.JSON(http.StatusNotFound, gin.H{
			"error": "POS not found",
		})
		return
	}
	totalSaleUnit := int64(12434232)

	param := models.UpdatePOSParams{
		ID:            id,
		Name:          ctx.PostForm("Name"),
		Location:      ctx.PostForm("Location"),
		Description:   ctx.PostForm("Description"),
		TotalSaleUnit: totalSaleUnit,
	}

	dbStart = time.Now()
	POS, err := qtx.UpdatePOS(spanCtx, param)
	dbDuration = time.Since(dbStart)

	if h.prometheusMetrics != nil {
		h.prometheusMetrics.RecordDBOperation("update", "pos", dbDuration, err)
	}

	if err != nil {
		slog.Error("failed to update pos", slog.Int64("pos.id", id), slog.Any("err", err))
		span.RecordError(err)
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to update pos",
		})
		return
	}
	if err := tx.Commit(spanCtx); err != nil {
		slog.Error("failed to commit transaction", slog.Any("err", err))
		span.RecordError(err)
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to commit transaction",
		})
		return
	}

	if h.prometheusMetrics != nil {
		h.prometheusMetrics.RecordPOSOperation("update", POS.Name, POS.Location)

		if existingPOS.Location != POS.Location {
			h.prometheusMetrics.RecordPOSOperation("location_change", POS.Name, POS.Location)
		}

		if existingPOS.Name != POS.Name {
			h.prometheusMetrics.RecordPOSOperation("name_change", POS.Name, POS.Location)
		}
	}

	span.SetAttributes(attribute.String(statusAttr, "success"))

	slog.Info("pos updated",
		slog.Int64("pos.id", id),
		slog.String("pos.name", POS.Name),
		slog.String("pos.location", POS.Location),
	)
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
		span.RecordError(err)
		slog.Info("create pos rejected: invalid total sale unit",
			slog.String("total_sale_unit", totalSaleUnitStr),
			slog.Any("err", err),
		)
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "Error parsing Total Sale Unit",
		})
		return
	}
	param := models.CreatePOSParams{
		Name:          ctx.PostForm("Name"),
		Location:      ctx.PostForm("Location"),
		Description:   ctx.PostForm("Description"),
		TotalSaleUnit: totalSaleUnit,
	}

	span.SetAttributes(
		attribute.String("pos.name", param.Name),
		attribute.String("pos.location", param.Location),
		attribute.String("pos.description", param.Description),
		attribute.Int64("pos.total_sale_unit", param.TotalSaleUnit),
	)

	dbStart := time.Now()
	POS, err := h.queries.CreatePOS(spanCtx, param)
	dbDuration := time.Since(dbStart)

	if h.prometheusMetrics != nil {
		h.prometheusMetrics.RecordDBOperation("create", "pos", dbDuration, err)
	}

	if err != nil {
		slog.Error("failed to create pos",
			slog.String("pos.name", param.Name),
			slog.Any("err", err),
		)
		span.RecordError(err)
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to create pos",
		})
		return
	}

	if h.prometheusMetrics != nil {
		h.prometheusMetrics.RecordPOSOperation("create", POS.Name, POS.Location)
		h.prometheusMetrics.UpdatePOSCount(1)
	}

	span.SetAttributes(attribute.String(statusAttr, "success"))

	slog.Info("pos created",
		slog.Int64("pos.id", POS.ID),
		slog.String("pos.name", POS.Name),
		slog.String("pos.location", POS.Location),
	)
	ctx.JSON(http.StatusCreated, gin.H{
		"message": "Create POS Successfully",
		"data":    POS,
	})
}

func (h *Handlers) DeletePOS(ctx *gin.Context) {
	spanCtx, span := h.tracer.Start(ctx.Request.Context(), "DeletePOS")
	defer span.End()

	id, ok := utils.PathPOSID(ctx, "delete pos rejected")
	if !ok {
		return
	}

	span.SetAttributes(attribute.Int64(posIDAttr, id))

	dbStart := time.Now()
	err := h.queries.DeletePOS(spanCtx, id)
	dbDuration := time.Since(dbStart)

	if h.prometheusMetrics != nil {
		h.prometheusMetrics.RecordDBOperation("delete", "pos", dbDuration, err)
	}

	if err != nil {
		span.RecordError(err)
		slog.Error("failed to delete pos", slog.Int64("pos.id", id), slog.Any("err", err))
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to delete POS",
		})
		return
	}

	if h.prometheusMetrics != nil {
		h.prometheusMetrics.RecordPOSOperation("delete", strconv.FormatInt(id, 10), "unknown")
		h.prometheusMetrics.UpdatePOSCount(-1)
	}

	span.SetAttributes(attribute.String("operation.status", "success"))

	slog.Info("pos deleted", slog.Int64("pos.id", id))
	ctx.JSON(200, gin.H{"message": "Delete POS Successfully"})
}
