package http

import (
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"link/internal/user/usecase"
	"link/pkg/common"
	"link/pkg/dto/req"
	"link/pkg/dto/res"
)

type UserHandler struct {
	userUsecase usecase.UserUsecase
}

// RegisterUserHandler는 회원가입 핸들러를 생성합니다.
func NewUserHandler(userUsecase usecase.UserUsecase) *UserHandler {
	return &UserHandler{userUsecase: userUsecase}
}

// ! 회원가입 핸들러

// TODO 회원가입할 때 userProfile 빈값들로 일단 생성
func (h *UserHandler) RegisterUser(c *gin.Context) {
	var request req.RegisterUserRequest

	// 요청 바디 검증
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, common.NewError(http.StatusBadRequest, "잘못된 요청입니다."))
		return
	}

	createdUser, err := h.userUsecase.RegisterUser(&request)
	if err != nil {
		if appError, ok := err.(*common.AppError); ok {
			c.JSON(appError.StatusCode, common.NewError(appError.StatusCode, appError.Message))
		} else {
			c.JSON(http.StatusInternalServerError, common.NewError(http.StatusInternalServerError, "서버 에러"))
		}
		return
	}

	// 유스케이스에서 반환된 엔티티를 응답 DTO로 변환
	response := res.RegisterUserResponse{
		ID:       *createdUser.ID,
		Name:     *createdUser.Name,
		Email:    *createdUser.Email,
		Phone:    *createdUser.Phone,
		Nickname: *createdUser.Nickname,
		Role:     uint(createdUser.Role),
	}

	// 성공 응답
	c.JSON(http.StatusCreated, common.NewResponse(http.StatusCreated, "회원가입 완료", response))
}

// ! 이메일 검증 핸들러
func (h *UserHandler) ValidateEmail(c *gin.Context) {
	// 쿼리 파라미터에서 이메일 추출
	email := c.Query("email")

	// 이메일 파라미터가 없는 경우, 잘못된 요청 처리
	if email == "" {
		c.JSON(http.StatusBadRequest, common.NewError(http.StatusBadRequest, "이메일이 입력되지 않았습니다."))
		return
	}

	// 이메일 유효성 검증 처리
	if err := h.userUsecase.ValidateEmail(email); err != nil {
		if appError, ok := err.(*common.AppError); ok {
			c.JSON(appError.StatusCode, common.NewError(appError.StatusCode, appError.Message))
		} else {
			c.JSON(http.StatusInternalServerError, common.NewError(http.StatusInternalServerError, "서버 에러"))
		}
		return
	}

	// 검증 성공 응답
	c.JSON(http.StatusOK, common.NewResponse(http.StatusOK, "이메일 사용 가능", nil))
}

// TODO 닉네임 중복확인
func (h *UserHandler) ValidateNickname(c *gin.Context) {
	nickname := c.Query("nickname")

	if nickname == "" {
		c.JSON(http.StatusBadRequest, common.NewError(http.StatusBadRequest, "닉네임이 입력되지 않았습니다"))
		return
	}

	err := h.userUsecase.ValidateNickname(nickname)
	if err != nil {
		if appError, ok := err.(*common.AppError); ok {
			c.JSON(appError.StatusCode, common.NewError(appError.StatusCode, appError.Message))
		} else {
			c.JSON(http.StatusInternalServerError, common.NewError(http.StatusInternalServerError, "서버 에러"))
		}
		return
	}

	c.JSON(http.StatusOK, common.NewResponse(http.StatusOK, "사용 가능한 닉네임입니다", nil))
}

// TODO UserProfile 있으면
func (h *UserHandler) GetUserInfo(c *gin.Context) {
	//params 받아야지
	userId, exists := c.Get("userId")
	targetUserId := c.Param("id")
	if !exists {
		c.JSON(http.StatusUnauthorized, common.NewError(http.StatusUnauthorized, "인증되지 않은 요청입니다"))
		return
	}

	targetUserIdUint, err := strconv.ParseUint(targetUserId, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, common.NewError(http.StatusBadRequest, "유효하지 않은 사용자 ID입니다"))
		return
	}

	user, err := h.userUsecase.GetUserInfo(userId.(uint), uint(targetUserIdUint), "user")
	if err != nil {
		if appError, ok := err.(*common.AppError); ok {
			c.JSON(appError.StatusCode, common.NewError(appError.StatusCode, appError.Message))
		} else {
			c.JSON(http.StatusInternalServerError, common.NewError(http.StatusInternalServerError, "서버 에러"))
		}
		return
	}

	response := res.GetUserByIdResponse{
		ID:       *user.ID,
		Email:    *user.Email,
		Name:     *user.Name,
		Phone:    *user.Phone,
		Nickname: *user.Nickname,
		Role:     uint(user.Role),
		UserProfile: res.UserProfile{
			Image:         user.UserProfile.Image,
			Birthday:      user.UserProfile.Birthday,
			CompanyID:     user.UserProfile.CompanyID,
			DepartmentIds: user.UserProfile.DepartmentIds,
			TeamIds:       user.UserProfile.TeamIds,
			PositionId:    user.UserProfile.PositionId,
		},
		CreatedAt: *user.CreatedAt,
		UpdatedAt: *user.UpdatedAt,
	}

	c.JSON(http.StatusOK, common.NewResponse(http.StatusOK, "사용자 조회 성공", response))
}

func (h *UserHandler) UpdateUserInfo(c *gin.Context) {
	requestUserId, exists := c.Get("userId")
	if !exists {
		c.JSON(http.StatusUnauthorized, common.NewError(http.StatusUnauthorized, "인증되지 않은 요청입니다"))
		return
	}

	targetUserId := c.Param("id")
	targetUserIdUint, err := strconv.ParseUint(targetUserId, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, common.NewError(http.StatusBadRequest, "유효하지 않은 사용자 ID입니다"))
		return
	}

	var request req.UpdateUserRequest
	if err := c.ShouldBind(&request); err != nil {
		c.JSON(http.StatusBadRequest, common.NewError(http.StatusBadRequest, "잘못된 요청입니다: "+err.Error()))
		return
	}

	// 미들웨어에서 처리한 이미지 URL 가져오기
	profileImageUrl, exists := c.Get("profile_image_url")
	if exists {
		imageURL := profileImageUrl.(string)
		request.Image = &imageURL
	}

	fmt.Printf("Received request: %+v\n", request)

	// DTO를 Usecase에 전달
	err = h.userUsecase.UpdateUserInfo(uint(targetUserIdUint), requestUserId.(uint), &request)
	if err != nil {
		if appError, ok := err.(*common.AppError); ok {
			c.JSON(appError.StatusCode, common.NewError(appError.StatusCode, appError.Message))
		} else {
			c.JSON(http.StatusInternalServerError, common.NewError(http.StatusInternalServerError, "서버 에러"))
		}
		return
	}

	c.JSON(http.StatusOK, common.NewResponse(http.StatusOK, "사용자 정보 수정 성공", nil))
}

// 사용자 정보 삭제 핸들러
func (h *UserHandler) DeleteUser(c *gin.Context) {
	requestUserId, exists := c.Get("userId")
	if !exists {
		c.JSON(http.StatusUnauthorized, common.NewError(http.StatusUnauthorized, "인증되지 않은 요청입니다"))
		return
	}

	targetUserId := c.Param("id")
	targetUserIdUint, err := strconv.ParseUint(targetUserId, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, common.NewError(http.StatusBadRequest, "유효하지 않은 사용자 ID입니다"))
		return
	}

	err = h.userUsecase.DeleteUser(uint(targetUserIdUint), requestUserId.(uint))
	if err != nil {
		if appError, ok := err.(*common.AppError); ok {
			c.JSON(appError.StatusCode, common.NewError(appError.StatusCode, appError.Message))
		} else {
			c.JSON(http.StatusInternalServerError, common.NewError(http.StatusInternalServerError, "서버 에러"))
		}
		return
	}

	c.JSON(http.StatusOK, common.NewResponse(http.StatusOK, "사용자 정보 삭제 성공", nil))
}

// TODO 사용자 검색 핸들러
// 사용자 검색 핸들러
func (h *UserHandler) SearchUser(c *gin.Context) {
	// 검색 요청 데이터 받기
	searchReq := req.SearchUserRequest{
		Email:    c.Query("email"),
		Name:     c.Query("name"),
		Nickname: c.Query("nickname"),
	}

	// 쿼리 내용이 아무것도 없는 경우
	if searchReq.Email == "" && searchReq.Name == "" && searchReq.Nickname == "" {
		c.JSON(http.StatusBadRequest, common.NewError(http.StatusBadRequest, "이메일 혹은 이름 혹은 닉네임이 입력되지 않았습니다."))
		return
	}

	// 사용자 검색
	users, err := h.userUsecase.SearchUser(&searchReq)
	if err != nil {
		if appError, ok := err.(*common.AppError); ok {
			c.JSON(appError.StatusCode, common.NewError(appError.StatusCode, appError.Message))
		} else {
			c.JSON(http.StatusInternalServerError, common.NewError(http.StatusInternalServerError, "서버 에러"))
		}
		return
	}

	if len(users) == 0 {
		c.JSON(http.StatusNotFound, common.NewError(http.StatusNotFound, "사용자를 찾을 수 없습니다"))
		return
	}

	var response []res.SearchUserResponse
	// 그룹 이름 또는 ID를 문자열 배열로 변환

	for _, user := range users {
		log.Printf("user: %v", user)
		userResponse := res.SearchUserResponse{
			ID:       *user.ID,
			Name:     *user.Name,
			Email:    *user.Email, // 민감 정보 포함할지 여부에 따라 처리
			Nickname: *user.Nickname,
			Phone:    *user.Phone,
			Role:     uint(user.Role),
			UserProfile: res.UserProfile{
				Image:         user.UserProfile.Image,
				Birthday:      user.UserProfile.Birthday,
				CompanyID:     user.UserProfile.CompanyID,
				DepartmentIds: user.UserProfile.DepartmentIds,
				TeamIds:       user.UserProfile.TeamIds,
				PositionId:    user.UserProfile.PositionId,
			},
			CreatedAt: *user.CreatedAt,
			UpdatedAt: *user.UpdatedAt,
		}

		response = append(response, userResponse)
	}

	c.JSON(http.StatusOK, common.NewResponse(http.StatusOK, "사용자 검색 성공", response))
}

// TODO 본인이 속한 회사 사용자 리스트 가져오기
func (h *UserHandler) GetUserByCompany(c *gin.Context) {
	requestUserId, exists := c.Get("userId")
	if !exists {
		c.JSON(http.StatusUnauthorized, common.NewError(http.StatusUnauthorized, "인증되지 않은 요청입니다"))
		return
	}

	users, err := h.userUsecase.GetUsersByCompany(requestUserId.(uint))
	if err != nil {
		if appError, ok := err.(*common.AppError); ok {
			c.JSON(appError.StatusCode, common.NewError(appError.StatusCode, appError.Message))
		} else {
			c.JSON(http.StatusInternalServerError, common.NewError(http.StatusInternalServerError, "서버 에러"))
		}
		return
	}

	var response []res.GetUserByIdResponse
	for _, user := range users {

		fmt.Println("response")
		fmt.Println(user.IsOnline)

		response = append(response, res.GetUserByIdResponse{
			ID:       *user.ID,
			Email:    *user.Email,
			Name:     *user.Name,
			Nickname: *user.Nickname,
			Phone:    *user.Phone,
			Role:     uint(user.Role),
			IsOnline: *user.IsOnline,
			UserProfile: res.UserProfile{
				Image:         user.UserProfile.Image,
				Birthday:      user.UserProfile.Birthday,
				CompanyID:     user.UserProfile.CompanyID,
				DepartmentIds: user.UserProfile.DepartmentIds,
				TeamIds:       user.UserProfile.TeamIds,
				PositionId:    user.UserProfile.PositionId,
			},
			CreatedAt: *user.CreatedAt,
			UpdatedAt: *user.UpdatedAt,
		})

	}
	// 회사 사용자 조회 성공 응답

	c.JSON(http.StatusOK, common.NewResponse(http.StatusOK, "회사 사용자 조회 성공", response))
}

// TODO 해당 부서에 속한 사용자 리스트 가져오기 (이후 디테일 잡을때)
func (h *UserHandler) GetUsersByDepartment(c *gin.Context) {
	departmentId := c.Param("departmentId")

	departmentIdUint, err := strconv.ParseUint(departmentId, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, common.NewError(http.StatusBadRequest, "유효하지 않은 부서 ID입니다"))
		return
	}

	users, err := h.userUsecase.GetUsersByDepartment(uint(departmentIdUint))
	if err != nil {
		if appError, ok := err.(*common.AppError); ok {
			c.JSON(appError.StatusCode, common.NewError(appError.StatusCode, appError.Message))
		} else {
			c.JSON(http.StatusInternalServerError, common.NewError(http.StatusInternalServerError, "서버 에러"))
		}
		return
	}

	c.JSON(http.StatusOK, common.NewResponse(http.StatusOK, "부서 사용자 조회 성공", users))
}
