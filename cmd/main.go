package main

import (
	"log"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	"link/config"
	handlerHttp "link/pkg/http"
	"link/pkg/interceptor"
	"link/pkg/ws"
)

func main() {

	cfg := config.LoadConfig()

	config.InitAdminUser(cfg.DB)
	config.AutoMigrate(cfg.DB)

	// TODO: Gin 모드 설정 (프로덕션일 경우)
	// gin.SetMode(gin.ReleaseMode)

	// dig 컨테이너 생성 및 의존성 주입
	container := config.BuildContainer(cfg.DB, cfg.Redis)

	// Gin 라우터 설정
	r := gin.Default()

	// CORS 설정
	// r.Use(cors.New(cors.Config{
	// 	AllowOrigins:     []string{"http://192.168.1.13:3000"}, // 허용할 도메인
	// 	AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
	// 	AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
	// 	ExposeHeaders:    []string{"Content-Length"},
	// 	AllowCredentials: true,
	// }))
	r.Use(cors.Default()) //! 개발환경 모든 도메인 허용

	// 프록시 신뢰 설정 (프록시를 사용하지 않으면 nil 설정)
	r.SetTrustedProxies(nil)
	// 글로벌 에러 처리 미들웨어 적용
	r.Use(interceptor.ErrorHandler())

	wsHub := ws.NewWebSocketHub()
	go wsHub.Run()

	err := container.Invoke(func(
		userHandler *handlerHttp.UserHandler,
		authHandler *handlerHttp.AuthHandler,
		departmentHandler *handlerHttp.DepartmentHandler,
		chatHandler *handlerHttp.ChatHandler,
		tokenInterceptor *interceptor.TokenInterceptor,
		wsHandler *ws.WsHandler,
	) {

		api := r.Group("/api")
		publicRoute := api.Group("/")
		{
			publicRoute.POST("user/signup", userHandler.RegisterUser)
			publicRoute.GET("user/validate-email", userHandler.ValidateEmail)
			publicRoute.POST("auth/signin", authHandler.SignIn)

			// WebSocket 핸들러 추가
			publicRoute.GET("/ws", wsHandler.HandleWebSocket)

		}
		protectedRoute := api.Group("/", tokenInterceptor.AccessTokenInterceptor(), tokenInterceptor.RefreshTokenInterceptor())
		{
			protectedRoute.POST("auth/signout", authHandler.SignOut)

			user := protectedRoute.Group("user")
			{
				user.GET("/", userHandler.GetAllUsers)
				user.GET("/:id", userHandler.GetUserInfo)
				user.PUT("/:id", userHandler.UpdateUserInfo)
				user.DELETE("/:id", userHandler.DeleteUser)
				user.GET("/search", userHandler.SearchUser)
				// user.GET("/department/:departmentId", userHandler.GetUsersByDepartment)
			}
			department := protectedRoute.Group("department")
			{
				department.POST("/", departmentHandler.CreateDepartment)
				department.GET("/", departmentHandler.GetDepartments)
				department.GET("/:id", departmentHandler.GetDepartment)
				department.PUT("/:id", departmentHandler.UpdateDepartment)
				department.DELETE("/:id", departmentHandler.DeleteDepartment)
			}

			chat := protectedRoute.Group("chat")
			{
				chat.POST("/", chatHandler.CreateChatRoom)
			}
		}
	})
	if err != nil {
		log.Fatal("의존성 주입에 실패했습니다: ", err)
	}

	// HTTP 서버 시작
	log.Printf("HTTP 서버 실행중: %s", cfg.HTTPPort)
	if err := r.Run(cfg.HTTPPort); err != nil {
		log.Fatalf("HTTP 서버 시작 실패: %v", err)
	}

}
