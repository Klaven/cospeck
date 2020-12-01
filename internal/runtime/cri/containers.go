package cri

import "github.com/Klaven/cospeck/internal/runtime"

// Container is an implementation of the container metadata needed for CRI implementation
type Container struct {
	name        string
	imageName   string
	cmdOverride string
	state       string
	process     string
	trace       bool
	containerID string
}

var _ runtime.Container = &Container{}
var _ runtime.Container = (*Container)(nil)

// Name returns the name of the container
func (c *Container) Name() string {
	return c.name
}

// Detached returns whether the container is to be started in detached state
func (c *Container) Detached() bool {
	return true
}

// Trace returns whether the container should be traced
func (c *Container) Trace() bool {
	return c.trace
}

// Image returns either a bundle path
func (c *Container) Image() string {
	return c.imageName
}

// Command returns an optional command that overrides the default image
// "CMD" or "ENTRYPOINT" for the Docker and Containerd (gRPC) drivers
func (c *Container) Command() string {
	return c.cmdOverride
}

//ContainerID return containers ID
func (c *Container) ContainerID() string {
	return c.containerID
}
