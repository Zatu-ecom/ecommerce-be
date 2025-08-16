package routes

import (
	"datun.com/be/common"
	"datun.com/be/common/middleware"
	"datun.com/be/user/handlers"
	"datun.com/be/user/repositories"
	"datun.com/be/user/service"
	"github.com/gin-gonic/gin"
)

// UserModule implements the Module interface for user routes
type UserModule struct {
	userHandler *handlers.UserHandler
}

// NewUserModule creates a new instance of UserModule
func NewUserModule() *UserModule {
	userRepo := repositories.NewUserRepository(common.GetDB())
	userService := service.NewUserService(userRepo)

	return &UserModule{
		userHandler: handlers.NewUserHandler(userService),
	}
}

// RegisterRoutes registers all user-related routes
func (m *UserModule) RegisterRoutes(router *gin.Engine) {
	// Auth middleware for protected routes
	auth := middleware.Auth()

	// Authentication routes
	authRoutes := router.Group("/api/auth")
	{
		authRoutes.POST("/register", m.userHandler.Register)
		authRoutes.POST("/login", m.userHandler.Login)
		authRoutes.POST("/refresh", auth, m.userHandler.RefreshToken)
		authRoutes.POST("/logout", auth, m.userHandler.Logout)
	}

	// User routes
	userRoutes := router.Group("/api/users")
	{
		// User profile routes (protected)
		userRoutes.GET("/profile", auth, m.userHandler.GetProfile)
		userRoutes.PUT("/profile", auth, m.userHandler.UpdateProfile)
		userRoutes.PATCH("/password", auth, m.userHandler.ChangePassword)
	}
}
