package main

import (
	"os"

	"ecommerce-be/common/cache"
	"ecommerce-be/common/db"
	logger "ecommerce-be/common/log"
	"ecommerce-be/common/middleware"
	"ecommerce-be/inventory"
	"ecommerce-be/notification"
	"ecommerce-be/order"
	"ecommerce-be/payment"
	product "ecommerce-be/product"
	user "ecommerce-be/user"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load(".env")
	if err != nil {
		logger.Info("No .env file found")
	}

	/* Initialize Logger */
	logger.InitLogger()

	/* Connect Database */
	db.ConnectDB()

	/* Connect Redis */
	cache.ConnectRedis()

	/* Initialize Gin Router */
	gin.SetMode(gin.ReleaseMode) // Set to release mode for production

	// Use gin.New() instead of gin.Default() to disable default logging
	router := gin.New()
	router.Use(gin.Recovery()) // Add recovery middleware

	/* Apply middleware */
	router.Use(middleware.CorrelationID()) // Mandatory correlation ID middleware
	router.Use(middleware.Logger())
	router.Use(middleware.CORS())

	/* Register modules */
	registerContainer(router)

	/* Start Server */
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080" // Default port
	}
	logger.Info("Server starting on port " + port)
	if err := router.Run(":" + port); err != nil {
		logger.Fatal("Failed to start server on port "+port, err)
	}
}

func registerContainer(router *gin.Engine) {
	_ = user.NewContainer(router)
	_ = product.NewContainer(router)
	_ = inventory.NewContainer(router)
	_ = order.NewContainer(router)
	_ = payment.NewContainer(router)
	_ = notification.NewContainer(router)
}
