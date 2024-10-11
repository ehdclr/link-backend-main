package config

import (

	// auth 패키지 추가 (필요한 경우)

	"link/infrastructure/persistence"
	"link/pkg/http"
	"link/pkg/interceptor"
	"link/pkg/ws"

	// 새로 추가

	authUsecase "link/internal/auth/usecase"
	chatUsecase "link/internal/chat/usecase"
	departmentUsecase "link/internal/department/usecase"
	userUsecase "link/internal/user/usecase"

	"github.com/go-redis/redis/v8"
	"go.uber.org/dig"
	"gorm.io/gorm"
)

func BuildContainer(db *gorm.DB, redisClient *redis.Client) *dig.Container {
	container := dig.New()

	// DB 및 Redis 클라이언트 등록
	container.Provide(func() *gorm.DB { return db })
	container.Provide(func() *redis.Client { return redisClient })

	//ws 주입
	container.Provide(ws.NewWebSocketHub)
	container.Provide(ws.NewWsHandler)

	//인터셉터 주입
	container.Provide(interceptor.NewTokenInterceptor)

	// Repository 계층 등록
	container.Provide(persistence.NewAuthPersistenceRedis)
	container.Provide(persistence.NewUserPersistencePostgres)
	container.Provide(persistence.NewDepartmentPersistencePostgres)
	container.Provide(persistence.NewChatPersistencePostgres)

	// Usecase 계층 등록
	container.Provide(authUsecase.NewAuthUsecase)
	container.Provide(userUsecase.NewUserUsecase)
	container.Provide(departmentUsecase.NewDepartmentUsecase)
	container.Provide(chatUsecase.NewChatUsecase)
	// Handler 계층 등록
	container.Provide(http.NewUserHandler)
	container.Provide(http.NewAuthHandler)
	container.Provide(http.NewDepartmentHandler)
	container.Provide(http.NewChatHandler)

	return container
}