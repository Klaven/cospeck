package cri

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"time"

	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	criapi "k8s.io/cri-api/pkg/apis/runtime/v1alpha2"
)

const (
	defaultPauseImage      = "k8s.gcr.io/pause:3.1"
	defaultPodNamePrefix   = "pod"
	defaultSandboxConfig   = "config/sandbox.json"
	defaultContainerConfig = "config/container.json"
)

var (
	pconfigGlobal criapi.PodSandboxConfig
	cconfigGlobal criapi.ContainerConfig
)

// Runtime is an implementation of the cri API
type Runtime struct {
	criSocketAddress string
	runtimeClient    *criapi.RuntimeServiceClient
	imageClient      *criapi.ImageServiceClient
	pconfig          criapi.PodSandboxConfig
	cconfig          criapi.ContainerConfig
}

// NewRuntime creates an instance of the CRI runtime
func NewRuntime(path string) (*Runtime, error) {
	if path == "" {
		return nil, fmt.Errorf("socket path unspecified")
	}

	conn, err := getGRPCConn(path, time.Duration(10*time.Second))
	if err != nil {
		return nil, err
	}

	runtimeClient := criapi.NewRuntimeServiceClient(conn)
	imageClient := criapi.NewImageServiceClient(conn)

	pconfig, err := loadPodSandboxConfig(defaultSandboxConfig)
	if err != nil {
		return nil, err
	}

	cconfig, err := loadContainerConfig(defaultContainerConfig)
	if err != nil {
		return nil, err
	}

	runtime := &Runtime{
		criSocketAddress: path,
		runtimeClient:    &runtimeClient,
		imageClient:      &imageClient,
		cconfig:          cconfig,
		pconfig:          pconfig,
	}

	return runtime, nil
}

func getGRPCConn(socket string, timeout time.Duration) (*grpc.ClientConn, error) {
	conn, err := grpc.Dial(socket, grpc.WithInsecure(), grpc.WithTimeout(timeout),
		grpc.WithDialer(func(addr string, timeout time.Duration) (net.Conn, error) {
			return net.DialTimeout("unix", addr, timeout)
		}))
	if err != nil {
		return nil, fmt.Errorf("failed to connect: %v", err)
	}
	return conn, nil
}

// Info returns a string with information about the container engine/runtime details
func (c *Runtime) Info(ctx context.Context) (string, error) {
	version, err := (*c.runtimeClient).Version(ctx, &criapi.VersionRequest{})
	if err != nil {
		return "", err
	}

	info := "CRI Client runtime (Version: " + version.GetVersion() + ", API Version: " + version.GetRuntimeApiVersion() + " Runtime" + version.GetRuntimeName() + version.GetRuntimeVersion() + " )"

	return info, nil
}

// Path returns the binary (or socket) path related to the runtime in use
func (c *Runtime) Path() string {
	return c.criSocketAddress
}

// CreateContainer will create a container instance matching the specific needs
// No pod sandbox is created.
func (c *Runtime) CreateContainer(ctx context.Context, name, image, cmdOverride string, trace bool) (*Container, error) {
	if status, err := (*c.imageClient).ImageStatus(ctx, &criapi.ImageStatusRequest{Image: &criapi.ImageSpec{Image: image}}); err != nil || status.Image == nil {
		if _, err := (*c.imageClient).PullImage(ctx, &criapi.PullImageRequest{Image: &criapi.ImageSpec{Image: image}}); err != nil {
			return nil, err
		}
	}

	if status, err := (*c.imageClient).ImageStatus(ctx, &criapi.ImageStatusRequest{Image: &criapi.ImageSpec{Image: defaultPauseImage}}); err != nil || status.Image == nil {
		if _, err := (*c.imageClient).PullImage(ctx, &criapi.PullImageRequest{Image: &criapi.ImageSpec{Image: defaultPauseImage}}); err != nil {
			return nil, err
		}
	}

	containerObj := &Container{
		name:        name,
		imageName:   image,
		cmdOverride: cmdOverride,
		trace:       trace,
	}

	return containerObj, nil
}

// CreatePodAndContainer will create a container instance inside a pod
func (c *Runtime) CreatePodAndContainer(ctx context.Context, name, image, cmdOverride string, trace bool) (*Pod, error) {
	if status, err := (*c.imageClient).ImageStatus(ctx, &criapi.ImageStatusRequest{Image: &criapi.ImageSpec{Image: image}}); err != nil || status.Image == nil {
		if _, err := (*c.imageClient).PullImage(ctx, &criapi.PullImageRequest{Image: &criapi.ImageSpec{Image: image}}); err != nil {
			return nil, err
		}
	}

	if status, err := (*c.imageClient).ImageStatus(ctx, &criapi.ImageStatusRequest{Image: &criapi.ImageSpec{Image: defaultPauseImage}}); err != nil || status.Image == nil {
		if _, err := (*c.imageClient).PullImage(ctx, &criapi.PullImageRequest{Image: &criapi.ImageSpec{Image: defaultPauseImage}}); err != nil {
			return nil, err
		}
	}

	pconfig := pconfigGlobal
	pconfig.Metadata.Name = defaultPodNamePrefix + name

	podInfo, err := (*c.runtimeClient).RunPodSandbox(ctx, &criapi.RunPodSandboxRequest{Config: &pconfig})
	if err != nil {
		return nil, err
	}

	cconfig := cconfigGlobal
	cconfig.Image.Image = image
	cconfig.Metadata.Name = name

	containerInfo, err := (*c.runtimeClient).CreateContainer(ctx, &criapi.CreateContainerRequest{PodSandboxId: podInfo.PodSandboxId, Config: &cconfig, SandboxConfig: &pconfig})
	if err != nil {
		return nil, err
	}

	containerObj := &Container{
		name:        name,
		imageName:   image,
		cmdOverride: cmdOverride,
		trace:       trace,
		containerID: containerInfo.ContainerId,
	}

	pod := &Pod{
		name:       pconfig.Metadata.Name,
		podID:      podInfo.PodSandboxId,
		containers: []*Container{containerObj},
	}
	return pod, nil
}

// CreatePod will create a Pod with no containers to be used later
func (c *Runtime) CreatePod(ctx context.Context, name string) (*Pod, error) {
	pconfig := pconfigGlobal
	pconfig.Metadata.Name = defaultPodNamePrefix + name

	podInfo, err := (*c.runtimeClient).RunPodSandbox(ctx, &criapi.RunPodSandboxRequest{Config: &pconfig})
	if err != nil {
		return nil, err
	}
	return &Pod{
		name:  pconfig.Metadata.Name,
		podID: podInfo.PodSandboxId,
	}, nil
}

// Clean will clean the operating environment of a specific runtime
func (c *Runtime) Clean(ctx context.Context) error {

	resp, err := (*c.runtimeClient).ListContainers(ctx, &criapi.ListContainersRequest{Filter: &criapi.ContainerFilter{}})
	if err != nil {
		return err
	}
	containers := resp.GetContainers()
	for _, ctr := range containers {
		podID := ctr.GetPodSandboxId()
		_, err := (*c.runtimeClient).StopContainer(ctx, &criapi.StopContainerRequest{ContainerId: ctr.GetId(), Timeout: 0})
		if err != nil {
			log.Errorf("Error stopping container: %v", err)
		}
		_, err = (*c.runtimeClient).RemoveContainer(ctx, &criapi.RemoveContainerRequest{ContainerId: ctr.GetId()})
		if err != nil {
			log.Errorf("Error deleting container %v", err)
		}
		_, err = (*c.runtimeClient).RemovePodSandbox(ctx, &criapi.RemovePodSandboxRequest{PodSandboxId: podID})
		if err != nil {
			log.Errorf("Error deleting pod %s, %v", podID, err)
		}
	}
	log.Infof("CRI cleanup complete.")
	return nil
}

// Run will execute a container using the cri runtime
func (c *Runtime) Run(ctx context.Context, ctr Container) (time.Duration, error) {
	start := time.Now()
	_, err := (*c.runtimeClient).StartContainer(ctx, &criapi.StartContainerRequest{ContainerId: ctr.GetContainerID()})
	elapsed := time.Since(start)
	return elapsed, err
}

// Stop will stop/kill a container will not stop a pod
func (c *Runtime) Stop(ctx context.Context, ctr *Container) (string, time.Duration, error) {
	start := time.Now()
	resp, err := (*c.runtimeClient).ListContainers(ctx, &criapi.ListContainersRequest{Filter: &criapi.ContainerFilter{Id: ctr.GetContainerID()}})
	if err != nil {
		return "", 0, nil
	}

	containers := resp.GetContainers()
	for _, ctr := range containers {
		podID := ctr.GetPodSandboxId()
		_, err := (*c.runtimeClient).StopContainer(ctx, &criapi.StopContainerRequest{ContainerId: ctr.GetId(), Timeout: 0})
		if err != nil {
			log.Errorf("Error Stoping container %v", err)
			return "", 0, nil
		}
		_, err = (*c.runtimeClient).StopPodSandbox(ctx, &criapi.StopPodSandboxRequest{PodSandboxId: podID})
		if err != nil {
			log.Errorf("Error Stoping pod %v", err)
			return "", 0, nil
		}
	}
	elapsed := time.Since(start)
	return "", elapsed, nil
}

// StopPod a pod, will stop all containers in the pod
func (c *Runtime) StopPod(ctx context.Context, pod *Pod) (time.Duration, error) {
	start := time.Now()
	_, err := (*c.runtimeClient).StopPodSandbox(ctx, &criapi.StopPodSandboxRequest{PodSandboxId: pod.PodID()})
	if err != nil {
		log.Errorf("Error Stoping pod %v", err)
		return 0, nil
	}

	elapsed := time.Since(start)
	return elapsed, nil
}

// RemovePod will remove a pod sandbox
func (c *Runtime) RemovePod(ctx context.Context, pod *Pod) (time.Duration, error) {

	start := time.Now()

	_, err := (*c.runtimeClient).RemovePodSandbox(ctx, &criapi.RemovePodSandboxRequest{PodSandboxId: pod.PodID()})
	if err != nil {
		log.Errorf("Error deleting pod %v", err)
		return 0, nil
	}

	elapsed := time.Since(start)
	return elapsed, nil
}

// Remove DEPRICATED will remove a container
func (c *Runtime) Remove(ctx context.Context, ctr *Container) (string, time.Duration, error) {

	start := time.Now()
	resp, err := (*c.runtimeClient).ListContainers(ctx, &criapi.ListContainersRequest{Filter: &criapi.ContainerFilter{Id: ctr.GetContainerID()}})
	if err != nil {
		return "", 0, nil
	}

	fmt.Println("Containers:", resp)

	containers := resp.GetContainers()
	for _, ctr := range containers {
		podID := ctr.GetPodSandboxId()
		_, err = (*c.runtimeClient).RemoveContainer(ctx, &criapi.RemoveContainerRequest{ContainerId: ctr.GetId()})
		if err != nil {
			log.Errorf("Error deleting container %v", err)
			return "", 0, nil
		}
		_, err = (*c.runtimeClient).RemovePodSandbox(ctx, &criapi.RemovePodSandboxRequest{PodSandboxId: podID})
		if err != nil {
			log.Errorf("Error deleting pod %v", err)
			return "", 0, nil
		}
	}
	elapsed := time.Since(start)
	return "", elapsed, nil
}

// Close allows the runtime to free any resources/close any
// connections
func (c *Runtime) Close() error {
	return nil
}

// PID returns daemon process id
func (c *Runtime) PID() (int, error) {
	return 0, errors.New("not implemented")
}

// Wait blocks thread until container stop
func (c *Runtime) Wait(ctx context.Context, ctr Container) (string, time.Duration, error) {
	return "", 0, errors.New("not implemented")
}

// Stats returns stats data from daemon for container
func (c *Runtime) Stats(ctx context.Context, ctr Container) (io.ReadCloser, error) {
	return nil, errors.New("not implemented")
}

// ProcNames returns the list of process names contributing to mem/cpu usage during overhead benchmark
func (c *Runtime) ProcNames() []string {
	return []string{}
}

func openFile(path string) (*os.File, error) {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("file %s not found", path)
		}
		return nil, err
	}
	return f, nil
}

func loadPodSandboxConfig(path string) (criapi.PodSandboxConfig, error) {
	f, err := openFile(path)
	if err != nil {
		return criapi.PodSandboxConfig{}, err
	}
	defer f.Close()

	if err := json.NewDecoder(f).Decode(&pconfigGlobal); err != nil {
		return criapi.PodSandboxConfig{}, err
	}
	return pconfigGlobal, nil
}

func loadContainerConfig(path string) (criapi.ContainerConfig, error) {
	f, err := openFile(path)
	if err != nil {
		return criapi.ContainerConfig{}, err
	}
	defer f.Close()

	if err := json.NewDecoder(f).Decode(&cconfigGlobal); err != nil {
		return criapi.ContainerConfig{}, err
	}
	return cconfigGlobal, nil
}
