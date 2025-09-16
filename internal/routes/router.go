package routes

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"postService/internal/bootstrap"
)

func SetupRoutes(r *gin.Engine, c *bootstrap.Container) {
	SetupPostRoutes(r, c)
	r.GET("/download", func(ctx *gin.Context) {
		fileURL := ctx.Query("url")
		if fileURL == "" {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "missing url parameter"})
			return
		}

		err := c.FileStorage.DownloadFile(ctx.Writer, ctx.Request, fileURL)
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	})
}
