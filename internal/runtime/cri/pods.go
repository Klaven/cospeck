package cri

// Pod defaines a Pod
type Pod struct {
	name       string
	podID      string
	containers []*Container
}

// Name returns the name of the Pod/sadbox
func (c *Pod) Name() string {
	return c.name
}

// PodID returns the pods id of the Pod/sadbox
func (c *Pod) PodID() string {
	return c.podID
}

// Containers returns the list of containers in the Pod/sadbox
func (c *Pod) Containers() []*Container {
	return c.containers
}

// AddContainer adds a container to a pod (does not run it)
func (c *Pod) AddContainer(container *Container) {
	c.containers = append(c.containers, container)
}

// GetContainer finds the container in the pod and returns it
func (c *Pod) GetContainer(name string) *Container {
	for _, c := range c.containers {
		if c.name == name {
			return c
		}
	}
	return nil
}
