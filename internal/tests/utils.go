package tests

// TestFlags is a struct that represents the flags that can be passed to flags
type TestFlags struct {
	Tests         string
	OCIRuntime    string
	CGroupPath    string
	PodConfigFile string
	Threads       int
}
