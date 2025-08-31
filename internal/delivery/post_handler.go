package delivery

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"net/http"
	"postService/internal/config"
	"postService/internal/service"
	"postService/internal/transfer/request"
)

type PostHandler struct {
	postService *service.PostService
	cfg         *config.Config
}

func NewPostHandler(postService *service.PostService, cfg *config.Config) *PostHandler {
	return &PostHandler{postService, cfg}
}

// CreatePost handles POST /posts
func (h *PostHandler) CreatePost(c *gin.Context) {
	ctx := c.Request.Context()

	var req request.PostRequest
	if err := c.ShouldBind(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input: " + err.Error()})
		return
	}

	userID, ok := c.Get("user_id")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	req.UserID = userID.(uuid.UUID)

	form, err := c.MultipartForm()
	if err == nil {
		req.Media = form.File["media"]
		req.Files = form.File["files"]
	}

	if err := h.postService.CreatePost(ctx, req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not create post: " + err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Post created successfully"})
}

// GetPosts handles GET /posts
func (h *PostHandler) GetPosts(c *gin.Context) {
	ctx := c.Request.Context()

	posts, err := h.postService.GetPosts(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not fetch posts"})
		return
	}

	c.JSON(http.StatusOK, posts)
}

// GetPostByID handles GET /posts/:id
func (h *PostHandler) GetPostByID(c *gin.Context) {
	ctx := c.Request.Context()

	id := c.Param("id")
	idUUID, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid post ID"})
		return
	}

	post, err := h.postService.GetPostByID(ctx, idUUID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Post not found"})
		return
	}

	c.JSON(http.StatusOK, post)
}

// UpdatePost handles PUT /posts/:id
func (h *PostHandler) UpdatePost(c *gin.Context) {
	ctx := c.Request.Context()

	postID := c.Param("id")
	postIDUUID, err := uuid.Parse(postID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid post ID"})
		return
	}

	var req request.PostRequest
	if err := c.ShouldBind(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input: " + err.Error()})
		return
	}

	userID, ok := c.Get("user_id")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	req.UserID = userID.(uuid.UUID)

	form, err := c.MultipartForm()
	if err == nil {
		req.Media = form.File["media"]
		req.Files = form.File["files"]
	}

	if err := h.postService.UpdatePost(ctx, postIDUUID, req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not update post: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Post updated successfully"})
}

// DeletePost handles DELETE /posts/:id
func (h *PostHandler) DeletePost(c *gin.Context) {
	ctx := c.Request.Context()

	postID := c.Param("id")
	postIDUUID, err := uuid.Parse(postID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid post ID"})
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	if err = h.postService.DeletePost(ctx, postIDUUID, userID.(uuid.UUID)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not delete post: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Post deleted successfully"})
}
