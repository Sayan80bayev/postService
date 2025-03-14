package delivery

import (
	"net/http"
	"postService/internal/config"
	"postService/internal/service"
	"strconv"

	"github.com/gin-gonic/gin"
)

type PostHandler struct {
	postService *service.PostService
	cfg         *config.Config
}

func NewPostHandler(postUsecase *service.PostService, cfg *config.Config) *PostHandler {
	return &PostHandler{postUsecase, cfg}
}

// Создание поста
func (h *PostHandler) CreatePost(c *gin.Context) {
	var req struct {
		Title      string `form:"title" binding:"required"`
		Content    string `form:"content" binding:"required"`
		CategoryID uint   `form:"category_id" binding:"required"`
	}

	if err := c.ShouldBind(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}

	userID, _ := c.Get("user_id")

	file, header, _ := c.Request.FormFile("file")

	if err := h.postService.CreatePost(req.Title, req.Content, userID.(uint), req.CategoryID, file, header, h.cfg); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not create post"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Post created successfully"})
}

// Получение списка постов
func (h *PostHandler) GetPosts(c *gin.Context) {
	posts, err := h.postService.GetPosts()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not fetch posts"})
		return
	}

	c.JSON(http.StatusOK, posts)
}

// Получение одного поста
func (h *PostHandler) GetPostByID(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	post, err := h.postService.GetPostByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Post not found"})
		return
	}

	c.JSON(http.StatusOK, post)
}

// ✅ Обновление поста
func (h *PostHandler) UpdatePost(c *gin.Context) {
	postID, _ := strconv.Atoi(c.Param("id"))

	var req struct {
		Title      string `form:"title"`
		Content    string `form:"content"`
		CategoryID uint   `form:"category_id"`
	}

	if err := c.ShouldBind(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}

	userID, _ := c.Get("user_id")
	file, header, _ := c.Request.FormFile("file")

	if err := h.postService.UpdatePost(req.Content, req.Title, userID.(uint), uint(postID), req.CategoryID, file, header, h.cfg); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Post updated successfully"})
}

func (h *PostHandler) DeletePost(c *gin.Context) {
	postID64, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid post ID"})
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	postID := uint(postID64)

	if err := h.postService.DeletePost(postID, userID.(uint)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not delete post" + ": " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Post deleted successfully"})
}
