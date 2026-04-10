package utils

import (
	"log/slog"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// PathPOSID parses the :id route parameter. On failure it logs, writes 400, and returns ok=false.
func PathPOSID(ctx *gin.Context, rejectPrefix string) (id int64, ok bool) {
	idStr := ctx.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		slog.Info(rejectPrefix+": invalid pos id", slog.String("id_param", idStr), slog.Any("err", err))
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid POS ID",
		})
		return 0, false
	}
	return id, true
}
