package delivery

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"postService/internal/service"
	"postService/internal/transfer/request"
	"strconv"
)

type CommentHandler struct {
	s *service.CommentService
}

func (c *CommentHandler) GetByPostID(ctx *gin.Context) {
	pid, err := strconv.Atoi(ctx.Param("pid"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, "Error on extracting id")
		return
	}

	comments, err := c.s.GetCommentsByPostID(pid)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, "Could not fetch comments")
		return
	}

	ctx.JSON(http.StatusOK, comments)
}

func (c *CommentHandler) GetByID(ctx *gin.Context) {
	id, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		ctx.JSON(http.StatusBadRequest, "Error on extracting id")
		return
	}

	comment, err := c.s.GetCommentByID(id)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, "Could not fetch the comment")
		return
	}

	ctx.JSON(http.StatusOK, comment)
}

func (c *CommentHandler) CreateComment(ctx *gin.Context) {
	req := request.CommentRequest{}
	if err := ctx.ShouldBindJSON(req); err != nil {
		ctx.JSON(http.StatusBadRequest, "Could not bind request")
		return
	}

	err := c.s.CreateComment(req)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, "Could not create post")
		return
	}

	ctx.JSON(http.StatusCreated, "Successfully created a comment")
}

func (c *CommentHandler) UpdateComment(ctx *gin.Context) {

}

func (c *CommentHandler) DeleteComment(ctx *gin.Context) {

}
