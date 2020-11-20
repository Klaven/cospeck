package runtime

type Pod interface {
	Name() string
	PodID() string
	Containers() []*Container
	AddContainer(container *Container)
	GetContainer(name string) *Container
}
