package repository

import (
	"link/internal/user/entity"
	"time"
)

// TODO 추상화
type UserRepository interface {
	CreateUser(user *entity.User) error
	ValidateEmail(email string) (*entity.User, error)
	ValidateNickname(nickname string) (*entity.User, error)

	GetUserByEmail(email string) (*entity.User, error)
	GetAllUsers(requestUserId uint) ([]entity.User, error)
	GetUserByID(id uint) (*entity.User, error)
	GetUserByIds(ids []uint) ([]entity.User, error)
	UpdateUser(id uint, updates map[string]interface{}, profileUpdates map[string]interface{}) error
	DeleteUser(id uint) error
	SearchUser(companyId uint, searchTerm string) ([]entity.User, error)

	GetUsersByCompany(companyId uint, query *entity.UserQueryOptions) ([]entity.User, error)
	GetUsersIdsByCompany(companyId uint) ([]uint, error)
	UpdateUserDepartments(userId uint, departmentIds []uint) error
	// GetOrganizationByCompany(companyId uint) ([]entity.User, error)

	//관리자 관련
	AdminSearchUser(searchTerm string) ([]entity.User, error)
	//TODO 부서
	CreateUserDepartment(userId uint, departmentId uint) error
	GetUsersByDepartment(departmentId uint) ([]entity.User, error)

	// GetUsersByTeam(teamId uint) ([]entity.User, error)

	//TODO redis 캐시 관련
	UpdateCacheUser(userId uint, fields map[string]interface{}, ttl time.Duration) error
	GetCacheUser(userId uint, fields []string) (*entity.User, error)
	GetCacheUsers(userIds []uint, fields []string) (map[uint]map[string]interface{}, error)
	IsUserCacheComplete(userData map[string]string) bool
}
