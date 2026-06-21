package error

import (
	"net/http"

	commonError "ecommerce-be/common/error"
	"ecommerce-be/product/utils"
)

var (
	// ErrPackageOptionNotFound is returned when a package option is not found
	ErrPackageOptionNotFound = &commonError.AppError{
		Code:       utils.PACKAGE_OPTION_NOT_FOUND_CODE,
		Message:    utils.PACKAGE_OPTION_NOT_FOUND_MSG,
		StatusCode: http.StatusNotFound,
	}
)
