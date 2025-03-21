package routes

import (
	"github.com/gin-gonic/gin"
	"postService/pkg/logging"

	"postService/internal/bootstrap"
)

var logger = logging.GetLogger()

func SetupRoutes(r *gin.Engine, c *bootstrap.Container) {
	SetupPostRoutes(r, c)
	SetupCategoryRoutes(r, c)
}
