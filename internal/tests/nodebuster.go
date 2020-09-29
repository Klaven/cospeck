package tests

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/Klaven/cospeck/internal/runtime/cri"
	"github.com/Klaven/cospeck/internal/stats"
	"github.com/tidwall/limiter"
)

type NodeBusterResults struct {
	Message string
	Total   int
	Error   error
}

// NodeBusterTest is a very basic general test of memory and CPU
func NodeBusterTest(testFlags *TestFlags, totalPods int) {
	fmt.Println("Running tests")
	sampler, err := stats.NewCGroupsSampler(testFlags.CGroupPath)
	if err != nil {
		fmt.Println(err)
		return
	}

	rt, err := cri.NewRuntime(testFlags.OCIRuntime, 30*time.Second, nil, nil)
	if err != nil {
		fmt.Println(err)
		return
	}

	ctx := context.Background()

	rt.Clean(ctx)

	initTotal, _ := sampler.Sample()
	fmt.Println("Total CPU: ", initTotal.CPU)
	fmt.Println("Total Memory: ", initTotal.Mem)

	fmt.Println("Starting Pods")

	l := limiter.New(testFlags.Threads)

	errorChan := make(chan *NodeBusterResults)

	for i := 0; ; /*don't stop*/ i++ {

		var podErr *NodeBusterResults
		select {
		case podErr = <-errorChan:
			fmt.Printf("This node can run %d of this pod before running into errors/n", podErr.Total)
			break
		default:
		}

		if podErr != nil {
			break
		}

		fmt.Println("starting pod number: ", i)
		runNumberAsString := strconv.Itoa(i)
		l.Begin()
		go func(i int, ctx context.Context, runtime *cri.Runtime, podConfigFile string, uid string, finished *limiter.Limiter) {
			err := createPod(ctx, rt, podConfigFile, runNumberAsString, l)
			if err != nil {
				errorChan <- &NodeBusterResults{
					Message: "It's Finished",
					Total:   i,
					Error:   err,
				}
			}
		}(i, ctx, rt, testFlags.PodConfigFile, runNumberAsString, l)
	}

	println("Finished Starting Pods")

	if err != nil {
		fmt.Println("Failed to get cgroup sampler")
	}

	fmt.Println("")
	fmt.Println("Stopping Pods")
	for _, p := range pods {
		l.Begin()
		stopPod(ctx, rt, &p, l)
	}

}
