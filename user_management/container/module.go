package container

import "github.com/gin-gonic/gin"

// Module interface ensures every module can register itself
type Module interface {
	RegisterRoutes(router *gin.Engine)
}

