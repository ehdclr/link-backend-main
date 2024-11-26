package http

import (
	"encoding/json"
	"fmt"
	"link/pkg/common"
	"link/pkg/dto/req"
	"net/http"
	"strconv"

	_commentUsecase "link/internal/comment/usecase"

	"github.com/gin-gonic/gin"
)

type CommentHandler struct {
	commentUsecase _commentUsecase.CommentUsecase
}

func NewCommentHandler(
	commentUsecase _commentUsecase.CommentUsecase) *CommentHandler {
	return &CommentHandler{commentUsecase: commentUsecase}
}

func (h *CommentHandler) CreateComment(c *gin.Context) {
	userId, exists := c.Get("userId")
	if !exists {
		fmt.Printf("인증되지 않은 사용자입니다.")
		c.JSON(http.StatusUnauthorized, common.NewError(http.StatusUnauthorized, "인증되지 않은 사용자입니다.", nil))
		return
	}

	var request req.CommentRequest
	if err := c.ShouldBind(&request); err != nil {
		fmt.Printf("잘못된 요청입니다: %v", err)
		c.JSON(http.StatusBadRequest, common.NewError(http.StatusBadRequest, "잘못된 요청입니다", err))
		return
	}

	if request.Content == "" {
		fmt.Printf("내용이 없습니다.")
		c.JSON(http.StatusBadRequest, common.NewError(http.StatusBadRequest, "내용이 없습니다.", nil))
		return
	}
	if request.PostID == 0 {
		fmt.Printf("게시글 ID가 없습니다.")
		c.JSON(http.StatusBadRequest, common.NewError(http.StatusBadRequest, "게시글 ID가 없습니다.", nil))
		return
	}

	if err := h.commentUsecase.CreateComment(userId.(uint), request); err != nil {
		if appError, ok := err.(*common.AppError); ok {
			c.JSON(appError.StatusCode, common.NewError(appError.StatusCode, appError.Message, appError.Err))
		} else {
			c.JSON(http.StatusInternalServerError, common.NewError(http.StatusInternalServerError, "댓글 생성 실패", err))
		}
		return
	}

	c.JSON(http.StatusOK, common.NewResponse(http.StatusOK, "댓글 생성 성공", nil))
}

func (h *CommentHandler) CreateReply(c *gin.Context) {
	userId, exists := c.Get("userId")
	if !exists {
		fmt.Printf("인증되지 않은 사용자입니다.")
		c.JSON(http.StatusUnauthorized, common.NewError(http.StatusUnauthorized, "인증되지 않은 사용자입니다.", nil))
		return
	}

	var request req.ReplyRequest
	if err := c.ShouldBind(&request); err != nil {
		fmt.Printf("잘못된 요청입니다: %v", err)
		c.JSON(http.StatusBadRequest, common.NewError(http.StatusBadRequest, "잘못된 요청입니다", err))
		return
	}

	if request.Content == "" {
		fmt.Printf("내용이 없습니다.")
		c.JSON(http.StatusBadRequest, common.NewError(http.StatusBadRequest, "내용이 없습니다.", nil))
		return
	}
	if request.PostID == 0 {
		fmt.Printf("게시글 ID가 없습니다.")
		c.JSON(http.StatusBadRequest, common.NewError(http.StatusBadRequest, "게시글 ID가 없습니다.", nil))
		return
	}

	if request.ParentID == 0 {
		fmt.Printf("부모 댓글 ID가 없습니다.")
		c.JSON(http.StatusBadRequest, common.NewError(http.StatusBadRequest, "부모 댓글 ID가 없습니다.", nil))
		return
	}

	if err := h.commentUsecase.CreateReply(userId.(uint), request); err != nil {
		if appError, ok := err.(*common.AppError); ok {
			fmt.Printf("대댓글 생성 실패: %v", appError)
			c.JSON(appError.StatusCode, common.NewError(appError.StatusCode, appError.Message, appError.Err))
		} else {
			fmt.Printf("대댓글 생성 실패: %v", err)
			c.JSON(http.StatusInternalServerError, common.NewError(http.StatusInternalServerError, "대댓글 생성 실패", err))
		}
		return
	}

	c.JSON(http.StatusOK, common.NewResponse(http.StatusOK, "대댓글 생성 성공", nil))
}

func (h *CommentHandler) GetComments(c *gin.Context) {
	userId, exists := c.Get("userId")
	if !exists {
		fmt.Printf("인증되지 않은 사용자입니다.")
		c.JSON(http.StatusUnauthorized, common.NewError(http.StatusUnauthorized, "인증되지 않은 사용자입니다.", nil))
		return
	}

	postId, err := strconv.Atoi(c.Param("post_id"))
	if err != nil || postId < 1 {
		fmt.Printf("게시물 ID가 유효하지 않습니다: %v", err)
		c.JSON(http.StatusBadRequest, common.NewError(http.StatusBadRequest, "게시물 ID가 유효하지 않습니다.", err))
		return
	}

	page, err := strconv.Atoi(c.DefaultQuery("page", "1"))
	if err != nil || page < 1 {
		page = 1
	}

	limit, err := strconv.Atoi(c.DefaultQuery("limit", "10"))
	if err != nil || limit < 1 || limit > 100 {
		limit = 10
	}

	sort := c.DefaultQuery("sort", "created_at")
	if sort != "created_at" && sort != "like_count" && sort != "comments_count" && sort != "id" {
		sort = "created_at"
	}

	order := c.DefaultQuery("order", "desc")
	if order != "asc" && order != "desc" {
		order = "desc"
	}

	cursorParam := c.Query("cursor")
	var cursor *req.CommentCursor

	if cursorParam != "" {
		if err := json.Unmarshal([]byte(cursorParam), &cursor); err != nil {
			fmt.Printf("커서 파싱 실패: %v", err)
			c.JSON(http.StatusBadRequest, common.NewError(http.StatusBadRequest, "유효하지 않은 커서 값입니다.", err))
			return
		}

		if sort == "created_at" && cursor.CreatedAt == "" {
			fmt.Printf("커서는 sort와 같은 값이 있어야 합니다.")
			c.JSON(http.StatusBadRequest, common.NewError(http.StatusBadRequest, "커서는 sort와 같은 값이 있어야 합니다.", nil))
			return
		} else if sort == "id" && cursor.ID == "" {
			fmt.Printf("커서는 sort와 같은 값이 있어야 합니다.")
			c.JSON(http.StatusBadRequest, common.NewError(http.StatusBadRequest, "커서는 sort와 같은 값이 있어야 합니다.", nil))
			return
		} else if sort == "like_count" && cursor.LikeCount == "" {
			fmt.Printf("커서는 sort와 같은 값이 있어야 합니다.")
			c.JSON(http.StatusBadRequest, common.NewError(http.StatusBadRequest, "커서는 sort와 같은 값이 있어야 합니다.", nil))
			return
		}
	}

	queryParams := req.GetCommentQueryParams{
		Page:   page,
		Limit:  limit,
		Sort:   sort,
		Order:  order,
		Cursor: cursor,
		PostID: uint(postId),
	}

	comments, err := h.commentUsecase.GetComments(userId.(uint), queryParams)
	if err != nil {
		if appError, ok := err.(*common.AppError); ok {
			fmt.Printf("댓글 조회 실패: %v", appError)
			c.JSON(appError.StatusCode, common.NewError(appError.StatusCode, appError.Message, appError.Err))
		} else {
			fmt.Printf("댓글 조회 실패: %v", err)
			c.JSON(http.StatusInternalServerError, common.NewError(http.StatusInternalServerError, "댓글 조회 실패", err))
		}
		return
	}

	c.JSON(http.StatusOK, common.NewResponse(http.StatusOK, "댓글 조회 성공", comments))
}
