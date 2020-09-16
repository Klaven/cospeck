package tests

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/Klaven/cospeck/internal/runtime/cri"
	"github.com/Klaven/cospeck/internal/stats"
)

type testPod struct {
	CreationTime    time.Duration
	DestructionTime time.Duration
	AverageMemory   int64
	Pod             *cri.Pod
}

// Find finds a pod in a list of test pods
func Find(a []testPod, x *cri.Pod) int {
	for i, n := range a {
		if x == n.Pod {
			return i
		}
	}
	return len(a)
}

func GeneralTest(testFlags *TestFlags, totalPods int) {
	fmt.Println("Running tests")

	pods := []testPod{}

	sampler, err := stats.NewCGroupsSampler("/system.slice/crio.service")
	rt, err := cri.NewRuntime("/var/run/crio/crio.sock", 30*time.Second)
	ctx := context.Background()

	rt.Clean(ctx)

	initTotal, _ := sampler.Sample()
	fmt.Println("Total CPU: ", initTotal.CPU)
	fmt.Println("Total Memory: ", initTotal.Mem)

	var totalStart int64 = 0
	for i := 0; i < totalPods; i++ {
		println("Starting pod: ", i)

		if err != nil {
			fmt.Println(err)
			return
		}

		s := strconv.Itoa(i)

		podName := "nginx-" + s + "-pod"

		ct, err := rt.CreatePodAndContainer(ctx, podName, "docker.io/library/alpine:latest", "sleep 5000", false)

		if err != nil {
			fmt.Println(err)
			fmt.Println("error here fool")
			return
		}

		startDeration, err := rt.Run(ctx, *(ct.GetContainer(podName)))
		totalStart += startDeration.Milliseconds()
		if err != nil {
			fmt.Println("error starting container you dumb dumb: ", err)
		}

		pods = append(pods, testPod{
			Pod:          ct,
			CreationTime: startDeration,
		})
	}

	println("Finished Starting Pods")

	if err != nil {
		fmt.Println("Failed to get cgroup sampler")
	}

	//Some time to just let things settle down?
	time.Sleep(10 * time.Second)

	total, err := sampler.Sample()
	stat, err := sampler.Stat()
	var totalStopping int64 = 0

	for _, p := range pods {
		fmt.Println("stopping pod ", p.Pod.Name())
		duration, err := rt.StopPod(ctx, p.Pod)
		if err != nil {
			fmt.Println("duration:", duration)
			fmt.Println(err)
		}
		totalStopping += duration.Milliseconds()
		p.DestructionTime = duration
	}

	fmt.Println("")
	fmt.Println("Stats: ", stat)
	fmt.Println("")

	fmt.Println("Total CPU: ", total.CPU)
	fmt.Println("Percent CPU: ", total.CPUPercent)
	fmt.Println("Total Memory: ", total.Mem)
	fmt.Println("Average Start Time: ", (totalStart / int64(totalPods)))
	fmt.Println("Average Stop Time: ", (totalStopping / int64(totalPods)))

	//TODO: check to make sure namesapce is cleaned up first (and maybe should create the namespace, failing if it exists)
	//TODO: fail if not clean

}
