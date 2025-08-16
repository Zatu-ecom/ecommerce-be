package common

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/go-redis/redis/v8"
)

var redisClient *redis.Client
var ctx = context.Background()

// ConnectRedis initializes a Redis client connection
func ConnectRedis() {
	redisHost := os.Getenv("REDIS_HOST")
	if redisHost == "" {
		redisHost = "localhost"
	}

	redisPort := os.Getenv("REDIS_PORT")
	if redisPort == "" {
		redisPort = "6379"
	}

	redisPassword := os.Getenv("REDIS_PASSWORD")

	redisDB := 0
	if dbEnv := os.Getenv("REDIS_DB"); dbEnv != "" {
		var err error
		redisDB, err = strconv.Atoi(dbEnv)
		if err != nil {
			redisDB = 0
		}
	}

	addr := fmt.Sprintf("%s:%s", redisHost, redisPort)
	fmt.Printf("Connecting to Redis at %s\n", addr)

	// Create a new Redis client
	redisClient = redis.NewClient(&redis.Options{
		Addr:        addr,
		Password:    redisPassword,
		DB:          redisDB,
		DialTimeout: 5 * time.Second,
	})

	// Test the connection
	_, err := redisClient.Ping(ctx).Result()
	if err != nil {
		fmt.Printf("Failed to connect to Redis: %v\n", err)
	} else {
		fmt.Println("🚀 Redis connected successfully!")
	}
}

// GetRedis returns the Redis client
func GetRedis() *redis.Client {
	return redisClient
}

// SetKey sets a key-value pair in Redis with expiration
func SetKey(key string, value interface{}, expiration time.Duration) error {
	if redisClient == nil {
		return errors.New(RedisNotInitializedMsg)
	}
	return redisClient.Set(ctx, key, value, expiration).Err()
}

// GetKey retrieves a value from Redis
func GetKey(key string) (string, error) {
	if redisClient == nil {
		return "", errors.New(RedisNotInitializedMsg)
	}
	return redisClient.Get(ctx, key).Result()
}

// DelKey deletes a key from Redis
func DelKey(key string) error {
	if redisClient == nil {
		return errors.New(RedisNotInitializedMsg)
	}
	return redisClient.Del(ctx, key).Err()
}

// BlacklistToken adds a JWT token to the blacklist
func BlacklistToken(token string, expiration time.Duration) error {
	if redisClient == nil {
		// If Redis is not available, log a warning but don't fail the operation
		// This allows the application to work without Redis (but tokens won't be blacklisted)
		fmt.Println("Warning: Redis is not available, token blacklisting is disabled")
		return nil
	}
	return SetKey("blacklist:"+token, "1", expiration)
}

// IsTokenBlacklisted checks if a token is blacklisted
func IsTokenBlacklisted(token string) bool {
	if redisClient == nil {
		// If Redis is not available, tokens can't be blacklisted
		return false
	}
	_, err := GetKey("blacklist:" + token)
	return err == nil
}
