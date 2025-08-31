package tests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"ecommerce-be/common"
	"ecommerce-be/common/middleware"
	"ecommerce-be/user_management/handlers"
	"ecommerce-be/user_management/repositories"
	"ecommerce-be/user_management/service"
	"ecommerce-be/user_management/utils"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// Test data structures matching the API specification
type RegisterRequest struct {
	FirstName       string `json:"firstName"`
	LastName        string `json:"lastName"`
	Email           string `json:"email"`
	Password        string `json:"password"`
	ConfirmPassword string `json:"confirmPassword"`
	Phone           string `json:"phone"`
	DateOfBirth     string `json:"dateOfBirth"`
	Gender          string `json:"gender"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type UpdateProfileRequest struct {
	FirstName   string `json:"firstName"`
	LastName    string `json:"lastName"`
	Phone       string `json:"phone"`
	DateOfBirth string `json:"dateOfBirth"`
	Gender      string `json:"gender"`
}

type AddAddressRequest struct {
	Street    string `json:"street"`
	City      string `json:"city"`
	State     string `json:"state"`
	ZipCode   string `json:"zipCode"`
	Country   string `json:"country"`
	IsDefault bool   `json:"isDefault"`
}

type ChangePasswordRequest struct {
	CurrentPassword string `json:"currentPassword"`
	NewPassword     string `json:"newPassword"`
	ConfirmPassword string `json:"confirmPassword"`
}

// IntegrationTestSuite holds the test suite dependencies
type IntegrationTestSuite struct {
	pgContainer *PostgreSQLTestContainer
	db          *gorm.DB
	router      *gin.Engine
	server      *httptest.Server
}

// SetupTestSuite initializes the test environment with PostgreSQL test container
func SetupTestSuite(t *testing.T) *IntegrationTestSuite {
	// Start PostgreSQL test container
	pgContainer := SetupPostgreSQLContainer(t)

	// Verify container is ready
	pgContainer.HealthCheck(t)

	// Get database connection
	_, db := pgContainer.GetConnectionInfo()

	// Initialize Redis for testing (mock or actual Redis instance)
	common.ConnectRedis()

	// Setup Gin router with test routes
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// Initialize your actual handlers here
	// This would be your actual router setup from main.go
	setupRoutes(router, db)

	// Create test server
	server := httptest.NewServer(router)

	return &IntegrationTestSuite{
		pgContainer: pgContainer,
		db:          db,
		router:      router,
		server:      server,
	}
}

// TearDownTestSuite cleans up the test environment
func (suite *IntegrationTestSuite) TearDown(t *testing.T) {
	// Close the test server
	if suite.server != nil {
		suite.server.Close()
	}

	// Close database connection
	if suite.db != nil {
		sqlDB, _ := suite.db.DB()
		if sqlDB != nil {
			sqlDB.Close()
		}
	}

	// Cleanup PostgreSQL container
	if suite.pgContainer != nil {
		suite.pgContainer.Cleanup(t)
	}
}

// CleanDatabase cleans test data between tests
func (suite *IntegrationTestSuite) CleanDatabase(t *testing.T) {
	if suite.pgContainer != nil {
		suite.pgContainer.CleanDatabase(t)
	}
}

// Helper function to make HTTP requests
func (suite *IntegrationTestSuite) makeRequest(method, path string, body interface{}, token string) (*http.Response, []byte) {
	var reqBody *bytes.Buffer
	if body != nil {
		jsonData, _ := json.Marshal(body)
		reqBody = bytes.NewBuffer(jsonData)
	} else {
		reqBody = bytes.NewBuffer([]byte{})
	}

	req, _ := http.NewRequest(method, suite.server.URL+path, reqBody)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	client := &http.Client{}
	resp, _ := client.Do(req)
	defer resp.Body.Close()

	responseBody := make([]byte, 0)
	if resp.Body != nil {
		buf := new(bytes.Buffer)
		buf.ReadFrom(resp.Body)
		responseBody = buf.Bytes()
	}

	return resp, responseBody
}

// Test User Registration
func TestUserRegistration(t *testing.T) {
	suite := SetupTestSuite(t)
	defer suite.TearDown(t)

	tests := []struct {
		name           string
		request        RegisterRequest
		expectedStatus int
		expectedCode   string
		checkResponse  func(t *testing.T, response APIResponse)
	}{
		{
			name: "Valid Registration",
			request: RegisterRequest{
				FirstName:       TestFirstName,
				LastName:        TestLastName,
				Email:           TestEmail,
				Password:        TestPassword,
				ConfirmPassword: TestPassword,
				Phone:           TestPhone,
				DateOfBirth:     TestDateOfBirth,
				Gender:          TestGender,
			},
			expectedStatus: http.StatusCreated,
			checkResponse: func(t *testing.T, response APIResponse) {
				assert.True(t, response.Success)
				assert.Equal(t, utils.RegisterSuccessMsg, response.Message)
				assert.NotNil(t, response.Data)
			},
		},
		{
			name: "Invalid Email Format",
			request: RegisterRequest{
				FirstName:       TestFirstName,
				LastName:        TestLastName,
				Email:           "invalid-email",
				Password:        TestPassword,
				ConfirmPassword: TestPassword,
				Phone:           TestPhone,
				DateOfBirth:     TestDateOfBirth,
				Gender:          TestGender,
			},
			expectedStatus: http.StatusBadRequest,
			expectedCode:   utils.ValidationErrorCode,
			checkResponse: func(t *testing.T, response APIResponse) {
				assert.False(t, response.Success)
				assert.Equal(t, utils.ValidationFailedMsg, response.Message)
				assert.Equal(t, utils.ValidationErrorCode, response.Code)
			},
		},
		{
			name: "Password Mismatch",
			request: RegisterRequest{
				FirstName:       TestFirstName,
				LastName:        TestLastName,
				Email:           "john2@example.com",
				Password:        TestPassword,
				ConfirmPassword: "DifferentPassword123!",
				Phone:           TestPhone,
				DateOfBirth:     TestDateOfBirth,
				Gender:          TestGender,
			},
			expectedStatus: http.StatusBadRequest,
			expectedCode:   utils.PasswordMismatchCode,
			checkResponse: func(t *testing.T, response APIResponse) {
				assert.False(t, response.Success)
				assert.Equal(t, utils.PasswordMismatchMsg, response.Message)
			},
		},
		{
			name: "Duplicate Email",
			request: RegisterRequest{
				FirstName:       "Jane",
				LastName:        "Smith",
				Email:           TestEmail, // Same as first test
				Password:        TestPassword,
				ConfirmPassword: TestPassword,
				Phone:           TestPhone,
				DateOfBirth:     TestDateOfBirth,
				Gender:          "female",
			},
			expectedStatus: http.StatusConflict,
			expectedCode:   utils.UserExistsCode,
			checkResponse: func(t *testing.T, response APIResponse) {
				assert.False(t, response.Success)
				assert.Equal(t, utils.UserExistsMsg, response.Message)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, body := suite.makeRequest("POST", EndpointRegister, tt.request, "")

			assert.Equal(t, tt.expectedStatus, resp.StatusCode)

			var response APIResponse
			err := json.Unmarshal(body, &response)
			require.NoError(t, err)

			if tt.expectedCode != "" {
				assert.Equal(t, tt.expectedCode, response.Code)
			}

			if tt.checkResponse != nil {
				tt.checkResponse(t, response)
			}
		})
	}
}

// Test User Login
func TestUserLogin(t *testing.T) {
	suite := SetupTestSuite(t)
	defer suite.TearDown(t)

	// First register a user
	registerReq := RegisterRequest{
		FirstName:       TestFirstName,
		LastName:        TestLastName,
		Email:           TestEmailLogin,
		Password:        TestPassword,
		ConfirmPassword: TestPassword,
		Phone:           TestPhone,
		DateOfBirth:     TestDateOfBirth,
		Gender:          TestGender,
	}
	suite.makeRequest("POST", EndpointRegister, registerReq, "")

	tests := []struct {
		name           string
		request        LoginRequest
		expectedStatus int
		expectedCode   string
		checkResponse  func(t *testing.T, response APIResponse)
	}{
		{
			name: "Valid Login",
			request: LoginRequest{
				Email:    TestEmailLogin,
				Password: TestPassword,
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, response APIResponse) {
				assert.True(t, response.Success)
				assert.Equal(t, utils.LoginSuccessMsg, response.Message)

				// Check if token is present in response
				data, ok := response.Data.(map[string]interface{})
				assert.True(t, ok)
				assert.NotEmpty(t, data["token"])
				assert.NotNil(t, data["user"])
			},
		},
		{
			name: "Invalid Email",
			request: LoginRequest{
				Email:    "nonexistent@example.com",
				Password: TestPassword,
			},
			expectedStatus: http.StatusUnauthorized,
			expectedCode:   utils.InvalidCredentialsCode,
			checkResponse: func(t *testing.T, response APIResponse) {
				assert.False(t, response.Success)
				assert.Equal(t, utils.InvalidCredentialsMsg, response.Message)
			},
		},
		{
			name: "Invalid Password",
			request: LoginRequest{
				Email:    TestEmailLogin,
				Password: "WrongPassword",
			},
			expectedStatus: http.StatusUnauthorized,
			expectedCode:   utils.InvalidCredentialsCode,
			checkResponse: func(t *testing.T, response APIResponse) {
				assert.False(t, response.Success)
				assert.Equal(t, utils.InvalidCredentialsMsg, response.Message)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, body := suite.makeRequest("POST", EndpointLogin, tt.request, "")

			assert.Equal(t, tt.expectedStatus, resp.StatusCode)

			var response APIResponse
			err := json.Unmarshal(body, &response)
			require.NoError(t, err)

			if tt.expectedCode != "" {
				assert.Equal(t, tt.expectedCode, response.Code)
			}

			if tt.checkResponse != nil {
				tt.checkResponse(t, response)
			}
		})
	}
}

// Test Token Refresh
func TestTokenRefresh(t *testing.T) {
	suite := SetupTestSuite(t)
	defer suite.TearDown(t)

	// Register and login to get a valid token
	token := suite.getValidToken(t)

	tests := []struct {
		name           string
		token          string
		expectedStatus int
		expectedCode   string
		checkResponse  func(t *testing.T, response APIResponse)
	}{
		{
			name:           "Valid Token Refresh",
			token:          token,
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, response APIResponse) {
				assert.True(t, response.Success)
				assert.Equal(t, utils.TokenRefreshedMsg, response.Message)

				data, ok := response.Data.(map[string]interface{})
				assert.True(t, ok)
				assert.NotEmpty(t, data["token"])
			},
		},
		{
			name:           "Invalid Token",
			token:          "invalid-token",
			expectedStatus: http.StatusUnauthorized,
			expectedCode:   utils.TokenInvalidCode,
			checkResponse: func(t *testing.T, response APIResponse) {
				assert.False(t, response.Success)
				assert.Equal(t, utils.TokenInvalidMsg, response.Message)
			},
		},
		{
			name:           "Missing Token",
			token:          "",
			expectedStatus: http.StatusUnauthorized,
			expectedCode:   common.AuthRequiredCode,
			checkResponse: func(t *testing.T, response APIResponse) {
				assert.False(t, response.Success)
				assert.Equal(t, common.AuthenticationRequiredMsg, response.Message)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, body := suite.makeRequest("POST", EndpointRefresh, map[string]interface{}{}, tt.token)

			assert.Equal(t, tt.expectedStatus, resp.StatusCode)

			var response APIResponse
			err := json.Unmarshal(body, &response)
			require.NoError(t, err)

			if tt.expectedCode != "" {
				assert.Equal(t, tt.expectedCode, response.Code)
			}

			if tt.checkResponse != nil {
				tt.checkResponse(t, response)
			}
		})
	}
}

// Test User Logout
func TestUserLogout(t *testing.T) {
	suite := SetupTestSuite(t)
	defer suite.TearDown(t)

	token := suite.getValidToken(t)

	resp, body := suite.makeRequest("POST", EndpointLogout, map[string]interface{}{}, token)

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var response APIResponse
	err := json.Unmarshal(body, &response)
	require.NoError(t, err)

	assert.True(t, response.Success)
	assert.Equal(t, utils.LogoutSuccessMsg, response.Message)
}

// Test Get User Profile
func TestGetUserProfile(t *testing.T) {
	suite := SetupTestSuite(t)
	defer suite.TearDown(t)

	token := suite.getValidToken(t)

	resp, body := suite.makeRequest("GET", EndpointProfile, nil, token)

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var response APIResponse
	err := json.Unmarshal(body, &response)
	require.NoError(t, err)

	assert.True(t, response.Success)
	assert.Equal(t, utils.ProfileRetrievedMsg, response.Message)
	assert.NotNil(t, response.Data)
}

// Test Update User Profile
func TestUpdateUserProfile(t *testing.T) {
	suite := SetupTestSuite(t)
	defer suite.TearDown(t)

	token := suite.getValidToken(t)

	updateReq := UpdateProfileRequest{
		FirstName:   "John Updated",
		LastName:    "Doe Updated",
		Phone:       "+1987654321",
		DateOfBirth: TestDateOfBirth,
		Gender:      TestGender,
	}

	resp, body := suite.makeRequest("PUT", EndpointUpdateProfile, updateReq, token)

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var response APIResponse
	err := json.Unmarshal(body, &response)
	require.NoError(t, err)

	assert.True(t, response.Success)
	assert.Equal(t, utils.ProfileUpdatedMsg, response.Message)
}

// Test Address Management
func TestAddressManagement(t *testing.T) {
	suite := SetupTestSuite(t)
	defer suite.TearDown(t)

	token := suite.getValidToken(t)

	// Test Add Address
	t.Run("Add Address", func(t *testing.T) {
		addAddressReq := AddAddressRequest{
			Street:    TestStreet,
			City:      TestCity,
			State:     TestState,
			ZipCode:   TestZipCode,
			Country:   TestCountry,
			IsDefault: true,
		}

		resp, body := suite.makeRequest("POST", EndpointAddresses, addAddressReq, token)

		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		var response APIResponse
		err := json.Unmarshal(body, &response)
		require.NoError(t, err)

		assert.True(t, response.Success)
		assert.Equal(t, utils.AddressCreatedMsg, response.Message)
	})

	// Test Get Addresses
	t.Run("Get Addresses", func(t *testing.T) {
		resp, body := suite.makeRequest("GET", EndpointAddresses, nil, token)

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var response APIResponse
		err := json.Unmarshal(body, &response)
		require.NoError(t, err)

		assert.True(t, response.Success)
		assert.Equal(t, utils.AddressesRetrievedMsg, response.Message)
	})

	// Test Update Address
	t.Run("Update Address", func(t *testing.T) {
		updateAddressReq := AddAddressRequest{
			Street:    "123 Main St Apt 2B",
			City:      TestCity,
			State:     TestState,
			ZipCode:   TestZipCode,
			Country:   TestCountry,
			IsDefault: true,
		}

		resp, body := suite.makeRequest("PUT", EndpointUpdateAddress, updateAddressReq, token)

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var response APIResponse
		err := json.Unmarshal(body, &response)
		require.NoError(t, err)

		assert.True(t, response.Success)
		assert.Equal(t, utils.AddressUpdatedMsg, response.Message)
	})

	// Test Set Default Address
	t.Run("Set Default Address", func(t *testing.T) {
		resp, body := suite.makeRequest("PATCH", EndpointSetDefault, map[string]interface{}{}, token)

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var response APIResponse
		err := json.Unmarshal(body, &response)
		require.NoError(t, err)

		assert.True(t, response.Success)
		assert.Equal(t, utils.DefaultAddressUpdatedMsg, response.Message)
	})
}

// Test Change Password
func TestChangePassword(t *testing.T) {
	suite := SetupTestSuite(t)
	defer suite.TearDown(t)

	token := suite.getValidToken(t)

	tests := []struct {
		name           string
		request        ChangePasswordRequest
		expectedStatus int
		expectedCode   string
		checkResponse  func(t *testing.T, response APIResponse)
	}{
		{
			name: "Valid Password Change",
			request: ChangePasswordRequest{
				CurrentPassword: TestPassword,
				NewPassword:     TestNewPassword,
				ConfirmPassword: TestNewPassword,
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, response APIResponse) {
				assert.True(t, response.Success)
				assert.Equal(t, utils.PasswordChangedMsg, response.Message)
			},
		},
		{
			name: "Password Mismatch",
			request: ChangePasswordRequest{
				CurrentPassword: TestPassword,
				NewPassword:     TestNewPassword,
				ConfirmPassword: "DifferentPassword123!",
			},
			expectedStatus: http.StatusBadRequest,
			expectedCode:   utils.PasswordMismatchCode,
			checkResponse: func(t *testing.T, response APIResponse) {
				assert.False(t, response.Success)
				assert.Equal(t, utils.PasswordMismatchMsg, response.Message)
			},
		},
		{
			name: "Invalid Current Password",
			request: ChangePasswordRequest{
				CurrentPassword: "WrongPassword",
				NewPassword:     TestNewPassword,
				ConfirmPassword: TestNewPassword,
			},
			expectedStatus: http.StatusBadRequest,
			expectedCode:   utils.InvalidCurrentPasswordCode,
			checkResponse: func(t *testing.T, response APIResponse) {
				assert.False(t, response.Success)
				assert.Equal(t, utils.InvalidCurrentPasswordMsg, response.Message)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, body := suite.makeRequest("PATCH", EndpointChangePassword, tt.request, token)

			assert.Equal(t, tt.expectedStatus, resp.StatusCode)

			var response APIResponse
			err := json.Unmarshal(body, &response)
			require.NoError(t, err)

			if tt.expectedCode != "" {
				assert.Equal(t, tt.expectedCode, response.Code)
			}

			if tt.checkResponse != nil {
				tt.checkResponse(t, response)
			}
		})
	}
}

// Test Authentication Middleware
func TestAuthenticationMiddleware(t *testing.T) {
	suite := SetupTestSuite(t)
	defer suite.TearDown(t)

	tests := []struct {
		name           string
		token          string
		expectedStatus int
		expectedCode   string
	}{
		{
			name:           "Missing Authorization Header",
			token:          "",
			expectedStatus: http.StatusUnauthorized,
			expectedCode:   common.AuthRequiredCode,
		},
		{
			name:           "Invalid Auth Format",
			token:          "no-bearer-prefix-token",
			expectedStatus: http.StatusUnauthorized,
			expectedCode:   common.InvalidAuthFormatCode,
		},
		{
			name:           "Invalid Token Format",
			token:          "invalid-token-format",
			expectedStatus: http.StatusUnauthorized,
			expectedCode:   common.TokenInvalidCode,
		},
		{
			name:           "Invalid Token",
			token:          "Bearer invalid-jwt-token",
			expectedStatus: http.StatusUnauthorized,
			expectedCode:   common.TokenInvalidCode,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, _ := http.NewRequest("GET", suite.server.URL+EndpointProfile, nil)
			req.Header.Set("Content-Type", "application/json")
			if tt.token != "" {
				if tt.token == "Bearer invalid-jwt-token" || tt.name == "Invalid Auth Format" {
					req.Header.Set("Authorization", tt.token)
				} else {
					req.Header.Set("Authorization", "Bearer "+tt.token)
				}
			}

			client := &http.Client{}
			resp, _ := client.Do(req)
			defer resp.Body.Close()

			assert.Equal(t, tt.expectedStatus, resp.StatusCode)

			if tt.expectedCode != "" {
				buf := new(bytes.Buffer)
				buf.ReadFrom(resp.Body)

				var response APIResponse
				err := json.Unmarshal(buf.Bytes(), &response)
				require.NoError(t, err)

				assert.Equal(t, tt.expectedCode, response.Code)
			}
		})
	}
}

// Helper function to get a valid token for authenticated tests
func (suite *IntegrationTestSuite) getValidToken(t *testing.T) string {
	// Register a user
	registerReq := RegisterRequest{
		FirstName:       "Test",
		LastName:        "User",
		Email:           fmt.Sprintf("test.user.%d@example.com", len(t.Name())),
		Password:        TestPassword,
		ConfirmPassword: TestPassword,
		Phone:           TestPhone,
		DateOfBirth:     TestDateOfBirth,
		Gender:          TestGender,
	}

	resp, body := suite.makeRequest("POST", EndpointRegister, registerReq, "")
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var response APIResponse
	err := json.Unmarshal(body, &response)
	require.NoError(t, err)

	data, ok := response.Data.(map[string]interface{})
	require.True(t, ok)

	token, ok := data["token"].(string)
	require.True(t, ok)
	require.NotEmpty(t, token)

	return token
}

// setupRoutes sets up the actual routes for testing
func setupRoutes(router *gin.Engine, db *gorm.DB) {
	// Set up repositories with test database
	userRepo := repositories.NewUserRepository(db)
	addressRepo := repositories.NewAddressRepository(db)

	// Set up services
	addressService := service.NewAddressService(addressRepo)
	userService := service.NewUserService(userRepo, addressService)

	// Set up handlers
	userHandler := handlers.NewUserHandler(userService)
	addressHandler := handlers.NewAddressHandler(addressService)

	// Set up middleware
	auth := middleware.Auth()

	// Authentication routes
	authRoutes := router.Group("/api/auth")
	{
		authRoutes.POST("/register", userHandler.Register)
		authRoutes.POST("/login", userHandler.Login)
		authRoutes.POST("/refresh", auth, userHandler.RefreshToken)
		authRoutes.POST("/logout", auth, userHandler.Logout)
	}

	// User routes
	userRoutes := router.Group("/api/users")
	{
		// User profile routes (protected)
		userRoutes.GET("/profile", auth, userHandler.GetProfile)
		userRoutes.PUT("/profile", auth, userHandler.UpdateProfile)
		userRoutes.PATCH("/password", auth, userHandler.ChangePassword)

		// Address routes (protected)
		userRoutes.GET("/addresses", auth, addressHandler.GetAddresses)
		userRoutes.POST("/addresses", auth, addressHandler.AddAddress)
		userRoutes.PUT("/addresses/:id", auth, addressHandler.UpdateAddress)
		userRoutes.DELETE("/addresses/:id", auth, addressHandler.DeleteAddress)
		userRoutes.PATCH("/addresses/:id/default", auth, addressHandler.SetDefaultAddress)
	}

	// Health check endpoint
	router.GET(EndpointHealth, func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})
}
