package main

import (
	"fmt"
	"log"
	"runtime"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	"link/config"
	handlerHttp "link/pkg/http"
	"link/pkg/interceptor"
	"link/pkg/logger"
	"link/pkg/middleware"
	ws "link/pkg/ws"
)

func startServer() {
	// 로거 초기화
	if err := logger.InitLogger(); err != nil {
		log.Fatalf("로거 초기화 실패: %v", err)
	}
	defer logger.CloseLogger()

	// panic 복구 처리
	defer func() {
		if r := recover(); r != nil {
			_, file, line, _ := runtime.Caller(2)
			logger.LogError(fmt.Sprintf("서버 실행 중 panic 발생 - 파일: %s, 라인: %d, 오류: %v", file, line, r))
			logger.LogError(fmt.Sprintf("오류가 발생한 시간: %v", time.Now().Format(time.RFC3339)))
			startServer() // 재시작
		}
	}()

	cfg := config.LoadConfig()

	config.AutoMigrate(cfg.DB)
	config.InitCompany(cfg.DB)
	config.InitAdminUser(cfg.DB)
	config.InitRedisUserState(cfg.Redis)
	// config.UpdateAllUserOffline(cfg.DB)
	config.EnsureDirectory("static/profiles")
	config.EnsureDirectory("static/posts")

	// TODO: Gin 모드 설정 (프로덕션일 경우)
	// gin.SetMode(gin.ReleaseMode)

	// dig 컨테이너 생성 및 의존성 주입
	container := config.BuildContainer(cfg.DB, cfg.Redis, cfg.Mongo, cfg.Nats)

	// Gin 라우터 설정
	r := gin.Default()

	//TODO 이미지 정적 파일 제공

	// r.Static("/static/posts", "./static/uploads/posts") 게시물
	r.Static("/static/profiles", "./static/profiles") //프로필

	// CORS 설정 - 개발 환경에서는 모든 오리진을 쿠키 허용
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:3000", "http://192.168.1.13:3000", "http://192.168.1.162:3000"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Length", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length", "Authorization", "Set-Cookie"},
		AllowCredentials: true,
	}))

	// r.Use(cors.Default()) //! 개발환경 모든 도메인 허용

	// 프록시 신뢰 설정 (프록시를 사용하지 않으면 nil 설정)
	r.SetTrustedProxies(nil)
	r.Use(interceptor.ErrorHandler())

	wsHub := ws.NewWebSocketHub()
	go wsHub.Run()

	err := container.Invoke(func(
		userHandler *handlerHttp.UserHandler,
		authHandler *handlerHttp.AuthHandler,

		companyHandler *handlerHttp.CompanyHandler,
		departmentHandler *handlerHttp.DepartmentHandler,
		chatHandler *handlerHttp.ChatHandler,
		notificationHandler *handlerHttp.NotificationHandler,
		postHandler *handlerHttp.PostHandler,

		adminHandler *handlerHttp.AdminHandler,

		imageUploadMiddleware *middleware.ImageUploadMiddleware,

		tokenInterceptor *interceptor.TokenInterceptor,

		wsHandler *ws.WsHandler,
	) {
		// WebSocket 관련 라우팅 그룹
		wsGroup := r.Group("/ws")
		{
			// 인증된 사용자만 WebSocket 사용 가능
			wsGroup.GET("/chat", wsHandler.HandleWebSocketConnection)
			wsGroup.GET("/user", wsHandler.HandleUserWebSocketConnection)
		}

		api := r.Group("/api")
		publicRoute := api.Group("/")
		{
			publicRoute.POST("user/signup", userHandler.RegisterUser)
			publicRoute.GET("user/validate-email", userHandler.ValidateEmail)
			publicRoute.GET("user/validate-nickname", userHandler.ValidateNickname)
			publicRoute.POST("auth/signin", authHandler.SignIn)
			publicRoute.GET("company/list", companyHandler.GetAllCompanies)
			publicRoute.GET("company/:id", companyHandler.GetCompanyInfo)
			publicRoute.POST("company/search", companyHandler.SearchCompany)
		}
		protectedRoute := api.Group("/", tokenInterceptor.AccessTokenInterceptor())
		//, tokenInterceptor.RefreshTokenInterceptor() accessToken 재발급 인터셉터 제거 -> accessToken 재발급 기능 따로 구현 (필요해지면 다시 사용)
		{

			auth := protectedRoute.Group("auth")
			{
				auth.POST("/signout", authHandler.SignOut)
				auth.GET("/refresh", authHandler.RefreshToken) //TODO accessToken 재발급
			}

			chat := protectedRoute.Group("chat")
			{
				//! 채팅방 관련 핸들러
				chat.GET("/list", chatHandler.GetChatRoomList)
				chat.GET("/:chatroomid", chatHandler.GetChatRoomById)
				chat.DELETE("/:chatroomid", chatHandler.LeaveChatRoom) //! 채팅방 나가기
				chat.POST("", chatHandler.CreateChatRoom)
				chat.GET("/:chatroomid/messages", chatHandler.GetChatMessages)
				chat.DELETE("/messages", chatHandler.DeleteChatMessage) //! 채팅 메시지 삭제

				// chat.GET("/:id", chatHandler.GetChatRoom) // 채팅방 정보
			}
			user := protectedRoute.Group("user")
			{
				user.GET("/:id", userHandler.GetUserInfo)
				user.PUT("/:id", imageUploadMiddleware.ProfileImageUploadMiddleware(), userHandler.UpdateUserInfo)
				user.DELETE("/:id", userHandler.DeleteUser)
				user.GET("/search", userHandler.SearchUser)
				user.GET("/company/list", userHandler.GetUserByCompany) //TODO 같은 회사 사용자 조회
				user.GET("/department/:departmentid", userHandler.GetUsersByDepartment)
			}

			company := protectedRoute.Group("company")
			{
				company.POST("/:companyId/invite", companyHandler.InviteUserToCompany)
			}
			department := protectedRoute.Group("department")
			{
				department.POST("", departmentHandler.CreateDepartment)
				department.GET("/list", departmentHandler.GetDepartments)
				department.GET("/:id", departmentHandler.GetDepartment)
				department.PUT("/:id", departmentHandler.UpdateDepartment)
				department.DELETE("/:id", departmentHandler.DeleteDepartment)
			}

			notification := protectedRoute.Group("notification")
			{
				// notification.POST("", notificationHandler.CreateNotification)
				notification.GET("/list", notificationHandler.GetNotifications)
				notification.PUT("/invite/status", notificationHandler.UpdateInviteNotificationStatus) //! 초대 알림 수락 및 거절
				// notification.PUT("/request/status", notificationHandler.UpdateRequestNotificationStatus)    //! 요청 알림 수락 및 거절
				notification.PUT("/:notificationId/read", notificationHandler.UpdateNotificationReadStatus) //! 알림 읽음 처리
			}

			post := protectedRoute.Group("post")
			{
				post.POST("", postHandler.CreatePost)
			}

			//TODO admin 요청 - 관리자 페이지
			admin := protectedRoute.Group("admin")
			{
				admin.POST("/signup", adminHandler.AdminCreateAdmin)
				admin.POST("/company", adminHandler.AdminCreateCompany)
				admin.PUT("/company", adminHandler.AdminUpdateCompany)
				admin.DELETE("/company/:companyid", adminHandler.AdminDeleteCompany)
				admin.GET("/user/list", adminHandler.AdminGetAllUsers)                     //TODO 전체 사용자 조회
				admin.GET("/user/company/:companyid", adminHandler.AdminGetUsersByCompany) //TODO 회사 사용자 조회
				admin.POST("/user/company", adminHandler.AdminAddUserToCompany)            //TODO 회사에 사용자 추가
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

func main() {
	startServer()
}
