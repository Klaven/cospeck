package tests

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/Klaven/cospeck/internal/runtime/cri"
	"github.com/Klaven/cospeck/internal/stats"
	"github.com/tidwall/limiter"
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

var (
	mutex = &sync.Mutex{}
	pods  = make([]testPod, 0)
)

// GeneralTest is a very basic general test of memory and CPU
func GeneralTest(testFlags *TestFlags, totalPods int) {
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

	var totalStart int64 = 0
	fmt.Println("Starting Pods")

	l := limiter.New(testFlags.Threads)

	for i := 0; i < totalPods; i++ {
		fmt.Println("starting pod number: ", i)
		runNumberAsString := strconv.Itoa(i)
		l.Begin()
		go createPod(ctx, rt, testFlags.PodConfigFile, runNumberAsString, l)
	}

	println("Finished Starting Pods")

	if err != nil {
		fmt.Println("Failed to get cgroup sampler")
	}

	total, err := sampler.Sample()
	stat, err := sampler.Stat()
	//TODO make some type of output printer
	fmt.Println("Starting Total CPU: ", total.CPU)
	fmt.Println("Starting Percent CPU: ", total.CPUPercent)
	fmt.Println("Starting Total Memory: ", total.Mem)
	fmt.Println("Starting Average Start Time: ", (totalStart / int64(totalPods)))

	//Some time to just let things settle down... probably should be more accurate
	time.Sleep(10 * time.Second)

	total, err = sampler.Sample()
	stat, err = sampler.Stat()
	fmt.Println("10sec Total CPU: ", total.CPU)
	fmt.Println("10sec Percent CPU: ", total.CPUPercent)
	fmt.Println("10sec Total Memory: ", total.Mem)
	fmt.Println("10sec Average Start Time: ", (totalStart / int64(totalPods)))

	var totalStopping int64 = 0

	fmt.Println("")
	fmt.Println("Stopping Pods")
	for _, p := range pods {
		l.Begin()
		stopPod(ctx, rt, &p, l)
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

func stopPod(ctx context.Context, runtime *cri.Runtime, pod *testPod, finished *limiter.Limiter) {
	defer finished.End()
	duration, err := runtime.StopPod(ctx, pod.Pod)
	if err != nil {
		fmt.Println("duration:", duration)
		fmt.Println(err)
	}
	pod.DestructionTime = duration
}

func createPod(ctx context.Context, runtime *cri.Runtime, podConfigFile string, uid string, finished *limiter.Limiter) {
	defer finished.End()
	start := time.Now()
	ct, err := runtime.CreatePodAndContainerFromSpec(ctx, podConfigFile, uid)

	if err != nil {
		fmt.Println(err)
		fmt.Println("error here fool")
		return
	}

	for _, c := range ct.Containers() {
		_, err = runtime.Run(ctx, *c)
		if err != nil {
			fmt.Println("error starting container you dumb dumb: ", err)
		}
	}

	elapsed := time.Since(start)
	mutex.Lock()
	pods = append(pods, testPod{
		Pod:          ct,
		CreationTime: elapsed,
	})
	mutex.Unlock()
}
