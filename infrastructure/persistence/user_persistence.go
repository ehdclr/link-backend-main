package persistence

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"reflect"
	"strconv"
	"time"

	"github.com/go-redis/redis/v8"
	"gorm.io/gorm"

	"link/infrastructure/model"

	"link/internal/user/entity"
	"link/internal/user/repository"
)

type userPersistence struct {
	db          *gorm.DB
	redisClient *redis.Client
}

// ! 생성자 함수
func NewUserPersistence(db *gorm.DB, redisClient *redis.Client) repository.UserRepository {
	return &userPersistence{db: db, redisClient: redisClient}
}

func (r *userPersistence) CreateUser(user *entity.User) error {
	// Entity -> Model 변경
	modelUser := &model.User{
		Name:     *user.Name,
		Email:    *user.Email,
		Nickname: *user.Nickname,
		Password: *user.Password,
		Phone:    *user.Phone,
		Role:     model.UserRole(user.Role),
	}

	var userOmitFields []string
	val := reflect.ValueOf(modelUser).Elem()
	typ := reflect.TypeOf(*modelUser)

	for i := 0; i < val.NumField(); i++ {
		fieldValue := val.Field(i).Interface()
		fieldName := typ.Field(i).Name
		if fieldValue == nil || fieldValue == "" || fieldValue == 0 {
			userOmitFields = append(userOmitFields, fieldName)
		}
	}

	//트랜잭션 시작
	tx := r.db.Begin()
	if tx.Error != nil {
		log.Printf("트랜잭션 시작 중 DB 오류: %v", tx.Error)
		return fmt.Errorf("트랜잭션 시작 중 DB 오류: %w", tx.Error)
	}

	// 오류 발생 시 롤백
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 사용자 생성
	if err := tx.Omit(userOmitFields...).Create(modelUser).Error; err != nil {
		log.Printf("사용자 생성중 DB 오류: %v", err)
		tx.Rollback()
		return fmt.Errorf("사용자 생성 중 DB 오류: %w", err)
	}

	// //TODO 초기 프로필은 User 테이블에서 생성한 userId 데이터만 생성 나머진 빈값

	// 초기 프로필 생성
	modelUserProfile := &model.UserProfile{
		UserID:       modelUser.ID, // 생성된 사용자 ID 사용
		CompanyID:    user.UserProfile.CompanyID,
		Image:        user.UserProfile.Image,
		Birthday:     user.UserProfile.Birthday,
		IsSubscribed: user.UserProfile.IsSubscribed,
	}

	// 프로필 정보를 Omit할 필드를 찾기 위한 로직
	var profileOmitFields []string
	valProfile := reflect.ValueOf(modelUserProfile).Elem()
	typProfile := reflect.TypeOf(*modelUserProfile)

	for i := 0; i < valProfile.NumField(); i++ {
		fieldValue := valProfile.Field(i).Interface()
		fieldName := typProfile.Field(i).Name
		if fieldValue == nil || fieldValue == "" || fieldValue == 0 {
			profileOmitFields = append(profileOmitFields, fieldName)
		}
	}

	// 프로필 생성
	if err := tx.Omit(profileOmitFields...).Create(modelUserProfile).Error; err != nil {
		tx.Rollback()
		log.Printf("사용자 프로필 생성 중 DB 오류: %v", err)
		return fmt.Errorf("사용자 프로필 생성 중 DB 오류: %w", err)
	}

	// 캐시 업데이트
	redisUserFields := make(map[string]interface{})
	redisUserFields["id"] = modelUser.ID
	redisUserFields["name"] = modelUser.Name
	redisUserFields["email"] = modelUser.Email
	redisUserFields["nickname"] = modelUser.Nickname
	redisUserFields["phone"] = modelUser.Phone
	redisUserFields["role"] = modelUser.Role

	redisUserFields["image"] = modelUserProfile.Image
	redisUserFields["company_id"] = modelUserProfile.CompanyID
	redisUserFields["birthday"] = modelUserProfile.Birthday
	redisUserFields["is_subscribed"] = modelUserProfile.IsSubscribed
	redisUserFields["is_online"] = false
	redisUserFields["entry_date"] = modelUserProfile.EntryDate
	redisUserFields["created_at"] = modelUser.CreatedAt
	redisUserFields["updated_at"] = modelUser.UpdatedAt

	if err := r.UpdateCacheUser(modelUser.ID, redisUserFields, 3*24*time.Hour); err != nil {
		log.Printf("사용자 캐시 업데이트 중 오류: %v", err)
		return fmt.Errorf("사용자 캐시 업데이트 중 오류: %w", err)
	}

	if err := tx.Commit().Error; err != nil {
		log.Printf("트랜잭션 커밋 중 DB 오류: %v", err)
		return fmt.Errorf("트랜잭션 커밋 중 DB 오류: %w", err)
	}

	return nil
}

func (r *userPersistence) ValidateEmail(email string) (*entity.User, error) {
	var user model.User

	err := r.db.Select("id", "email", "nickname", "name", "role").Where("email = ?", email).First(&user).Error
	//TODO 못찾았으면, 응답 해야함
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("사용자 조회 중 DB 오류: %w", err)
	}

	entityUser := &entity.User{
		ID:       &user.ID,
		Email:    &user.Email,
		Nickname: &user.Nickname,
		Name:     &user.Name,
		Role:     entity.UserRole(user.Role),
	}

	return entityUser, nil
}

// 닉네임 중복확인
func (r *userPersistence) ValidateNickname(nickname string) (*entity.User, error) {
	var user model.User
	err := r.db.Select("id,nickname").Where("nickname = ?", nickname).First(&user).Error

	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("사용자 조회 중 DB 오류: %w", err)
	}

	entityUser := &entity.User{
		ID:       &user.ID,
		Nickname: &user.Nickname,
	}

	return entityUser, nil
}

func (r *userPersistence) GetUserByEmail(email string) (*entity.User, error) {
	var user model.User
	// var userProfile model.UserProfile
	// err := r.db.
	// 	Table("users").
	// 	Joins("LEFT JOIN user_profiles ON user_profiles.user_id = users.id").
	// 	Select("users.id", "users.email", "users.nickname", "users.name", "users.role", "users.password", "user_profiles.company_id").
	// 	Where("users.email = ?", email).First(&user).Error
	err := r.db.Preload("UserProfile").Where("email = ?", email).First(&user).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("사용자를 찾을 수 없습니다: %s", email)
		}
		return nil, fmt.Errorf("사용자 조회 중 DB 오류: %w", err)
	}

	// UserProfile이 nil일 경우 기본값 설정

	entityUser := &entity.User{
		ID:       &user.ID,
		Email:    &user.Email,
		Nickname: &user.Nickname,
		Name:     &user.Name,
		Role:     entity.UserRole(user.Role),
		Password: &user.Password,
		UserProfile: &entity.UserProfile{
			CompanyID: user.UserProfile.CompanyID,
			Image:     user.UserProfile.Image,
		},
	}

	return entityUser, nil
}

func (r *userPersistence) GetUserByID(id uint) (*entity.User, error) {

	cacheKey := fmt.Sprintf("user:%d", id)
	userData, err := r.redisClient.HGetAll(context.Background(), cacheKey).Result()

	if err == nil && len(userData) > 0 && r.IsUserCacheComplete(userData) {

		departments := make([]*map[string]interface{}, len(userData["departments"]))
		teams := make([]*map[string]interface{}, len(userData["teams"]))

		if depsStr, ok := userData["departments"]; ok {
			json.Unmarshal([]byte(depsStr), &departments)
		}
		if teamsStr, ok := userData["teams"]; ok {
			json.Unmarshal([]byte(teamsStr), &teams)
		}

		userID, _ := strconv.ParseUint(userData["id"], 10, 64)
		role, _ := strconv.ParseUint(userData["role"], 10, 64)
		companyID, _ := strconv.ParseUint(userData["company_id"], 10, 64)
		isSubscribed, _ := strconv.ParseBool(userData["is_subscribed"])

		//TODO 온라인 상태는 레디스에서 직접가져오기
		isOnlineStr, _ := r.redisClient.HGet(context.Background(), cacheKey, "is_online").Result()

		var isOnline bool
		if isOnlineStr == "" {
			isOnline = false
		} else {
			isOnline, _ = strconv.ParseBool(userData["is_online"])
		}

		//TODO 온라인 상태가 없다면 그냥 false로 줘야함

		id := uint(userID)
		email := userData["email"]
		nickname := userData["nickname"]
		name := userData["name"]
		phone := userData["phone"]
		cid := uint(companyID)
		image := userData["image"]
		birthday := userData["birthday"]
		entryDate := userData["entry_date"]
		var parsedEntryDate time.Time
		if entryDate != "" {
			parsedEntryDate, _ = time.Parse(time.RFC3339, entryDate)
		}
		createdAt := userData["created_at"]
		updatedAt := userData["updated_at"]
		var parsedCreatedAt time.Time
		var parsedUpdatedAt time.Time
		if createdAt != "" {
			parsedCreatedAt, err = time.Parse("2006-01-02 15:04:05.999999 -0700 MST", createdAt)
			if err != nil {
				log.Printf("시간 파싱 오류 (created_at): %v, 값: %s", err, createdAt)
			}
		}
		if updatedAt != "" {
			parsedUpdatedAt, err = time.Parse("2006-01-02 15:04:05.999999 -0700 MST", updatedAt)
			if err != nil {
				log.Printf("시간 파싱 오류 (updated_at): %v, 값: %s", err, updatedAt)
			}
		}

		return &entity.User{
			ID:       &id,
			Email:    &email,
			Nickname: &nickname,
			Name:     &name,
			Phone:    &phone,
			Role:     entity.UserRole(role),
			IsOnline: &isOnline,
			UserProfile: &entity.UserProfile{
				Image:        &image,
				Birthday:     birthday,
				IsSubscribed: isSubscribed,
				CompanyID:    &cid,
				Departments:  departments,
				Teams:        teams,
				EntryDate:    &parsedEntryDate,
			},
			CreatedAt: &parsedCreatedAt,
			UpdatedAt: &parsedUpdatedAt,
		}, nil

	}

	//TODO UserProfile 조인 추가
	var user model.User
	err = r.db.
		Preload("UserProfile.Departments").
		Preload("UserProfile.Teams").
		Preload("UserProfile.Position").
		Where("id = ?", id).First(&user).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("사용자를 찾을 수 없습니다: %d", id)
		}
		return nil, fmt.Errorf("사용자 조회 중 DB 오류: %w", err)
	}

	departments := make([]*map[string]interface{}, len(user.UserProfile.Departments))
	for i, dept := range user.UserProfile.Departments {
		departments[i] = &map[string]interface{}{
			"id":   dept.ID,
			"name": dept.Name,
		}
	}

	teams := make([]*map[string]interface{}, len(user.UserProfile.Teams))
	for i, team := range user.UserProfile.Teams {
		teams[i] = &map[string]interface{}{
			"id":   team.ID,
			"name": team.Name,
		}
	}

	isOnline := false
	if onlineStr, err := r.redisClient.HGet(context.Background(), cacheKey, "is_online").Result(); err == nil {
		isOnline, _ = strconv.ParseBool(onlineStr)
	}

	entityUser := &entity.User{
		ID:       &user.ID,
		Email:    &user.Email,
		Nickname: &user.Nickname,
		Name:     &user.Name,
		Phone:    &user.Phone,
		Role:     entity.UserRole(user.Role),
		IsOnline: &isOnline,
		UserProfile: &entity.UserProfile{
			Image:        user.UserProfile.Image,
			Birthday:     user.UserProfile.Birthday,
			IsSubscribed: user.UserProfile.IsSubscribed,
			CompanyID:    user.UserProfile.CompanyID,
			Departments:  departments,
			Teams:        teams,
			PositionId:   user.UserProfile.PositionID,
			EntryDate:    &user.UserProfile.EntryDate,
			// Position:     user.UserProfile.Position,
		},
		CreatedAt: &user.CreatedAt,
		UpdatedAt: &user.UpdatedAt,
	}

	//TODO 캐시 비동기 업데이트
	go func() {
		// departments와 teams를 JSON으로 변환
		cacheData := map[string]interface{}{
			"id":       *entityUser.ID,
			"email":    *entityUser.Email,
			"nickname": *entityUser.Nickname,
			"name":     *entityUser.Name,
			"role":     entityUser.Role,
		}

		// Optional fields
		if entityUser.Phone != nil {
			cacheData["phone"] = *entityUser.Phone
		}
		if entityUser.UserProfile != nil {
			if entityUser.UserProfile.Image != nil {
				cacheData["image"] = *entityUser.UserProfile.Image
			}
			if entityUser.UserProfile.Birthday != "" {
				cacheData["birthday"] = entityUser.UserProfile.Birthday
			}
			cacheData["is_subscribed"] = entityUser.UserProfile.IsSubscribed
			if entityUser.UserProfile.CompanyID != nil {
				cacheData["company_id"] = *entityUser.UserProfile.CompanyID
			}
			if len(departments) > 0 {
				if depsJSON, err := json.Marshal(departments); err == nil {
					cacheData["departments"] = string(depsJSON)
				}
			}
			if len(teams) > 0 {
				if teamsJSON, err := json.Marshal(teams); err == nil {
					cacheData["teams"] = string(teamsJSON)
				}
			}
			if entityUser.UserProfile.EntryDate != nil {
				cacheData["entry_date"] = *entityUser.UserProfile.EntryDate
			}
		}
		if entityUser.CreatedAt != nil {
			cacheData["created_at"] = entityUser.CreatedAt
		}
		if entityUser.UpdatedAt != nil {
			cacheData["updated_at"] = entityUser.UpdatedAt
		}

		if err := r.UpdateCacheUser(id, cacheData, 3*24*time.Hour); err != nil {
			log.Printf("Redis 캐시 업데이트 실패: %v", err)
		}
	}()

	return entityUser, nil
}

func (r *userPersistence) GetUserByIds(ids []uint) ([]entity.User, error) {
	var users []model.User

	// ids 슬라이스가 비어있는지 확인
	if len(ids) == 0 {
		return nil, fmt.Errorf("유효하지 않은 사용자 ID 목록")
	}

	// 관련 데이터를 Preload하여 로드
	if err := r.db.Preload("UserProfile.Company").
		Preload("UserProfile.Departments").
		Preload("UserProfile.Teams").
		Preload("UserProfile.Position").
		Where("id IN ?", ids).
		Find(&users).Error; err != nil {
		return nil, fmt.Errorf("사용자 조회 중 DB 오류: %w", err)
	}

	// Entity 변환
	entityUsers := make([]entity.User, len(users))
	for i, user := range users {

		// Departments 변환
		var departmentMaps []*map[string]interface{}
		for _, dept := range user.UserProfile.Departments {
			deptMap := map[string]interface{}{
				"id":   dept.ID,
				"name": dept.Name,
			}
			departmentMaps = append(departmentMaps, &deptMap)
		}

		// Teams 변환
		var teamMaps []*map[string]interface{}
		for _, team := range user.UserProfile.Teams {
			teamMap := map[string]interface{}{
				"id":   team.ID,
				"name": team.Name,
			}
			teamMaps = append(teamMaps, &teamMap)
		}

		// Position 변환
		var positionMap *map[string]interface{}
		if user.UserProfile.Position != nil {
			posMap := map[string]interface{}{
				"id":   user.UserProfile.Position.ID,
				"name": user.UserProfile.Position.Name,
			}
			positionMap = &posMap
		}

		// Entity User 변환
		entityUsers[i] = entity.User{
			ID:       &user.ID,
			Email:    &user.Email,
			Nickname: &user.Nickname,
			Name:     &user.Name,
			Role:     entity.UserRole(user.Role),
			UserProfile: &entity.UserProfile{
				UserId:       user.ID,
				CompanyID:    user.UserProfile.CompanyID,
				Departments:  departmentMaps,
				Teams:        teamMaps,
				Image:        user.UserProfile.Image,
				Birthday:     user.UserProfile.Birthday,
				IsSubscribed: user.UserProfile.IsSubscribed,
				PositionId:   user.UserProfile.PositionID,
				Position:     positionMap,
				CreatedAt:    user.UserProfile.CreatedAt,
				UpdatedAt:    user.UserProfile.UpdatedAt,
			},
		}
	}

	return entityUsers, nil
}

func (r *userPersistence) UpdateUser(id uint, updates map[string]interface{}, profileUpdates map[string]interface{}) error {

	tx := r.db.Begin()
	if tx.Error != nil {
		return fmt.Errorf("트랜잭션 시작 중 DB 오류: %w", tx.Error)
	}

	if len(updates) > 0 {
		if err := tx.Model(&model.User{}).Where("id = ?", id).Updates(updates).Error; err != nil {
			tx.Rollback()
			return fmt.Errorf("사용자 업데이트 중 DB 오류: %w", err)
		}
	}
	//TODO 캐시 업데이트 - 이미지 프로필 업데이트 - hash-set

	// UserProfile 업데이트
	if len(profileUpdates) > 0 {
		if err := tx.Model(&model.UserProfile{}).Where("user_id = ?", id).Updates(profileUpdates).Error; err != nil {
			tx.Rollback()
			return fmt.Errorf("사용자 프로필 업데이트 중 DB 오류: %w", err)
		}
	}

	//TODO 캐시 업데이트 - updates와 profileUpdates 둘 다 업데이트
	// Redis 캐시 비동기 업데이트
	go func() {
		// 모든 업데이트를 하나의 맵으로 병합
		cacheUpdates := make(map[string]interface{})
		for k, v := range updates {
			cacheUpdates[k] = v
		}
		for k, v := range profileUpdates {
			cacheUpdates[k] = v
		}

		if err := r.UpdateCacheUser(id, cacheUpdates, 3*24*time.Hour); err != nil {
			log.Printf("Redis 캐시 업데이트 실패: %v", err)
		}
	}()

	//TODO 캐시 업데이트 - profileUpdates 업데이트

	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("트랜잭션 커밋 중 DB 오류: %w", err)
	}

	return nil
}

// TODO CasCade 되는지 확인
func (r *userPersistence) DeleteUser(id uint) error {
	if err := r.db.Delete(&entity.User{}, id).Error; err != nil {
		return fmt.Errorf("사용자 삭제 중 DB 오류: %w", err)
	}
	return nil
}

func (r *userPersistence) SearchUser(user *entity.User) ([]entity.User, error) {
	var users []model.User

	// 기본 쿼리: 관리자를 제외함 (role != 1)
	query := r.db.Where("role != ? AND role != ?", 1, 2)
	// 이메일이 입력된 경우 이메일로 검색 조건 추가
	if user.Email != nil {
		query = query.Where("email LIKE ?", "%"+*user.Email+"%")
	}

	// 이름이 입력된 경우 이름으로 검색 조건 추가
	if user.Name != nil {
		query = query.Where("name LIKE ?", "%"+*user.Name+"%")
	}

	if user.Nickname != nil {
		query = query.Where("nickname LIKE ?", "%"+*user.Nickname+"%")
	}

	//!TODO 입력된 것 토대로 조건

	// 최종 쿼리 실행
	if err := query.Preload("UserProfile").Find(&users).Error; err != nil {
		return nil, fmt.Errorf("사용자 검색 중 DB 오류: %w", err)
	}

	entityUsers := make([]entity.User, len(users))
	for i, user := range users {
		entityUsers[i] = entity.User{
			ID:       &user.ID,
			Email:    &user.Email,
			Nickname: &user.Nickname,
			Name:     &user.Name,
			Role:     entity.UserRole(user.Role),
			UserProfile: &entity.UserProfile{
				Image:        user.UserProfile.Image,
				Birthday:     user.UserProfile.Birthday,
				IsSubscribed: user.UserProfile.IsSubscribed,
				CompanyID:    user.UserProfile.CompanyID,
			},
			CreatedAt: &user.CreatedAt,
			UpdatedAt: &user.UpdatedAt,
		}
	}

	return entityUsers, nil
}

//! 회사

// TODO 회사 사용자 조회 (일반 사용자, 회사 관리자 포함)
func (r *userPersistence) GetUsersByCompany(companyId uint, queryOptions *entity.UserQueryOptions) ([]entity.User, error) {
	var users []entity.User

	// UserProfile의 company_id 필드를 사용하여 조건을 설정
	dbQuery := r.db.
		Table("users").
		Select("users.id", "users.name", "users.email", "users.nickname", "users.role", "users.phone", "users.created_at", "users.updated_at",
			"user_profiles.birthday", "user_profiles.is_subscribed", "user_profiles.entry_date", "user_profiles.image",
			"companies.id as company_id", "companies.cp_name as company_name",
			"departments.id as department_id", "departments.name as department_name",
			"teams.id as team_id", "teams.name as team_name",
			"positions.id as position_id", "positions.name as position_name").
		Joins("JOIN user_profiles ON user_profiles.user_id = users.id").
		Joins("JOIN companies ON companies.id = user_profiles.company_id").
		Joins("LEFT JOIN user_profile_departments ON user_profile_departments.user_profile_user_id = users.id").
		Joins("LEFT JOIN departments ON departments.id = user_profile_departments.department_id").
		Joins("LEFT JOIN user_profile_teams ON user_profile_teams.user_profile_user_id = users.id").
		Joins("LEFT JOIN teams ON teams.id = user_profile_teams.team_id").
		Joins("LEFT JOIN positions ON positions.id = user_profiles.position_id").
		Where("user_profiles.company_id = ? OR users.role = ? OR users.role = ?", companyId, entity.RoleCompanyManager, entity.RoleUser)

	sortBy := "users.id"
	if queryOptions.SortBy != "" {
		sortBy = queryOptions.SortBy
	}

	order := "asc"
	if queryOptions.Order == "desc" {
		order = "desc"
	}

	dbQuery = dbQuery.Order(sortBy + " " + order)

	// 쿼리 실행
	rows, err := dbQuery.Rows()
	if err != nil {
		return nil, fmt.Errorf("회사 사용자 조회 중 DB 오류: %w", err)
	}
	defer rows.Close()

	// 사용자 ID를 키로 하는 맵을 사용하여 중복 사용자 데이터를 누적
	userMap := make(map[uint]*entity.User)

	for rows.Next() {
		var userID uint
		var user entity.User
		var userProfile entity.UserProfile
		var companyName, departmentName, teamName, positionName *string
		var companyID, departmentID, teamID, positionID *uint
		var (
			birthday     sql.NullString
			entryDate    sql.NullTime
			isSubscribed bool
			image        sql.NullString
		)

		// 데이터베이스에서 조회된 데이터를 변수에 스캔
		if err := rows.Scan(
			&userID, &user.Name, &user.Email, &user.Nickname, &user.Role, &user.Phone, &user.CreatedAt, &user.UpdatedAt,
			&birthday, &isSubscribed, &entryDate, &image,
			&companyID, &companyName, &departmentID, &departmentName, &teamID, &teamName, &positionID, &positionName,
		); err != nil {
			return nil, fmt.Errorf("조회 결과 스캔 중 오류: %w", err)
		}

		// 기존 사용자 데이터를 찾거나 새로 생성
		existingUser, found := userMap[userID]
		if found {
			user = *existingUser
		} else {
			user.ID = &userID
			if birthday.Valid {
				userProfile.Birthday = birthday.String
			}
			if entryDate.Valid {
				userProfile.EntryDate = &entryDate.Time
			}
			userProfile.IsSubscribed = isSubscribed
			if image.Valid {
				userProfile.Image = &image.String
			}
			userProfile.CompanyID = companyID

			if companyName != nil {
				companyMap := map[string]interface{}{
					"name": *companyName,
				}
				userProfile.Company = &companyMap
			}

			// 직책 추가
			if positionID != nil && positionName != nil {
				position := map[string]interface{}{
					"name": *positionName,
				}
				userProfile.Position = &position
			}

			user.UserProfile = &userProfile
			userMap[userID] = &user
		}

		// 부서가 이미 추가되지 않았다면 추가
		if departmentID != nil && departmentName != nil {
			departmentExists := false
			for _, dept := range user.UserProfile.Departments {
				if dept != nil && (*dept)["name"] == *departmentName {
					departmentExists = true
					break
				}
			}
			if !departmentExists {
				department := map[string]interface{}{
					"name": *departmentName,
				}
				user.UserProfile.Departments = append(user.UserProfile.Departments, &department)
			}
		}

		// 팀이 이미 추가되지 않았다면 추가
		if teamID != nil && teamName != nil {
			teamExists := false
			for _, team := range user.UserProfile.Teams {
				if team != nil && (*team)["name"] == *teamName {
					teamExists = true
					break
				}
			}
			if !teamExists {
				team := map[string]interface{}{
					"name": *teamName,
				}
				user.UserProfile.Teams = append(user.UserProfile.Teams, &team)
			}
		}
	}

	// 최종 사용자 목록 생성
	for _, user := range userMap {
		users = append(users, *user)
	}

	return users, nil
}

// ! 부서
func (r *userPersistence) CreateUserDepartment(userId uint, departmentId uint) error {
	tx := r.db.Begin()
	if tx.Error != nil {
		return fmt.Errorf("트랜잭션 시작 중 오류: %w", tx.Error)
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 사용자 프로필 조회
	var userProfile model.UserProfile
	if err := tx.Where("user_id = ?", userId).First(&userProfile).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("사용자 프로필 조회 중 오류: %w", err)
	}

	// 부서 조회
	var department model.Department
	if err := tx.First(&department, departmentId).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("부서 조회 중 오류: %w", err)
	}

	// Association 추가
	if err := tx.Model(&userProfile).Association("Departments").Append(&department); err != nil {
		tx.Rollback()
		return fmt.Errorf("부서 할당 중 오류: %w", err)
	}

	return tx.Commit().Error
}

func (r *userPersistence) GetUsersByDepartment(departmentId uint) ([]entity.User, error) {
	var users []entity.User

	if err := r.db.Where("department_id = ?", departmentId).Find(&users).Error; err != nil {
		return nil, fmt.Errorf("부서 사용자 조회 중 DB 오류: %w", err)
	}

	return users, nil
}

// ! 팀
func (r *userPersistence) CreateUserTeam(userId uint, teamId uint) error {
	tx := r.db.Begin()
	if tx.Error != nil {
		return fmt.Errorf("트랜잭션 시작 중 오류: %w", tx.Error)
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	var userProfile model.UserProfile
	if err := tx.Where("user_id = ?", userId).First(&userProfile).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("사용자 프로필 조회 중 오류: %w", err)
	}

	var team model.Team
	if err := tx.First(&team, teamId).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("팀 조회 중 오류: %w", err)
	}

	if err := tx.Model(&userProfile).Association("Teams").Append(&team); err != nil {
		tx.Rollback()
		return fmt.Errorf("팀 할당 중 오류: %w", err)
	}

	return tx.Commit().Error
}

// !---------------------------------------------- 관리자 관련

func (r *userPersistence) GetAllUsers(requestUserId uint) ([]entity.User, error) {
	var users []model.User
	var requestUser model.User

	// 먼저 요청한 사용자의 정보를 가져옴
	if err := r.db.First(&requestUser, requestUserId).Error; err != nil {
		log.Printf("요청한 사용자 조회 중 DB 오류: %v", err)
		return nil, err
	}

	// 관리자는 자기보다 권한이 낮은 사용자 리스트들을 가져옴
	query := r.db.Preload("UserProfile").
		Preload("UserProfile.Company").
		Preload("UserProfile.Departments").
		Preload("UserProfile.Teams").
		Preload("UserProfile.Position")

	if requestUser.Role == model.UserRole(entity.RoleAdmin) {
		if err := query.Find(&users).Error; err != nil {
			log.Printf("사용자 조회 중 DB 오류: %v", err)
			return nil, err
		}
	} else if requestUser.Role == model.UserRole(entity.RoleSubAdmin) {
		if err := query.Where("role >= ?", model.UserRole(entity.RoleSubAdmin)).Find(&users).Error; err != nil {
			log.Printf("사용자 조회 중 DB 오류: %v", err)
			return nil, err
		}
	} else {
		log.Printf("잘못된 사용자 권한: %d", requestUser.Role)
		return nil, fmt.Errorf("잘못된 사용자 권한")
	}

	// model.User -> entity.User 변환
	entityUsers := make([]entity.User, len(users))
	for i, user := range users {

		// 각 사용자별로 회사 정보 매핑
		var companyMap *map[string]interface{}
		if user.UserProfile.Company != nil {
			companyData := map[string]interface{}{
				"id":   user.UserProfile.Company.ID,
				"name": user.UserProfile.Company.CpName,
			}
			companyMap = &companyData
		}

		var departmentMaps []*map[string]interface{}
		if user.UserProfile.Departments != nil {
			for _, department := range user.UserProfile.Departments {
				departmentMap := map[string]interface{}{
					"id":   department.ID,
					"name": department.Name,
				}
				departmentMaps = append(departmentMaps, &departmentMap)
			}
		}

		var teamMaps []*map[string]interface{}
		if user.UserProfile.Teams != nil {
			for _, team := range user.UserProfile.Teams {
				teamMap := map[string]interface{}{
					"id":   team.ID,
					"name": team.Name,
				}
				teamMaps = append(teamMaps, &teamMap)
			}
		}

		// Position mapping
		var positionMap *map[string]interface{}
		if user.UserProfile.Position != nil {
			posData := map[string]interface{}{
				"id":   user.UserProfile.Position.ID,
				"name": user.UserProfile.Position.Name,
			}
			positionMap = &posData
		}

		entityUsers[i] = entity.User{
			ID:       &user.ID,
			Email:    &user.Email,
			Name:     &user.Name,
			Nickname: &user.Nickname,
			Phone:    &user.Phone,
			Role:     entity.UserRole(user.Role),
			UserProfile: &entity.UserProfile{
				Image:        user.UserProfile.Image,
				Birthday:     user.UserProfile.Birthday,
				IsSubscribed: user.UserProfile.IsSubscribed,
				CompanyID:    user.UserProfile.CompanyID,
				Company:      companyMap,
				Departments:  departmentMaps,
				Teams:        teamMaps,
				PositionId:   user.UserProfile.PositionID,
				Position:     positionMap,
			},
			CreatedAt: &user.CreatedAt,
			UpdatedAt: &user.UpdatedAt,
		}
	}

	return entityUsers, nil
}

// !--------------------------- ! redis 캐시 관련
func (r *userPersistence) UpdateCacheUser(userId uint, fields map[string]interface{}, ttl time.Duration) error {
	cacheKey := fmt.Sprintf("user:%d", userId)
	redisFields := make(map[string]interface{})
	for key, value := range fields {
		switch v := value.(type) {
		case nil:
			redisFields[key] = ""
		case string:
			redisFields[key] = v
		case bool:
			redisFields[key] = strconv.FormatBool(v)
		case int:
			redisFields[key] = strconv.Itoa(v)
		case uint:
			redisFields[key] = strconv.FormatUint(uint64(v), 10)
		default:
			redisFields[key] = fmt.Sprintf("%v", v)
		}
	}
	if len(redisFields) == 0 {
		return nil
	}

	// HMSet 명령어로 여러 필드를 한 번에 업데이트
	if err := r.redisClient.HMSet(context.Background(), cacheKey, redisFields).Err(); err != nil {
		return fmt.Errorf("redis 사용자 캐시 업데이트 중 오류: %w", err)
	}
	return nil
}

// Redis에서 사용자 캐시를 조회하는 함수
func (r *userPersistence) GetCacheUser(userId uint, fields []string) (*entity.User, error) {
	cacheKey := fmt.Sprintf("user:%d", userId)

	// Redis에서 지정된 필드의 데이터 조회
	values, err := r.redisClient.HMGet(context.Background(), cacheKey, fields...).Result()
	if err != nil {
		return nil, fmt.Errorf("redis에서 사용자 조회 중 오류: %w", err)
	}

	// 데이터가 없으면 nil 반환
	if len(values) == 0 {
		return nil, nil
	}
	// 해시셋 데이터를 User 구조체로 매핑
	user := &entity.User{
		ID: &userId,
	}

	// 필드 값 매핑
	for i, field := range fields {
		if values[i] == nil {
			continue
		}

		switch field {
		case "is_online":
			isOnline := values[i].(string) == "true"
			user.IsOnline = &isOnline
		case "image":
			image := values[i].(string)
			user.UserProfile.Image = &image
		case "birthday":
			user.UserProfile.Birthday = values[i].(string)
		}
	}

	fmt.Println("redis 테스트", user)
	fmt.Println("user", user)

	return user, nil
}

// TODO 여러명의 캐시 내용 가져오기 - TTL 설정
func (r *userPersistence) GetCacheUsers(userIds []uint, fields []string) (map[uint]map[string]interface{}, error) {
	userCacheMap := make(map[uint]map[string]interface{})

	if len(userIds) == 0 {
		return userCacheMap, nil
	}

	for _, userId := range userIds {
		cacheUserKey := fmt.Sprintf("user:%d", userId)
		values, err := r.redisClient.HMGet(context.Background(), cacheUserKey, fields...).Result()
		if err != nil {
			return nil, fmt.Errorf("redis에서 사용자 %d의 데이터를 조회하는 중 오류 발생: %w", userId, err)
		}

		if len(values) == 0 {
			continue
		}

		fieldMap := make(map[string]interface{})
		for i, field := range fields {
			if values[i] != nil {
				fieldMap[field] = values[i]
			}
		}
		userCacheMap[userId] = fieldMap
	}

	return userCacheMap, nil
}

// ! 캐시 데이터가 완전한지 확인하는 헬퍼 함수
func (r *userPersistence) IsUserCacheComplete(userData map[string]string) bool {
	requiredFields := []string{
		"id", "name", "email", "nickname", "role",
		"image", "company_id", "departments", "teams",
		"birthday", "is_subscribed",
		"created_at", "updated_at", "entry_date",
	}

	for _, field := range requiredFields {
		if _, exists := userData[field]; !exists {
			return false
		}
	}
	return true
}
