package common

import (
	"context"
	"errors"
	"os"
	"time"

	"github.com/go-redis/redis/v8"
)

var redisClient *redis.Client
var ctx = context.Background()

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

// GetRedisClient returns the Redis client instance
func GetRedisClient() (*redis.Client, error) {
	if redisClient == nil {
		return nil, errors.New(REDIS_NOT_INITIALIZED_MSG)
	}
	return redisClient, nil
}

// SetKey sets a key-value pair in Redis with expiration
func SetKey(key string, value interface{}, expiration time.Duration) error {
	if redisClient == nil {
		return errors.New(REDIS_NOT_INITIALIZED_MSG)
	}
	return redisClient.Set(ctx, key, value, expiration).Err()
}

// GetKey retrieves a value from Redis
func GetKey(key string) (string, error) {
	if redisClient == nil {
		return "", errors.New(REDIS_NOT_INITIALIZED_MSG)
	}
	return redisClient.Get(ctx, key).Result()
}

// DelKey deletes a key from Redis
func DelKey(key string) error {
	if redisClient == nil {
		return errors.New(REDIS_NOT_INITIALIZED_MSG)
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
