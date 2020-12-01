package runtime

import (
	"context"
	"time"
)

// Runtime is an inteface to a pod runtime
type Runtime interface {
	//StartPod(ctx context.Context, file string)
	StopPod(ctx context.Context, pod *Pod, file string) (time.Duration, error)
	RemovePod(ctx context.Context, pod *Pod, file string) (time.Duration, error)
	Close() error
	PID() (int, error)
	ProcNames() []string
	CreatePodAndContainerFromSpec(ctx context.Context, fileName, uid string) (Pod, error)
	Run(ctx context.Context, ctr Container) (time.Duration, error)
}
