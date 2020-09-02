package runtime

type Runtime interface {
	// RunPod creates a pod (should I worry about pods right now)
	Type()
	RunPod()
	RunContainer()
	StopContainer()
	WaitRunContainer()
	WaitStopContainer()
}
