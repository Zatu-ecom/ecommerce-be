package main

import (
	"fmt"
	"log"
	"os"

	"ecommerce-be/common"
	"ecommerce-be/common/db"
	"ecommerce-be/common/middleware"
	productManagement "ecommerce-be/product_management"
	userManagement "ecommerce-be/user_management"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load(".env")
	if err != nil {
		log.Println("No .env file found")
	}

	/* Connect Database */
	db.ConnectDB(autoMigrations())

	/* Connect Redis */
	common.ConnectRedis()

	/* Initialize Gin Router */
	gin.SetMode(gin.ReleaseMode) // Set to release mode for production
	router := gin.Default()

	/* Apply middleware */
	router.Use(middleware.Logger())
	router.Use(middleware.CORS())

	/* Register modules */
	registerContainer(router)

	/* Start Server */
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080" // Default port
	}
	fmt.Println("🚀 Server running on port:", port)
	router.Run(":" + port)
}

func registerContainer(router *gin.Engine) {
	_ = userManagement.NewContainer(router)
	_ = productManagement.NewContainer(router)
}

func autoMigrations() []db.AutoMigrate {
	return []db.AutoMigrate{
		userManagement.NewUserAutoMigrate(),
		productManagement.NewProductAutoMigrate(),
	}
}
