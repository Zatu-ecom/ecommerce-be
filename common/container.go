package common

type Container struct {
	Modules []Module // List of modules (User, Auth, etc.)
}

/* RegisterModule adds a new module dynamically */
func (c *Container) RegisterModule(m Module) {
	c.Modules = append(c.Modules, m)
}
