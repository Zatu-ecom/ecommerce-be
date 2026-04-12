package error

import (
	"net/http"

	commonError "ecommerce-be/common/error"
	"ecommerce-be/file/utils/constant"
)

var (
	ErrProviderNotFound = commonError.NewAppError(
		constant.FILE_PROVIDER_NOT_FOUND_CODE,
		constant.FILE_PROVIDER_NOT_FOUND_MSG,
		http.StatusBadRequest,
	)
	ErrInvalidRole = commonError.NewAppError(
		constant.FILE_INVALID_ROLE_CODE,
		constant.FILE_INVALID_ROLE_MSG,
		http.StatusForbidden,
	)
	ErrConfigNotFound = commonError.NewAppError(
		constant.FILE_CONFIG_NOT_FOUND_CODE,
		constant.FILE_CONFIG_NOT_FOUND_MSG,
		http.StatusNotFound,
	)
	ErrUnauthorized = commonError.NewAppError(
		constant.FILE_CONFIG_FORBIDDEN_CODE,
		constant.FILE_CONFIG_FORBIDDEN_MSG,
		http.StatusForbidden,
	)
	ErrSerializationFailed = commonError.NewAppError(
		constant.FILE_CONFIG_SERIALIZATION_ERR_CODE,
		constant.FILE_CONFIG_SERIALIZATION_ERR_MSG,
		http.StatusBadRequest,
	)
	ErrEncryptionFailed = commonError.NewAppError(
		constant.FILE_CONFIG_ENCRYPTION_ERR_CODE,
		constant.FILE_CONFIG_ENCRYPTION_ERR_MSG,
		http.StatusInternalServerError,
	)
	ErrPersistenceFailed = commonError.NewAppError(
		constant.FILE_CONFIG_PERSISTENCE_ERR_CODE,
		constant.FILE_CONFIG_PERSISTENCE_ERR_MSG,
		http.StatusInternalServerError,
	)
	ErrActivationFailed = commonError.NewAppError(
		constant.FILE_CONFIG_ACTIVATION_ERR_CODE,
		constant.FILE_CONFIG_ACTIVATION_ERR_MSG,
		http.StatusInternalServerError,
	)
	ErrListFailed = commonError.NewAppError(
		constant.FILE_CONFIG_LIST_ERR_CODE,
		constant.FILE_CONFIG_LIST_ERR_MSG,
		http.StatusInternalServerError,
	)
)
