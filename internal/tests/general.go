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
	defer rt.Clean(ctx)
	// removes all pods before we start
	if testFlags.cleanRuntime {
		rt.Clean(ctx)
	}

	metricsRuntime := []stats.Metrics{}
	total, err := sampler.Sample("init")
	if err != nil {
		fmt.Println(err)
	}
	metricsRuntime = append(metricsRuntime, *total)

	metricsContainers := []stats.MetricsV2{}
	totalContainers, err := stats.Stats(rt, "init")
	if err != nil {
		fmt.Println(err)
	}
	metricsContainers = append(metricsContainers, *totalContainers)

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

	total, err = sampler.Sample("pods-created")
	if err != nil {
		fmt.Println(err)
	}
	metricsRuntime = append(metricsRuntime, *total)

	totalContainers, err = stats.Stats(rt, "pods-created")
	if err != nil {
		fmt.Println(err)
	}
	metricsContainers = append(metricsContainers, *totalContainers)

	//Some time to just let things settle down... probably should be more accurate
	time.Sleep(10 * time.Second)

	total, err = sampler.Sample("sleep-10")
	if err != nil {
		fmt.Println(err)
	}
	metricsRuntime = append(metricsRuntime, *total)

	totalContainers, err = stats.Stats(rt, "sleep-10")
	if err != nil {
		fmt.Println(err)
	}
	metricsContainers = append(metricsContainers, *totalContainers)

	fmt.Println("")
	fmt.Println("Stopping Pods")
	for _, p := range pods {
		l.Begin()
		stopPod(ctx, rt, &p, l)
	}

	total, err = sampler.Sample("stopping")
	if err != nil {
		fmt.Println(err)
	}
	metricsRuntime = append(metricsRuntime, *total)

	totalContainers, err = stats.Stats(rt, "stopping")
	if err != nil {
		fmt.Println(err)
	}

	metricsContainers = append(metricsContainers, *totalContainers)

	fmt.Println("--Container Metrics--")
	MetricsV2Writer(&metricsContainers)

	fmt.Println("")
	fmt.Println("--Runtime Metrics--")
	MetricsWriter(&metricsRuntime)

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

func createPod(ctx context.Context, runtime *cri.Runtime, podConfigFile string, uid string, finished *limiter.Limiter) error {
	defer finished.End()
	start := time.Now()
	ct, err := runtime.CreatePodAndContainerFromSpec(ctx, podConfigFile, uid)

	if err != nil {
		fmt.Println(err)
		return err
	}

	for _, c := range ct.Containers() {
		_, err = runtime.Run(ctx, *c)
		if err != nil {
			fmt.Println("error starting container you dumb dumb: ", err)
			return err
		}
	}

	elapsed := time.Since(start)
	mutex.Lock()
	pods = append(pods, testPod{
		Pod:          ct,
		CreationTime: elapsed,
	})
	mutex.Unlock()
	return nil
}
