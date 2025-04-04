package http

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"link/internal/post/usecase"
	"link/pkg/common"
	"link/pkg/dto/req"
)

type PostHandler struct {
	postUsecase usecase.PostUsecase
}

func NewPostHandler(postUsecase usecase.PostUsecase) *PostHandler {
	return &PostHandler{postUsecase: postUsecase}
}

// TODO 게시물 생성 - 전체 사용자 게시물
func (h *PostHandler) CreatePost(c *gin.Context) {
	//TODO 게시물 생성
	userId, exists := c.Get("userId")
	if !exists {
		c.JSON(http.StatusUnauthorized, common.NewError(http.StatusUnauthorized, "인증되지 않은 사용자입니다.", nil))
		return
	}

	var request req.CreatePostRequest
	if err := c.ShouldBind(&request); err != nil {
		c.JSON(http.StatusBadRequest, common.NewError(http.StatusBadRequest, "잘못된 요청입니다", err))
		return
	}

	if request.Title == "" {
		c.JSON(http.StatusBadRequest, common.NewError(http.StatusBadRequest, "제목이 없습니다.", nil))
		return
	} else if request.Content == "" {
		c.JSON(http.StatusBadRequest, common.NewError(http.StatusBadRequest, "내용이 없습니다.", nil))
		return
	}

	postImageUrls, exists := c.Get("post_image_urls")
	if exists {
		imageUrls, ok := postImageUrls.([]string)
		if !ok {
			c.JSON(http.StatusBadRequest, common.NewError(http.StatusBadRequest, "이미지 처리 실패", nil))
			return
		}
		if len(imageUrls) > 0 {
			ptrUrls := make([]*string, len(imageUrls))
			for i := range imageUrls {
				ptrUrls[i] = &imageUrls[i]
			}
			request.Images = ptrUrls
		}
	}

	err := h.postUsecase.CreatePost(userId.(uint), &request)
	if err != nil {
		if appError, ok := err.(*common.AppError); ok {
			c.JSON(appError.StatusCode, common.NewError(appError.StatusCode, appError.Message, appError.Err))
		} else {
			c.JSON(http.StatusInternalServerError, common.NewError(http.StatusInternalServerError, "서버 에러", err))
		}
		return
	}

	c.JSON(http.StatusOK, common.NewResponse(http.StatusOK, "게시물 생성 완료", nil))
}

// TODO 게시물 리스트 조회
func (h *PostHandler) GetPosts(c *gin.Context) {
	// 인증된 사용자 확인
	userId, exists := c.Get("userId")
	if !exists {
		c.JSON(http.StatusUnauthorized, common.NewError(http.StatusUnauthorized, "인증되지 않은 사용자입니다.", nil))
		return
	}

	// 게시물 조회 파라미터 처리
	category := strings.ToLower(c.DefaultQuery("category", "public"))
	if category != "public" && category != "company" && category != "department" {
		category = "public"
	}

	viewType := strings.ToLower(c.DefaultQuery("view_type", "infinite"))
	if viewType != "infinite" && viewType != "pagination" {
		viewType = "infinite"
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
	var cursor *req.Cursor

	if strings.ToLower(viewType) == "infinite" && cursorParam == "" {
		cursor = nil //첫요청
		page = 1
	} else if strings.ToLower(viewType) == "infinite" {
		var tempCursor req.Cursor
		if err := json.Unmarshal([]byte(cursorParam), &tempCursor); err != nil {
			c.JSON(http.StatusBadRequest, common.NewError(http.StatusBadRequest, "유효하지 않은 커서 값입니다.", err))
			return
		}
		//TODO kst로 받은걸 utc로 변환
		//            "next_cursor": "2024-11-20 11:36:59",

		if sort == "created_at" && tempCursor.CreatedAt == "" {
			c.JSON(http.StatusBadRequest, common.NewError(http.StatusBadRequest, "커서는 sort와 같은 값이 있어야 합니다.", nil))
			return
		} else if sort == "like_count" && tempCursor.LikeCount == "" {
			c.JSON(http.StatusBadRequest, common.NewError(http.StatusBadRequest, "커서는 sort와 같은 값이 있어야 합니다.", nil))
			return
		} else if sort == "comments_count" && tempCursor.CommentsCount == "" {
			c.JSON(http.StatusBadRequest, common.NewError(http.StatusBadRequest, "커서는 sort와 같은 값이 있어야 합니다.", nil))
			return
		} else if sort == "id" && tempCursor.ID == "" {
			c.JSON(http.StatusBadRequest, common.NewError(http.StatusBadRequest, "커서는 sort와 같은 값이 있어야 합니다.", nil))
			return
		}

		cursor = &tempCursor

	} else if strings.ToLower(viewType) == "pagination" && cursorParam != "" {
		c.JSON(http.StatusBadRequest, common.NewError(http.StatusBadRequest, "페이지네이션 타입인데 커서가 있습니다.", nil))
		return
	}

	var companyId, departmentId uint
	companyIdValue, _ := strconv.ParseUint(c.DefaultQuery("company_id", "0"), 10, 32)
	companyId = uint(companyIdValue)
	departmentIdValue, _ := strconv.ParseUint(c.DefaultQuery("department_id", "0"), 10, 32)
	departmentId = uint(departmentIdValue)
	if strings.ToLower(category) == "company" {
		if companyId == 0 {
			c.JSON(http.StatusBadRequest, common.NewError(http.StatusBadRequest, "회사 게시물 조회 시 company_id가 필요합니다.", nil))
			return
		}
	} else if strings.ToLower(category) == "department" {

		if departmentId == 0 || companyId == 0 {
			c.JSON(http.StatusBadRequest, common.NewError(http.StatusBadRequest, "부서 게시물 조회 시 department_id와 company_id가 필요합니다.", nil))
			return
		}
	} else if strings.ToLower(category) == "public" {
		if companyId != 0 || departmentId != 0 {
			c.JSON(http.StatusBadRequest, common.NewError(http.StatusBadRequest, "PUBLIC 게시물은 company_id와 department_id가 없어야 합니다.", nil))
			return
		}
	}

	queryParams := req.GetPostQueryParams{
		Category:     strings.ToLower(category),
		Page:         page,
		Limit:        limit,
		Order:        order,
		Sort:         sort,
		ViewType:     strings.ToLower(viewType),
		Cursor:       cursor,
		CompanyId:    companyId,
		DepartmentId: departmentId,
	}

	// 게시물 조회
	posts, err := h.postUsecase.GetPosts(userId.(uint), queryParams)
	if err != nil {
		if appError, ok := err.(*common.AppError); ok {
			c.JSON(appError.StatusCode, common.NewError(appError.StatusCode, appError.Message, appError.Err))
		} else {
			c.JSON(http.StatusInternalServerError, common.NewError(http.StatusInternalServerError, "서버 에러", err))
		}
		return
	}

	c.JSON(http.StatusOK, common.NewResponse(http.StatusOK, "게시물 조회 완료", posts))
}

// TODO 게시물 상세보기 - 전체 사용자
func (h *PostHandler) GetPost(c *gin.Context) {
	userId, exists := c.Get("userId")
	if !exists {
		c.JSON(http.StatusUnauthorized, common.NewError(http.StatusUnauthorized, "인증되지 않은 사용자입니다.", nil))
		return
	}

	postId, err := strconv.Atoi(c.Param("postid"))
	if err != nil {
		c.JSON(http.StatusBadRequest, common.NewError(http.StatusBadRequest, "게시물 아이디 처리 실패", err))
		return
	}

	post, err := h.postUsecase.GetPost(userId.(uint), uint(postId))
	if err != nil {
		if appError, ok := err.(*common.AppError); ok {
			c.JSON(appError.StatusCode, common.NewError(appError.StatusCode, appError.Message, appError.Err))
		} else {
			c.JSON(http.StatusInternalServerError, common.NewError(http.StatusInternalServerError, "서버 에러", err))
		}
		return
	}

	c.JSON(http.StatusOK, common.NewResponse(http.StatusOK, "게시물 조회 완료", post))
}

// TODO : 게시물 조회수 증가
func (h *PostHandler) IncreasePostViewCount(c *gin.Context) {
	//헤더에 req.ip 가져오기
	ip := c.ClientIP()

	userId, exists := c.Get("userId")
	if !exists {
		c.JSON(http.StatusUnauthorized, common.NewError(http.StatusUnauthorized, "인증되지 않은 사용자입니다.", nil))
		return
	}

	postId, err := strconv.Atoi(c.Param("postid"))
	if err != nil {
		c.JSON(http.StatusBadRequest, common.NewError(http.StatusBadRequest, "게시물 아이디 처리 실패", err))
		return
	}

	err = h.postUsecase.IncreasePostViewCount(userId.(uint), uint(postId), ip)
	if err != nil {
		if appError, ok := err.(*common.AppError); ok {
			c.JSON(appError.StatusCode, common.NewError(appError.StatusCode, appError.Message, appError.Err))
		} else {
			c.JSON(http.StatusInternalServerError, common.NewError(http.StatusInternalServerError, "서버 에러", err))
		}
		return
	}

	c.JSON(http.StatusOK, common.NewResponse(http.StatusOK, "게시물 조회수 증가 완료", nil))
}

// TODO : 게시물 조회수 가져오기
func (h *PostHandler) GetPostViewCount(c *gin.Context) {

	userId, exists := c.Get("userId")
	if !exists {
		c.JSON(http.StatusUnauthorized, common.NewError(http.StatusUnauthorized, "인증되지 않은 사용자입니다.", nil))
		return
	}

	postId, err := strconv.Atoi(c.Param("postid"))
	if err != nil {
		c.JSON(http.StatusBadRequest, common.NewError(http.StatusBadRequest, "게시물 아이디 처리 실패", err))
		return
	}

	viewCount, err := h.postUsecase.GetPostViewCount(userId.(uint), uint(postId))
	if err != nil {
		if appError, ok := err.(*common.AppError); ok {
			c.JSON(appError.StatusCode, common.NewError(appError.StatusCode, appError.Message, appError.Err))
		} else {
			c.JSON(http.StatusInternalServerError, common.NewError(http.StatusInternalServerError, "서버 에러", err))
		}
		return
	}

	c.JSON(http.StatusOK, common.NewResponse(http.StatusOK, "게시물 조회수 가져오기 완료", viewCount))
}

// TODO 게시물 수정
func (h *PostHandler) UpdatePost(c *gin.Context) {
	userId, exists := c.Get("userId")
	if !exists {
		c.JSON(http.StatusUnauthorized, common.NewError(http.StatusUnauthorized, "인증되지 않은 사용자입니다.", nil))
		return
	}

	postId, err := strconv.Atoi(c.Param("postid"))
	if err != nil {
		c.JSON(http.StatusBadRequest, common.NewError(http.StatusBadRequest, "게시물 아이디 처리 실패", err))
		return
	}

	var request req.UpdatePostRequest
	if err := c.ShouldBind(&request); err != nil {
		c.JSON(http.StatusBadRequest, common.NewError(http.StatusBadRequest, "잘못된 요청입니다", err))
		return
	}

	postImageUrls, exists := c.Get("post_image_urls")
	if exists {
		imageUrls, ok := postImageUrls.([]string)
		if !ok {
			c.JSON(http.StatusBadRequest, common.NewError(http.StatusBadRequest, "이미지 처리 실패", nil))
			return
		}
		if len(imageUrls) > 0 {
			request.Images = imageUrls
		}
	}

	err = h.postUsecase.UpdatePost(userId.(uint), uint(postId), &request)
	if err != nil {
		if appError, ok := err.(*common.AppError); ok {
			c.JSON(appError.StatusCode, common.NewError(appError.StatusCode, appError.Message, appError.Err))
		} else {
			c.JSON(http.StatusInternalServerError, common.NewError(http.StatusInternalServerError, "서버 에러", err))
		}
		return
	}

	c.JSON(http.StatusOK, common.NewResponse(http.StatusOK, "게시물 수정 완료", nil))
}

// TODO 게시물 삭제
func (h *PostHandler) DeletePost(c *gin.Context) {
	userId, exists := c.Get("userId")
	if !exists {
		c.JSON(http.StatusUnauthorized, common.NewError(http.StatusUnauthorized, "인증되지 않은 사용자입니다.", nil))
		return
	}

	postId, err := strconv.Atoi(c.Param("postid"))
	if err != nil {
		c.JSON(http.StatusBadRequest, common.NewError(http.StatusBadRequest, "게시물 아이디 처리 실패", err))
		return
	}

	err = h.postUsecase.DeletePost(userId.(uint), uint(postId))
	if err != nil {
		if appError, ok := err.(*common.AppError); ok {
			c.JSON(appError.StatusCode, common.NewError(appError.StatusCode, appError.Message, appError.Err))
		} else {
			c.JSON(http.StatusInternalServerError, common.NewError(http.StatusInternalServerError, "서버 에러", err))
		}
		return
	}

	c.JSON(http.StatusOK, common.NewResponse(http.StatusOK, "게시물 삭제 완료", nil))
}
