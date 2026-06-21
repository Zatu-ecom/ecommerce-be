package user_test

import (
	"fmt"
	"net/http"
	"testing"

	userModel "ecommerce-be/user/model"
	"ecommerce-be/test/integration/helpers"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSellerRegisterWithBusinessLogoFileId(t *testing.T) {
	env := helpers.SetupFileStorageEnv(t, helpers.DefaultFileStorageEnvConfig())

	uploaderToken := helpers.Login(
		t,
		env.Client,
		helpers.Seller2Email,
		helpers.Seller2Password,
	)
	fileID := helpers.UploadSellerLogo(t, env.Server, uploaderToken)

	nextUserID := helpers.PredictNextUserID(t, env.Containers)
	helpers.ReassignFileOwnerToSeller(t, env.Containers, fileID, nextUserID)

	uniqueEmail := fmt.Sprintf("new-seller-%s@example.com", uuid.NewString()[:8])
	registerBody := userModel.SellerRegisterRequest{
		User: userModel.UserRegisterRequest{
			CreateUserRequest: userModel.CreateUserRequest{
				FirstName: "New",
				LastName:  "Seller",
				Email:     uniqueEmail,
				Password:  "seller123",
			},
			ConfirmPassword: "seller123",
		},
		Profile: userModel.SellerProfileCreateRequest{
			BusinessName:       "New Seller Store",
			BusinessLogoFileID: fileID,
		},
	}

	env.Client.SetToken("")
	w := env.Client.Post(t, "/api/user/seller/register", registerBody)
	resp := helpers.AssertSuccessResponse(t, w, http.StatusCreated)
	data, ok := resp["data"].(map[string]any)
	require.True(t, ok, "response: %#v", resp)

	var seller map[string]any
	if wrapped, ok := data["seller"].(map[string]any); ok {
		seller = wrapped
	} else {
		seller = data
	}
	profile, ok := seller["profile"].(map[string]any)
	require.True(t, ok, "seller payload: %#v", seller)
	logo, ok := profile["businessLogo"].(map[string]any)
	require.True(t, ok, "registration response must include businessLogo object")
	assert.Equal(t, fileID, logo["fileId"])
	assert.NotEmpty(t, logo["url"])

	token, ok := seller["token"].(string)
	require.True(t, ok)
	require.NotEmpty(t, token)

	loginW := env.Client.Post(t, "/api/user/auth/login", map[string]any{
		"email":    uniqueEmail,
		"password": "seller123",
	})
	loginResp := helpers.AssertSuccessResponse(t, loginW, http.StatusOK)
	loginData := loginResp["data"].(map[string]any)
	sellerProfile, ok := loginData["sellerProfile"].(map[string]any)
	require.True(t, ok, "login must include sellerProfile for new seller")
	loginProfile := sellerProfile["profile"].(map[string]any)
	loginLogo, ok := loginProfile["businessLogo"].(map[string]any)
	require.True(t, ok, "login profile must include businessLogo")
	assert.Equal(t, fileID, loginLogo["fileId"])
	assert.NotEmpty(t, loginLogo["url"])
}

func TestSellerUpdateProfileWithBusinessLogoFileId(t *testing.T) {
	env := helpers.SetupFileStorageEnv(t, helpers.DefaultFileStorageEnvConfig())

	token := helpers.Login(t, env.Client, helpers.Seller2Email, helpers.Seller2Password)
	env.Client.SetToken(token)

	fileID := helpers.UploadSellerLogo(t, env.Server, token)

	updateW := env.Client.Put(t, "/api/user/seller/profile", map[string]any{
		"businessLogoFileId": fileID,
	})
	updateResp := helpers.AssertSuccessResponse(t, updateW, http.StatusOK)
	profile := helpers.GetResponseData(t, updateResp, "profile")
	logo, ok := profile["businessLogo"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, fileID, logo["fileId"])
	assert.NotEmpty(t, logo["url"])
}
