package stats

import (
	"context"
	"time"

	"github.com/Klaven/cospeck/internal/runtime/cri"
	"github.com/pkg/errors"
	criapi "k8s.io/cri-api/pkg/apis/runtime/v1alpha2"
)

type statsOptions struct {
	// all containers
	all bool
	// id of container
	id string
	// podID of container
	podID string
	// sample is the duration for sampling cpu usage.
	sample time.Duration
	// labels are selectors for the sandbox
	labels map[string]string
	// output format
	output string
	// live watch
	watch bool
}

// Stats gets them stats
func Stats(runtime *cri.Runtime, name string) (*MetricsV2, error) {

	opts := statsOptions{
		all:    true,
		id:     "",
		podID:  "",
		sample: time.Duration(2 * time.Second),
		output: "",
		watch:  false,
	}
	metrics := &MetricsV2{}
	var err error
	if metrics, err = ContainerStats(runtime.GetRuntimeClient(), opts, name); err != nil {
		return nil, errors.Wrap(err, "get container stats")
	}
	return metrics, nil
}

// ContainerStats sends a ListContainerStatsRequest to the server, and
// parses the returned ListContainerStatsResponse.
func ContainerStats(client *criapi.RuntimeServiceClient, opts statsOptions, name string) (*MetricsV2, error) {
	filter := &criapi.ContainerStatsFilter{}
	if opts.id != "" {
		filter.Id = opts.id
	}
	if opts.podID != "" {
		filter.PodSandboxId = opts.podID
	}
	if opts.labels != nil {
		filter.LabelSelector = opts.labels
	}
	request := &criapi.ListContainerStatsRequest{
		Filter: filter,
	}

	metrics := &MetricsV2{}
	var err error
	if metrics, err = displayStats(client, request, name); err != nil {
		return nil, err
	}

	return metrics, nil
}

func displayStats(client *criapi.RuntimeServiceClient, request *criapi.ListContainerStatsRequest, name string) (*MetricsV2, error) {
	r, err := getContainerStats(client, request)
	if err != nil {
		return nil, err
	}

	oldStats := make(map[string]*criapi.ContainerStats)
	for _, s := range r.GetStats() {
		oldStats[s.Attributes.Id] = s
	}

	r, err = getContainerStats(client, request)
	if err != nil {
		return nil, err
	}

	metrics := &MetricsV2{}
	metrics.Name = name
	for _, s := range r.GetStats() {
		cpu := s.GetCpu().GetUsageCoreNanoSeconds().GetValue()
		mem := s.GetMemory().GetWorkingSetBytes().GetValue()
		disk := s.GetWritableLayer().GetUsedBytes().GetValue()
		metrics.CPU += (cpu / 1000)
		metrics.Disk += (disk / bytesInMiB)
		metrics.Mem += (mem / bytesInMiB)

	}
	return metrics, nil
}

func getContainerStats(client *criapi.RuntimeServiceClient, request *criapi.ListContainerStatsRequest) (*criapi.ListContainerStatsResponse, error) {

	r, err := (*client).ListContainerStats(context.Background(), request)

	if err != nil {
		return nil, err
	}

	return r, nil
}
