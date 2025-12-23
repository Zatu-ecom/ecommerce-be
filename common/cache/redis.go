package cache

import (
	"context"
	"errors"
	"os"
	"time"

	"ecommerce-be/common/constants"

	"github.com/go-redis/redis/v8"
)

var (
	redisClient *redis.Client
	ctx         = context.Background()
)

// InitializeRedis initializes the Redis client
func InitializeRedis() error {
	addr := os.Getenv("REDIS_ADDR")
	password := os.Getenv("REDIS_PASSWORD")
	db := 0

	redisClient = redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})

	return nil
}

// ConnectRedis is a backward-compatible alias for InitializeRedis
func ConnectRedis() error {
	return InitializeRedis()
}

func SetRedisClient(client *redis.Client) {
	redisClient = client
}

// GetRedisClient returns the Redis client instance
func GetRedisClient() (*redis.Client, error) {
	if redisClient == nil {
		return nil, errors.New(constants.REDIS_NOT_INITIALIZED_MSG)
	}
	return redisClient, nil
}

// Set sets a key-value pair in Redis with expiration
func Set(key string, value any, expiration time.Duration) error {
	if redisClient == nil {
		return errors.New(constants.REDIS_NOT_INITIALIZED_MSG)
	}
	return redisClient.Set(ctx, key, value, expiration).Err()
}

// Get retrieves a value from Redis
func Get(key string) (string, error) {
	if redisClient == nil {
		return "", errors.New(constants.REDIS_NOT_INITIALIZED_MSG)
	}
	return redisClient.Get(ctx, key).Result()
}

// Del deletes a key from Redis
func Del(key string) error {
	if redisClient == nil {
		return errors.New(constants.REDIS_NOT_INITIALIZED_MSG)
	}
	return redisClient.Del(ctx, key).Err()
}

// BlacklistToken stores a token in Redis with an expiration time
func BlacklistToken(token string, expiration time.Duration) error {
	client, err := GetRedisClient()
	if err != nil {
		return err
	}

	return client.Set(ctx, token, "blacklisted", expiration).Err()
}

// IsTokenBlacklisted checks if a token is blacklisted
func IsTokenBlacklisted(token string) bool {
	client, err := GetRedisClient()
	if err != nil {
		return false
	}

	result, err := client.Get(ctx, token).Result()
	if err != nil {
		return false
	}

	return result == "blacklisted"
}

// CloseRedis closes the Redis connection gracefully
func CloseRedis() {
	if redisClient != nil {
		if err := redisClient.Close(); err != nil {
			// Log error but don't panic during shutdown
			println("Error closing Redis connection:", err.Error())
		}
	}
}
