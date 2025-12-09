package http

import (
	"fmt"
	"net/http"
	"postService/internal/config"
	"postService/internal/metrics"
	"postService/internal/service"
	"postService/internal/transfer/request"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const (
	MaxUploadSize    = 20 << 20 // 20 MB for whole request
	MaxMediaFileSize = 5 << 20  // 5 MB for media
	MaxOtherFileSize = 10 << 20 // 10 MB for other files
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
	metrics.HttpRequests.WithLabelValues("create_post").Inc()

	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, MaxUploadSize)

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
		for _, f := range form.File["media"] {
			if f.Size > MaxMediaFileSize {
				c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("media file %s is too large (max %d MB)", f.Filename, MaxMediaFileSize>>20)})
				return
			}
		}

		for _, f := range form.File["files"] {
			if f.Size > MaxOtherFileSize {
				c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("file %s is too large (max %d MB)", f.Filename, MaxOtherFileSize>>20)})
				return
			}
		}

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
	metrics.HttpRequests.WithLabelValues("get_posts").Inc()

	ctx := c.Request.Context()

	page, limit := parsePagination(c)

	posts, err := h.postService.GetPosts(ctx, page, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not fetch posts"})
		return
	}

	c.JSON(http.StatusOK, posts)
}

// GetPostsByUserID handles GET /posts/user/:id
func (h *PostHandler) GetPostsByUserID(c *gin.Context) {
	metrics.HttpRequests.WithLabelValues("get_posts_by_user").Inc()

	ctx := c.Request.Context()

	userIDParam := c.Param("id")
	userID, err := uuid.Parse(userIDParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	page, limit := parsePagination(c)

	posts, err := h.postService.GetPostsByUserID(ctx, userID, page, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not fetch posts for user"})
		return
	}

	c.JSON(http.StatusOK, posts)
}

// GetPostByID handles GET /posts/:id
func (h *PostHandler) GetPostByID(c *gin.Context) {
	metrics.HttpRequests.WithLabelValues("get_post_by_id").Inc()

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
	metrics.HttpRequests.WithLabelValues("update_post").Inc()

	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, MaxUploadSize)

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
		for _, f := range form.File["media"] {
			if f.Size > MaxMediaFileSize {
				c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("media file %s is too large (max %d MB)", f.Filename, MaxMediaFileSize>>20)})
				return
			}
		}

		for _, f := range form.File["files"] {
			if f.Size > MaxOtherFileSize {
				c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("file %s is too large (max %d MB)", f.Filename, MaxOtherFileSize>>20)})
				return
			}
		}

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
	metrics.HttpRequests.WithLabelValues("delete_post").Inc()

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

// helper to parse pagination params
func parsePagination(c *gin.Context) (int64, int64) {
	page, _ := strconv.ParseInt(c.DefaultQuery("page", "1"), 10, 64)
	limit, _ := strconv.ParseInt(c.DefaultQuery("limit", "10"), 10, 64)

	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 10
	}
	return page, limit
}
