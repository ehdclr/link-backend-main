package usecase

import (
	"log"
	"net/http"
	"time"

	_companyRepo "link/internal/company/repository"
	"link/internal/user/entity"
	_userRepo "link/internal/user/repository"
	"link/pkg/common"
	"link/pkg/dto/req"
	"link/pkg/dto/res"

	_utils "link/pkg/util"
)

// UserUsecase 인터페이스 정의
type UserUsecase interface {
	RegisterUser(request *req.RegisterUserRequest) (*res.RegisterUserResponse, error)
	ValidateEmail(email string) error
	ValidateNickname(nickname string) error
	GetUserInfo(targetUserId, requestUserId uint, role string) (*res.GetUserByIdResponse, error)
	GetUserMyInfo(userId uint) (*entity.User, error)

	UpdateUserInfo(requestUserId, targetUserId uint, request *req.UpdateUserRequest) error
	DeleteUser(targetUserId, requestUserId uint) error
	SearchUser(request *req.SearchUserRequest) ([]res.SearchUserResponse, error)

	UpdateUserOnlineStatus(userId uint, online bool) error

	//TODO 복합 관련
	GetUsersByCompany(requestUserId uint, query *req.UserQuery) ([]res.GetUserByIdResponse, error)
	GetUsersByDepartment(departmentId uint) ([]entity.User, error)
}

type userUsecase struct {
	userRepo    _userRepo.UserRepository
	companyRepo _companyRepo.CompanyRepository
}

// NewUserUsecase 생성자
func NewUserUsecase(repo _userRepo.UserRepository, companyRepo _companyRepo.CompanyRepository) UserUsecase {
	return &userUsecase{userRepo: repo, companyRepo: companyRepo}
}

// TODO 사용자 생성 - 무조건 일반 사용자
func (u *userUsecase) RegisterUser(request *req.RegisterUserRequest) (*res.RegisterUserResponse, error) {

	hashedPassword, err := _utils.HashPassword(request.Password)
	if err != nil {
		log.Printf("비밀번호 해싱 오류: %v", err)
		return nil, common.NewError(http.StatusInternalServerError, "비밀번호 해쉬화에 실패했습니다")
	}
	user := &entity.User{
		Name:     &request.Name,
		Email:    &request.Email,
		Password: &hashedPassword,
		Nickname: &request.Nickname,
		Phone:    &request.Phone,
		Role:     entity.RoleUser,
		UserProfile: &entity.UserProfile{
			IsSubscribed: false,
		},
	}

	if err := u.userRepo.CreateUser(user); err != nil {
		log.Printf("사용자 생성 오류: %v", err)
		return nil, common.NewError(http.StatusInternalServerError, "사용자 생성에 실패했습니다")
	}

	response := res.RegisterUserResponse{
		ID:       _utils.GetValueOrDefault(user.ID, 0),
		Name:     _utils.GetValueOrDefault(user.Name, ""),
		Email:    _utils.GetValueOrDefault(user.Email, ""),
		Phone:    _utils.GetValueOrDefault(user.Phone, ""),
		Nickname: _utils.GetValueOrDefault(user.Nickname, ""),
		Role:     uint(_utils.GetValueOrDefault(&user.Role, entity.RoleUser)),
	}

	return &response, nil
}

// TODO 이메일 중복 체크
func (u *userUsecase) ValidateEmail(email string) error {
	user, err := u.userRepo.ValidateEmail(email)
	if err != nil {
		//TODO ErrRecordNotFound면 오류 안나게 하기
		return common.NewError(http.StatusInternalServerError, "이메일 확인 중 오류가 발생했습니다")
	}
	if user != nil {
		return common.NewError(http.StatusBadRequest, "이미 사용 중인 이메일입니다")
	}
	return nil
}

// TODO 닉네임 중복확인
func (u *userUsecase) ValidateNickname(nickname string) error {
	user, err := u.userRepo.ValidateNickname(nickname)
	if err != nil {
		//TODO ErrRecordNotFound면 오류 안나게 하기
		log.Printf("닉네임 중복확인에 실패했습니다: %v", err)
		return common.NewError(http.StatusInternalServerError, "닉네임 중복확인에 실패했습니다")
	}

	if user != nil {
		return common.NewError(http.StatusBadRequest, "이미 사용 중인 닉네임입니다")
	}

	return nil
}

// TODO 사용자 정보 가져오기 (다른 사용자)
func (u *userUsecase) GetUserInfo(requestUserId, targetUserId uint, role string) (*res.GetUserByIdResponse, error) {
	requestUser, err := u.userRepo.GetUserByID(requestUserId)
	if err != nil {
		log.Printf("요청 사용자 조회에 실패했습니다: %v", err)
		return nil, common.NewError(http.StatusInternalServerError, "사용자 조회에 실패했습니다")
	}

	user, err := u.userRepo.GetUserByID(targetUserId)
	if err != nil {
		log.Printf("사용자 조회에 실패했습니다: %v", err)
		return nil, common.NewError(http.StatusInternalServerError, "사용자 조회에 실패했습니다")
	}

	if (requestUser.Role >= entity.RoleCompanyManager) && (user.Role <= entity.RoleAdmin) {
		log.Printf("권한이 없는 사용자가 관리자 정보를 조회하려 했습니다: 요청자 ID %d, 대상 ID %d", requestUserId, targetUserId)
		return nil, common.NewError(http.StatusForbidden, "권한이 없습니다")
	}

	//TODO 캐시 있다면 조회 없으면 데이터베이스 조회
	//TODO 캐시에는 이미지 온라인 상태 , birthday 있음

	response := res.GetUserByIdResponse{
		ID:              _utils.GetValueOrDefault(user.ID, 0),
		Email:           _utils.GetValueOrDefault(user.Email, ""),
		Name:            _utils.GetValueOrDefault(user.Name, ""),
		Phone:           _utils.GetValueOrDefault(user.Phone, ""),
		Nickname:        _utils.GetValueOrDefault(user.Nickname, ""),
		Role:            uint(_utils.GetValueOrDefault(&user.Role, entity.RoleUser)),
		Image:           _utils.GetValueOrDefault(user.UserProfile.Image, ""),
		Birthday:        _utils.GetValueOrDefault(&user.UserProfile.Birthday, ""),
		CompanyID:       _utils.GetValueOrDefault(user.UserProfile.CompanyID, 0),
		CompanyName:     _utils.GetFirstOrEmpty(_utils.ExtractValuesFromMapSlice[string]([]*map[string]interface{}{user.UserProfile.Company}, "name"), ""),
		DepartmentIds:   _utils.ExtractValuesFromMapSlice[uint](user.UserProfile.Departments, "id"),
		DepartmentNames: _utils.ExtractValuesFromMapSlice[string](user.UserProfile.Departments, "name"),
		TeamIds:         _utils.ExtractValuesFromMapSlice[uint](user.UserProfile.Teams, "id"),
		TeamNames:       _utils.ExtractValuesFromMapSlice[string](user.UserProfile.Teams, "name"),
		PositionId:      _utils.GetValueOrDefault(user.UserProfile.PositionId, 0),
		PositionName:    _utils.GetFirstOrEmpty(_utils.ExtractValuesFromMapSlice[string]([]*map[string]interface{}{user.UserProfile.Position}, "name"), ""),
		CreatedAt:       _utils.GetValueOrDefault(user.CreatedAt, time.Time{}),
		UpdatedAt:       _utils.GetValueOrDefault(user.UpdatedAt, time.Time{}),
	}

	return &response, nil
}

// TODO 본인 정보 가져오기
func (u *userUsecase) GetUserMyInfo(userId uint) (*entity.User, error) {
	user, err := u.userRepo.GetUserByID(userId)
	if err != nil {
		log.Printf("사용자 조회에 실패했습니다: %v", err)
		return nil, common.NewError(http.StatusInternalServerError, "사용자 조회에 실패했습니다")
	}

	return user, nil
}

// TODO 사용자 정보 업데이트 -> 확인해야함 (관리자용으로 나중에 빼기)
func (u *userUsecase) UpdateUserInfo(targetUserId, requestUserId uint, request *req.UpdateUserRequest) error {
	// 요청 사용자 조회
	requestUser, err := u.userRepo.GetUserByID(requestUserId)
	if err != nil {
		log.Printf("요청 사용자 조회에 실패했습니다: %v", err)
		return common.NewError(http.StatusInternalServerError, "요청 사용자를 찾을 수 없습니다")
	}

	// 대상 사용자 조회
	targetUser, err := u.userRepo.GetUserByID(targetUserId)
	if err != nil {
		log.Printf("대상 사용자 조회에 실패했습니다: %v", err)
		return common.NewError(http.StatusInternalServerError, "대상 사용자를 찾을 수 없습니다")
	}

	//TODO 본인이 아니거나 시스템 관리자가 아니라면 업데이트 불가
	if requestUserId != targetUserId && requestUser.Role != entity.RoleAdmin && requestUser.Role != entity.RoleSubAdmin {
		log.Printf("권한이 없는 사용자가 사용자 정보를 업데이트하려 했습니다: 요청자 ID %d, 대상 ID %d", requestUserId, targetUserId)
		return common.NewError(http.StatusForbidden, "권한이 없습니다")
	}

	//TODO 루트 관리자는 절대 변경 불가
	if targetUser.Role == entity.RoleAdmin {
		log.Printf("루트 관리자는 변경할 수 없습니다")
		return common.NewError(http.StatusForbidden, "루트 관리자는 변경할 수 없습니다")
	}

	//TODO 관리자가 아니면, Role 변경 불가
	if requestUser.Role != entity.RoleAdmin && requestUser.Role != entity.RoleSubAdmin && request.Role != nil {
		log.Printf("권한이 없는 사용자가 권한을 변경하려 했습니다: 요청자 ID %d, 대상 ID %d", requestUserId, targetUserId)
		return common.NewError(http.StatusForbidden, "권한이 없습니다")
	}

	// 업데이트할 필드 준비
	userUpdates := make(map[string]interface{})
	profileUpdates := make(map[string]interface{})

	if request.Name != nil {
		userUpdates["name"] = *request.Name
	}
	if request.Email != nil {
		userUpdates["email"] = *request.Email
	}
	if request.Password != nil {
		hashedPassword, err := _utils.HashPassword(*request.Password)
		if err != nil {
			return common.NewError(http.StatusInternalServerError, "비밀번호 해싱 실패")
		}
		userUpdates["password"] = hashedPassword
	}
	if request.Role != nil {
		userUpdates["role"] = *request.Role
	}
	if request.Nickname != nil {
		userUpdates["nickname"] = *request.Nickname
	}
	if request.Phone != nil {
		userUpdates["phone"] = *request.Phone
	}
	if request.Birthday != nil {
		profileUpdates["birthday"] = *request.Birthday
	}
	if request.CompanyID != nil {
		profileUpdates["company_id"] = *request.CompanyID
	}
	if request.DepartmentID != nil {
		profileUpdates["department_id"] = *request.DepartmentID
	}
	if request.TeamID != nil {
		profileUpdates["team_id"] = *request.TeamID
	}
	if request.PositionID != nil {
		profileUpdates["position_id"] = *request.PositionID
	}
	if request.Image != nil {
		profileUpdates["image"] = *request.Image
	}

	//TODO db 업데이트 하고
	err = u.userRepo.UpdateUser(targetUserId, userUpdates, profileUpdates)
	if err != nil {
		log.Printf("Postgres 사용자 업데이트에 실패했습니다: %v", err)
		return common.NewError(http.StatusInternalServerError, "사용자 업데이트에 실패했습니다")
	}
	// //TODO redis 캐시 업데이트
	// err = u.userRepo.UpdateCacheUser(targetUserId, profileUpdates)
	// if err != nil {
	// 	log.Printf("Redis 사용자 캐시 업데이트에 실패했습니다: %v", err)
	// 	return common.NewError(http.StatusInternalServerError, "사용자 캐시 업데이트에 실패했습니다")
	// }
	// Persistence 레이어로 업데이트 요청 전달
	return nil
}

// TODO 사용자 정보 삭제
// !시스템관리자랑 본인만가능
func (u *userUsecase) DeleteUser(targetUserId, requestUserId uint) error {
	requestUser, err := u.userRepo.GetUserByID(requestUserId)
	if err != nil {
		log.Printf("요청 사용자 조회에 실패했습니다: %v", err)
		return common.NewError(http.StatusInternalServerError, "요청 사용자를 찾을 수 없습니다")
	}

	// 대상 사용자 조회
	targetUser, err := u.userRepo.GetUserByID(targetUserId)
	if err != nil {
		log.Printf("대상 사용자 조회에 실패했습니다: %v", err)
		return common.NewError(http.StatusInternalServerError, "대상 사용자를 찾을 수 없습니다")
	}

	//TODO 본인이 아니거나 시스템 관리자가 아니라면 삭제 불가
	if requestUserId != targetUserId && requestUser.Role != entity.RoleAdmin && requestUser.Role != entity.RoleSubAdmin {
		log.Printf("권한이 없는 사용자가 사용자 정보를 삭제하려 했습니다: 요청자 ID %d, 대상 ID %d", requestUserId, targetUserId)
		return common.NewError(http.StatusForbidden, "권한이 없습니다")
	}

	//TODO 삭제하려는 대상에 시스템관리자는 불가능함
	if targetUser.Role == entity.RoleAdmin {
		log.Printf("시스템 관리자는 삭제 불가능합니다: 요청자 ID %d, 대상 ID %d", requestUserId, targetUserId)
		return common.NewError(http.StatusForbidden, "시스템 관리자 계정은 삭제가 불가능합니다")
	}

	return u.userRepo.DeleteUser(targetUserId)
}

// TODO 사용자 검색
func (u *userUsecase) SearchUser(request *req.SearchUserRequest) ([]res.SearchUserResponse, error) {
	// 사용자 저장소에서 검색
	//request를 entity.User로 변환
	user := entity.User{
		Email:    &request.Email,
		Name:     &request.Name,
		Nickname: &request.Nickname,
	}

	users, err := u.userRepo.SearchUser(&user)
	if err != nil {
		log.Printf("사용자 검색에 실패했습니다: %v", err)
		return nil, common.NewError(http.StatusInternalServerError, "사용자 검색에 실패했습니다")
	}

	var response []res.SearchUserResponse

	for _, user := range users {

		userResponse := res.SearchUserResponse{
			ID:        *user.ID,
			Name:      *user.Name,
			Email:     *user.Email, // 민감 정보 포함할지 여부에 따라 처리
			Nickname:  *user.Nickname,
			Role:      uint(user.Role),
			Image:     user.UserProfile.Image,
			CreatedAt: *user.CreatedAt,
			UpdatedAt: *user.UpdatedAt,
		}

		response = append(response, userResponse)
	}

	return response, nil
}

// TODO 유저 상태 업데이트
func (u *userUsecase) UpdateUserOnlineStatus(userId uint, online bool) error {
	return u.userRepo.UpdateCacheUser(userId, map[string]interface{}{"is_online": online})
}

// TODO 자기가 속한 회사에 사용자 리스트 가져오기(일반 사용자용)
func (u *userUsecase) GetUsersByCompany(requestUserId uint, query *req.UserQuery) ([]res.GetUserByIdResponse, error) {

	user, err := u.userRepo.GetUserByID(requestUserId)
	if err != nil {
		log.Printf("사용자 조회에 실패했습니다: %v", err)
		return nil, common.NewError(http.StatusInternalServerError, "사용자 조회에 실패했습니다")
	}

	// 회사 ID 유효성 검사
	if user.UserProfile == nil || user.UserProfile.CompanyID == nil || *user.UserProfile.CompanyID == 0 {
		log.Printf("사용자의 회사 ID가 없습니다: 사용자 ID %d", requestUserId)
		return nil, common.NewError(http.StatusBadRequest, "사용자의 회사 ID가 없습니다")
	}

	// 쿼리 기본값 설정
	if query.SortBy == "" {
		query.SortBy = req.UserSortBy(req.UserSortByID)
	}
	if query.Order == "" {
		query.Order = req.UserSortOrder(req.UserSortOrderAsc)
	}

	queryOptions := &entity.UserQueryOptions{
		SortBy: string(query.SortBy),
		Order:  string(query.Order),
	}

	// 회사 ID로 사용자 목록 조회
	users, err := u.userRepo.GetUsersByCompany(*user.UserProfile.CompanyID, queryOptions)
	if err != nil {
		log.Printf("회사 사용자 조회에 실패했습니다: %v", err)
		return nil, common.NewError(http.StatusInternalServerError, "회사 사용자 조회에 실패했습니다")
	}

	// 사용자 ID 배열 생성
	userIds := make([]uint, len(users))
	for i, user := range users {
		userIds[i] = *user.ID
	}

	// 온라인 상태 조회
	onlineStatusMap, err := u.userRepo.GetCacheUsers(userIds, []string{"is_online"})
	if err != nil {
		log.Printf("온라인 상태 조회에 실패했습니다: %v", err)
		return nil, common.NewError(http.StatusInternalServerError, "온라인 상태 조회에 실패했습니다")
	}

	// 사용자 리스트 변환
	return _utils.MapSlice(users, func(user entity.User) res.GetUserByIdResponse {
		isOnline := false
		if status, exists := onlineStatusMap[_utils.GetValueOrDefault(user.ID, 0)]["is_online"]; exists {
			if online, ok := status.(bool); ok {
				isOnline = online
			}
		}

		return res.GetUserByIdResponse{
			ID:              _utils.GetValueOrDefault(user.ID, 0),
			Email:           _utils.GetValueOrDefault(user.Email, ""),
			Name:            _utils.GetValueOrDefault(user.Name, ""),
			Nickname:        _utils.GetValueOrDefault(user.Nickname, ""),
			Phone:           _utils.GetValueOrDefault(user.Phone, ""),
			Role:            uint(_utils.GetValueOrDefault(&user.Role, entity.RoleUser)),
			IsOnline:        isOnline,
			Image:           _utils.GetValueOrDefault(user.UserProfile.Image, ""),
			Birthday:        _utils.GetValueOrDefault(&user.UserProfile.Birthday, ""),
			CompanyID:       _utils.GetValueOrDefault(user.UserProfile.CompanyID, 0),
			CompanyName:     _utils.GetFirstOrEmpty(_utils.ExtractValuesFromMapSlice[string]([]*map[string]interface{}{user.UserProfile.Company}, "name"), ""),
			DepartmentNames: _utils.ExtractValuesFromMapSlice[string](user.UserProfile.Departments, "name"),
			TeamNames:       _utils.ExtractValuesFromMapSlice[string](user.UserProfile.Teams, "name"),
			PositionId:      _utils.GetValueOrDefault(user.UserProfile.PositionId, 0),
			PositionName:    _utils.GetFirstOrEmpty(_utils.ExtractValuesFromMapSlice[string]([]*map[string]interface{}{user.UserProfile.Position}, "name"), ""),
			EntryDate:       user.UserProfile.EntryDate,
			CreatedAt:       _utils.GetValueOrDefault(user.CreatedAt, time.Time{}),
			UpdatedAt:       _utils.GetValueOrDefault(user.UpdatedAt, time.Time{}),
		}
	}), nil

}

// TODO 해당 부서에 속한 사용자 리스트 가져오기(일반 사용자용)
func (u *userUsecase) GetUsersByDepartment(departmentId uint) ([]entity.User, error) {
	users, err := u.userRepo.GetUsersByDepartment(departmentId)
	if err != nil {
		log.Printf("부서 사용자 조회에 실패했습니다: %v", err)
		return nil, common.NewError(http.StatusInternalServerError, "부서 사용자 조회에 실패했습니다")
	}
	return users, nil
}
