package runtime

// Container We could make it generic if we wanted other runners.... in the future.
type Container interface {
	// Name returns its name
	Name() string

	// Detached returns whether the container is to be started in detached state
	Detached() bool

	// Trace returns whether the container should be traced
	Trace() bool

	// Image returns either a bundle path
	Image() string

	// Command returns an optional command that overrides the default image
	// "CMD" or "ENTRYPOINT" for the Docker and Containerd (gRPC) drivers
	Command() string

	//GetPodID return pod-id associated with container.
	GetPodID() string
}
