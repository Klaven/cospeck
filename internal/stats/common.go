package stats

const bytesInMiB = 1024 * 1024

// Metrics represents stats sample from daemon
type Metrics struct {
	Mem        uint64
	CPU        float64
	CPUPercent float64
	Name       string
}

// Process represents an interfaces of a daemon to be sampled
type Process interface {
	// PID returns daemon process id
	PID() (int, error)

	// ProcNames returns the list of process names contributing to mem/cpu usage during overhead benchmark
	ProcNames() []string
}

// Sampler represents an interface of a sampler
type Sampler interface {
	// Sample a process metrics or error
	Sample()
	GetCPU()
	GetMemory()
}
