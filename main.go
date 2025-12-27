package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"ecommerce-be/common/cache"
	"ecommerce-be/common/config"
	"ecommerce-be/common/db"
	logger "ecommerce-be/common/log"
	"ecommerce-be/common/middleware"
	"ecommerce-be/common/scheduler"
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
		// Can't use logger yet, just print
		fmt.Println("No .env file found")
	}

	/* Load Configuration */
	cfg, err := config.Load()
	if err != nil {
		fmt.Println("Failed to load configuration:", err)
		os.Exit(1)
	}

	/* Initialize Logger (needs config for log level) */
	logger.InitLogger(cfg)

	/* Connect Database */
	db.ConnectDB(cfg)

	/* Connect Redis */
	cache.ConnectRedis(cfg)

	/* Initialize Gin Router */
	gin.SetMode(cfg.Server.Mode)

	// Use gin.New() instead of gin.Default() to disable default logging
	router := gin.New()
	router.Use(gin.Recovery()) // Add recovery middleware

	/* Apply middleware */
	router.Use(middleware.CorrelationID()) // Mandatory correlation ID middleware
	router.Use(middleware.Logger())
	router.Use(middleware.CORS())

	/* Register modules */
	registerContainer(router)

	/* Start background workers (must be before router.Run which blocks) */
	go scheduler.StartRedisWorkerPool()

	/* Start Server with Graceful Shutdown */
	srv := &http.Server{
		Addr:    cfg.Server.Addr(),
		Handler: router,
	}

	// Start server in a goroutine
	go func() {
		logger.Info("Server starting on port " + cfg.Server.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("Failed to start server on port "+cfg.Server.Port, err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	gracefulShutdown(srv)
}

// gracefulShutdown handles OS signals and performs cleanup
func gracefulShutdown(srv *http.Server) {
	quit := make(chan os.Signal, 1)
	// SIGINT (Ctrl+C), SIGTERM (Docker/Kubernetes stop)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// Block until signal received
	sig := <-quit
	logger.Info("Received shutdown signal: " + sig.String())

	// Create a deadline for shutdown (give ongoing requests time to complete)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Shutdown HTTP server (stops accepting new requests, waits for ongoing)
	logger.Info("Shutting down HTTP server...")
	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("HTTP server forced to shutdown", err)
	}

	// Close database connections
	logger.Info("Closing database connections...")
	db.CloseDB()

	// Close Redis connections
	logger.Info("Closing Redis connections...")
	cache.CloseRedis()

	logger.Info("Server shutdown complete")
}

func registerContainer(router *gin.Engine) {
	_ = user.NewContainer(router)
	_ = product.NewContainer(router)
	_ = inventory.NewContainer(router)
	_ = order.NewContainer(router)
	_ = payment.NewContainer(router)
	_ = notification.NewContainer(router)
}
