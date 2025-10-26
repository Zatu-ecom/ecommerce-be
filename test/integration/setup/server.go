package setup

import (
	"net/http"
	"testing"

	"ecommerce-be/common/cache"
	"ecommerce-be/common/db"
	"ecommerce-be/common/middleware"
	"ecommerce-be/product"
	"ecommerce-be/user"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"gorm.io/gorm"
)

// SetupTestServer initializes the application for testing
func SetupTestServer(t *testing.T, database *gorm.DB, redisClient *redis.Client) http.Handler {
	gin.SetMode(gin.TestMode)
	router := gin.Default()

	// Set the test database as the global database instance
	// This is required because the modules use db.GetDB() to get the database instance
	if database != nil {
		db.SetDB(database)
	}

	// Set the test Redis client as the global Redis instance
	if redisClient != nil {
		cache.SetRedisClient(redisClient)
	}

	router.Use(middleware.Logger())
	router.Use(middleware.CORS())

	registerContainer(router)

	return router
}

func registerContainer(router *gin.Engine) {
	_ = user.NewContainer(router)
	_ = product.NewContainer(router)
}
