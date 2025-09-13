package common

import "github.com/gin-gonic/gin"

// Module interface ensures every module can register itself
type Module interface {
	RegisterRoutes(router *gin.Engine)
}

type Container struct {
	Modules []Module // List of modules (User, Auth, etc.)
}

/* RegisterModule adds a new module dynamically */
func (c *Container) RegisterModule(m Module) {
	c.Modules = append(c.Modules, m)
}
