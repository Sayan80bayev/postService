package routes

import (
	"github.com/gin-gonic/gin"
	"postService/pkg/logging"

	"postService/internal/bootstrap"
	"postService/pkg/middleware"
)

var logger = logging.GetLogger()

func SetupRoutes(r *gin.Engine, bs *bootstrap.Bootstrap) {
	authMiddleware := middleware.AuthMiddleware(bs.Config.JWTSecret)

	SetupPostRoutes(r, bs, authMiddleware)
	SetupCategoryRoutes(r, bs, authMiddleware)
}
