package auth

import (
	"errors"
	"fmt"
	"os"
	"time"

	"ecommerce-be/common/constants"

	"github.com/dgrijalva/jwt-go"
)

// JWT claims struct with enhanced role-based information
type Claims struct {
	UserID    *uint   `json:"user_id"`             // Required - pointer to detect missing field
	Email     *string `json:"email"`               // Required - pointer to detect missing field
	RoleID    *uint   `json:"role_id"`             // Required - pointer to detect missing field
	RoleName  *string `json:"role_name"`           // Required - pointer to detect missing field
	RoleLevel *uint   `json:"role_level"`          // Required - pointer to detect missing field
	SellerID  *uint   `json:"seller_id,omitempty"` // Optional - only for seller-related users
	jwt.StandardClaims
}

// TokenUserInfo contains all user information needed for token generation
type TokenUserInfo struct {
	UserID    uint
	Email     string
	RoleID    uint
	RoleName  string
	RoleLevel uint
	SellerID  *uint // Optional - only for seller-related users
}

// GenerateToken generates a JWT token for a user with role-based information
func GenerateToken(userInfo TokenUserInfo, secret string) (string, error) {
	expiryHoursInStr := os.Getenv("JWT_EXPIRY_HOURS")
	expiryHours, err := time.ParseDuration(expiryHoursInStr + "h")
	if err != nil {
		return "", errors.New("invalid JWT_EXPIRY_HOURS value")
	}

	// Create the claims with pointers
	claims := Claims{
		UserID:    &userInfo.UserID,
		Email:     &userInfo.Email,
		RoleID:    &userInfo.RoleID,
		RoleName:  &userInfo.RoleName,
		RoleLevel: &userInfo.RoleLevel,
		SellerID:  userInfo.SellerID,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: time.Now().Add(expiryHours).Unix(),
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
	token, err := jwt.ParseWithClaims(
		tokenString,
		&Claims{},
		func(token *jwt.Token) (interface{}, error) {
			// Validate the signing method
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return []byte(secret), nil
		},
	)
	if err != nil {
		return nil, err
	}

	// Validate the token and return claims
	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New(constants.TOKEN_INVALID_MSG)
}
