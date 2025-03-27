package routes

import (
	"github.com/gin-gonic/gin"
	"postService/internal/bootstrap"
	"postService/internal/delivery"
	"postService/internal/pkg/middleware"
	"postService/internal/repository"
	"postService/internal/service"
)

func SetupCategoryRoutes(router *gin.Engine, c *bootstrap.Container) {
	repoInterface, err := c.GetRepository("category")
	if err != nil {
		logger.Error("Failed to get category repository: " + err.Error())
	}

	categoryRepo, ok := repoInterface.(*repository.CategoryRepositoryImpl)
	if !ok {
		logger.Error("Invalid repository type for category")
	}

	categoryService := service.NewCategoryService(categoryRepo)
	categoryHandler := delivery.NewCategoryHandler(categoryService)

	router.GET("api/v1/category", categoryHandler.ListCategory)

	categoryGroup := router.Group("/category", middleware.AuthMiddleware(c.Config.JWTSecret), middleware.AdminMiddleware())
	{
		categoryGroup.POST("/", categoryHandler.CreateCategory)
		categoryGroup.DELETE("/:id", categoryHandler.DeleteCategory)
	}
}
