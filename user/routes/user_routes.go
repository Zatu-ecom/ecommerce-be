package routes

import (
	"ecommerce-be/common/middleware"
	"ecommerce-be/user/factory/singleton"
	"ecommerce-be/user/handler"

	"github.com/gin-gonic/gin"
)

// UserModule implements the Module interface for user routes
type UserModule struct {
	userHandler      *handler.UserHandler
	userQueryHandler *handler.UserQueryHandler
}

// NewUserModule creates a new instance of UserModule
func NewUserModule() *UserModule {
	f := singleton.GetInstance()
	return &UserModule{
		userHandler:      f.GetUserHandler(),
		userQueryHandler: f.GetUserQueryHandler(),
	}
}

// TODO: We have to reate different routes for Seller or can be use existing regester route
// but in that case we have to add seller details in the register request

// RegisterRoutes registers all user-related routes
func (m *UserModule) RegisterRoutes(router *gin.Engine) {
	// Auth middleware for protected routes
	auth := middleware.CustomerAuth()
	sellerAuth := middleware.SellerAuth()

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

		// User query routes (seller or admin only)
		// Sellers can only see users in their seller scope
		// Admins can see all users
		userRoutes.GET("", sellerAuth, m.userQueryHandler.ListUsers)
	}
}
