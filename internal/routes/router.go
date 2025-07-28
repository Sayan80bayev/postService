package routes

import (
	"github.com/gin-gonic/gin"
	"postService/internal/bootstrap"
)

func SetupRoutes(r *gin.Engine, c *bootstrap.Container) {
	SetupPostRoutes(r, c)
}
