package common

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
)

// Token constants - using local constants

// JWT claims struct
type Claims struct {
	UserID uint   `json:"user_id"`
	Email  string `json:"email"`
	jwt.StandardClaims
}

// GenerateToken generates a JWT token for a user
func GenerateToken(userID uint, email string, secret string) (string, error) {
	// Create the claims
	claims := Claims{
		UserID: userID,
		Email:  email,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: time.Now().Add(TokenExpireDuration).Unix(),
			IssuedAt:  time.Now().Unix(),
		},
	}

	// Create token with claims
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Generate the signed token
	tokenString, err := token.SignedString([]byte(secret))
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

// ParseToken parses a JWT token
func ParseToken(tokenString string, secret string) (*Claims, error) {
	// Parse the token
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		// Validate the signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(secret), nil
	})

	if err != nil {
		return nil, err
	}

	// Validate the token and return claims
	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New(TokenInvalidMsg)
}

// AuthMiddleware creates a Gin middleware for JWT authentication
func AuthMiddleware(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get the Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			ErrorWithCode(c, http.StatusUnauthorized, AuthenticationRequiredMsg, AuthRequiredCode)
			c.Abort()
			return
		}

		// Check if the header has the Bearer prefix
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != BearerPrefix {
			ErrorWithCode(c, http.StatusUnauthorized, InvalidAuthFormatMsg, InvalidAuthFormatCode)
			c.Abort()
			return
		}

		// Parse and validate the token
		tokenString := parts[1]

		// Check if token is blacklisted
		if IsTokenBlacklisted(tokenString) {
			ErrorWithCode(c, http.StatusUnauthorized, TokenRevokedMsg, TokenRevokedCode)
			c.Abort()
			return
		}

		claims, err := ParseToken(tokenString, secret)
		if err != nil {
			ErrorWithCode(c, http.StatusUnauthorized, TokenInvalidMsg, TokenInvalidCode)
			c.Abort()
			return
		}

		// Set user info in context
		c.Set(UserIDKey, claims.UserID)
		c.Set(EmailKey, claims.Email)

		c.Next()
	}
}
