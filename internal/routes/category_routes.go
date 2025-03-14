package routes

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"postService/internal/delivery"
	"postService/internal/repository"
	"postService/internal/service"
	"postService/pkg/middleware"
)

func SetupCategoryRoutes(router *gin.Engine, db *gorm.DB, authMiddleware gin.HandlerFunc) {
	categoryRepo := repository.NewCategoryRepository(db)
	categoryService := service.NewCategoryService(categoryRepo)
	categoryHandler := delivery.NewCategoryHandler(categoryService)

	router.GET("/category", categoryHandler.ListCategory)

	categoryGroup := router.Group("/category", authMiddleware, middleware.CheckAdminRole())
	{
		categoryGroup.POST("/", categoryHandler.CreateCategory)
		categoryGroup.DELETE("/:id", categoryHandler.DeleteCategory)
	}

}
