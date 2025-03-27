package delivery

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"postService/internal/service"
	"strconv"
)

type CategoryHandler struct {
	CategoryService *service.CategoryService
}

func NewCategoryHandler(categoryService *service.CategoryService) *CategoryHandler {
	return &CategoryHandler{categoryService}
}

// CreateCategory
// @Summary Create a new category
// @Description Adds a new category to the system (Admin only).
// @Tags Categories
// @Accept json
// @Produce json
// @Security BearerAuth
// @Router /api/v1/category [post]
func (ch *CategoryHandler) CreateCategory(c *gin.Context) {
	var req struct {
		Name string `json:"name"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}
	_, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	if err := ch.CategoryService.CreateCategory(req.Name); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": req.Name})
}

// ListCategory
// @Summary List all categories
// @Description Retrieves all available categories.
// @Tags Categories
// @Accept json
// @Produce json
// @Router /api/v1/category [get]
func (ch *CategoryHandler) ListCategory(c *gin.Context) {
	categories, err := ch.CategoryService.GetCategories()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not fetch categories"})
	}
	c.JSON(http.StatusOK, categories)
}

// DeleteCategory
// @Summary Delete a category
// @Description Deletes a category by ID (Admin only).
// @Tags Categories
// @Accept json
// @Produce json
// @Param id path int true "Category ID"
// @Security BearerAuth
// @Router /api/v1/category/{id} [delete]
func (ch *CategoryHandler) DeleteCategory(c *gin.Context) {
	_, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
	}

	id, _ := strconv.Atoi(c.Param("id"))
	err := ch.CategoryService.DeleteCategory(uint(id))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not delete category"})
	}

	c.JSON(http.StatusOK, gin.H{"data": "Category deleted successfully"})
}
