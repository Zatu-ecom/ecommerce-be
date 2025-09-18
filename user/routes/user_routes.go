package routes

import (
	"ecommerce-be/common/db"
	"ecommerce-be/common/middleware"
	"ecommerce-be/user/handlers"
	"ecommerce-be/user/repositories"
	"ecommerce-be/user/service"

	"github.com/gin-gonic/gin"
)

// UserModule implements the Module interface for user routes
type UserModule struct {
	userHandler *handlers.UserHandler
}

// NewUserModule creates a new instance of UserModule
func NewUserModule() *UserModule {
	addressRepo := repositories.NewAddressRepository(db.GetDB())
	addressService := service.NewAddressService(addressRepo)

	userRepo := repositories.NewUserRepository(db.GetDB())
	userService := service.NewUserService(userRepo, addressService)

	return &UserModule{
		userHandler: handlers.NewUserHandler(userService),
	}
}

// TODO: We have to reate different routes for Seller or can be use existing regester route
// but in that case we have to add seller details in the register request

// RegisterRoutes registers all user-related routes
func (m *UserModule) RegisterRoutes(router *gin.Engine) {
	// Auth middleware for protected routes
	auth := middleware.CustomerAuth()

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
