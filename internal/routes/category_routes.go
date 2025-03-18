package routes

import (
	"github.com/gin-gonic/gin"
	"postService/internal/bootstrap"
	"postService/internal/delivery"
	"postService/internal/pkg/middleware"
	"postService/internal/repository"
	"postService/internal/service"
)

func SetupCategoryRoutes(router *gin.Engine, bs *bootstrap.Bootstrap, authMiddleware gin.HandlerFunc) {
	repoInterface, err := bs.GetRepository("category")
	if err != nil {
		logger.Error("Failed to get category repository: " + err.Error())
	}
	categoryRepo, ok := repoInterface.(*repository.CategoryRepositoryImpl)
	if !ok {
		logger.Error("Invalid repository type for category")
	}

	categoryService := service.NewCategoryService(categoryRepo)
	categoryHandler := delivery.NewCategoryHandler(categoryService)

	router.GET("/category", categoryHandler.ListCategory)

	categoryGroup := router.Group("/category", authMiddleware, middleware.CheckAdminRole())
	{
		categoryGroup.POST("/", categoryHandler.CreateCategory)
		categoryGroup.DELETE("/:id", categoryHandler.DeleteCategory)
	}
}
