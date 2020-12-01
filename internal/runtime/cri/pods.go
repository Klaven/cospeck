package cri

import "github.com/Klaven/cospeck/internal/runtime"

// Pod defaines a Pod
type Pod struct {
	name       string
	podID      string
	containers []runtime.Container
}

var _ runtime.Pod = &Pod{}
var _ runtime.Pod = (*Pod)(nil)

// Name returns the name of the Pod/sadbox
func (p *Pod) Name() string {
	return p.name
}

// PodID returns the pods id of the Pod/sadbox
func (p *Pod) PodID() string {
	return p.podID
}

// Containers returns the list of containers in the Pod/sadbox
func (p *Pod) Containers() []runtime.Container {
	return p.containers
}

// AddContainer adds a container to a pod (does not run it)
func (p *Pod) AddContainer(container runtime.Container) {
	p.containers = append(p.containers, container)
}

// GetContainer finds the container in the pod and returns it
func (p *Pod) GetContainer(name string) runtime.Container {
	for _, c := range p.containers {
		if c.Name() == name {
			return c
		}
	}
	return nil
}
