package setup

import (
	"net/http"
	"os"
	"testing"

	"ecommerce-be/common/cache"
	"ecommerce-be/common/config"
	"ecommerce-be/common/db"
	"ecommerce-be/common/log"
	"ecommerce-be/common/middleware"
	"ecommerce-be/inventory"
	"ecommerce-be/notification"
	"ecommerce-be/order"
	"ecommerce-be/payment"
	"ecommerce-be/product"
	"ecommerce-be/promotion"
	"ecommerce-be/user"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"gorm.io/gorm"
)

// setTestEnvVars sets required environment variables for test configuration
func setTestEnvVars() {
	// Only set if not already set (allows override from .env.test or CI)
	envDefaults := map[string]string{
		"DB_HOST":        "localhost",
		"DB_PORT":        "5432",
		"DB_USER":        "postgres",
		"DB_PASSWORD":    "postgres",
		"DB_NAME":        "testdb",
		"REDIS_HOST":     "localhost",
		"REDIS_PORT":     "6379",
		"REDIS_PASSWORD": "",
		"JWT_SECRET":     "test-secret-key-for-integration-tests",
		"APP_ENV":        "test",
		"LOG_LEVEL":      "debug",
	}

	for key, value := range envDefaults {
		if os.Getenv(key) == "" {
			os.Setenv(key, value)
		}
	}
}

// SetupTestServer initializes the application for testing
func SetupTestServer(t *testing.T, database *gorm.DB, redisClient *redis.Client) http.Handler {
	// 1. Set test environment variables (required by config.Load)
	setTestEnvVars()

	// 2. Reset config singleton (in case previous test set it)
	config.Reset()

	// 3. Load configuration (required by middleware and other components)
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Failed to load configuration: %v", err)
	}

	// 4. Initialize logger (needs config for log level)
	log.InitLogger(cfg)

	// 5. Set the test database as the global database instance
	// This is required because the modules use db.GetDB() to get the database instance
	if database != nil {
		db.SetDB(database)
	}

	// 6. Set the test Redis client as the global Redis instance
	if redisClient != nil {
		cache.SetRedisClient(redisClient)
	}

	// 7. Initialize Gin Router
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(gin.Recovery())

	// 8. Apply middleware (same as main.go)
	router.Use(middleware.CorrelationID())
	router.Use(middleware.Logger())
	router.Use(middleware.CORS())

	// 9. Register modules
	registerContainer(router)

	return router
}

func registerContainer(router *gin.Engine) {
	_ = user.NewContainer(router)
	_ = product.NewContainer(router)
	_ = inventory.NewContainer(router)
	_ = order.NewContainer(router)
	_ = payment.NewContainer(router)
	_ = notification.NewContainer(router)
	_ = promotion.NewContainer(router)
}
