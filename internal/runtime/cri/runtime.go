package cri

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
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
	defaultPodConfig       = "config/pod.yaml"
)

// Runtime is an implementation of the cri API
type Runtime struct {
	criSocketAddress    string
	runtimeClient       *criapi.RuntimeServiceClient
	imageClient         *criapi.ImageServiceClient
	baseSandboxConfig   string
	baseContainerConfig string
	timeout             time.Duration
	baseYaml            []byte
}

// NewRuntime creates an instance of the CRI runtime
func NewRuntime(path string, timeout time.Duration, baseContainerConfig, baseSandboxConfig *string) (*Runtime, error) {
	if path == "" {
		return nil, fmt.Errorf("socket path unspecified")
	}

	bcc := defaultContainerConfig
	if baseContainerConfig != nil {
		bcc = *baseContainerConfig
	}

	bsc := defaultSandboxConfig
	if baseSandboxConfig != nil {
		bsc = *baseSandboxConfig
	}

	conn, err := getGRPCConn(path, time.Duration(10*time.Second))
	if err != nil {
		return nil, err
	}

	runtimeClient := criapi.NewRuntimeServiceClient(conn)
	imageClient := criapi.NewImageServiceClient(conn)

	runtime := &Runtime{
		criSocketAddress:    path,
		runtimeClient:       &runtimeClient,
		imageClient:         &imageClient,
		baseContainerConfig: bcc,
		baseSandboxConfig:   bsc,
		timeout:             timeout,
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

// PullImage pulls an image
func (c *Runtime) PullImage(ctx context.Context, image string) error {
	if status, err := (*c.imageClient).ImageStatus(ctx, &criapi.ImageStatusRequest{Image: &criapi.ImageSpec{Image: image}}); err != nil || status.Image == nil {
		if _, err := (*c.imageClient).PullImage(ctx, &criapi.PullImageRequest{Image: &criapi.ImageSpec{Image: image}}); err != nil {
			return err
		}
	}

	if status, err := (*c.imageClient).ImageStatus(ctx, &criapi.ImageStatusRequest{Image: &criapi.ImageSpec{Image: defaultPauseImage}}); err != nil || status.Image == nil {
		if _, err := (*c.imageClient).PullImage(ctx, &criapi.PullImageRequest{Image: &criapi.ImageSpec{Image: defaultPauseImage}}); err != nil {
			return err
		}
	}
	return nil
}

// CreateContainer creates a container in the specified pod
func (c *Runtime) CreateContainer(podSandBoxID string, config *criapi.ContainerConfig, sandboxConfig *criapi.PodSandboxConfig) (time.Duration, string, error) {
	start := time.Now()
	ctx, cancel := getContextWithTimeout(c.timeout)
	defer cancel()

	resp, err := (*c.runtimeClient).CreateContainer(ctx, &criapi.CreateContainerRequest{
		PodSandboxId:  podSandBoxID,
		Config:        config,
		SandboxConfig: sandboxConfig,
	})
	if err != nil {
		return 0, "", err
	}

	if resp.ContainerId == "" {
		errorMessage := fmt.Sprintf("ContainerId is not set for container %q", config.GetMetadata())
		return 0, "", errors.New(errorMessage)
	}

	elapsed := time.Since(start)
	return elapsed, resp.ContainerId, nil
}

// CreatePodAndContainerFromSpec simple helper function to create a pod and it's contaienrs from a spec
func (c *Runtime) CreatePodAndContainerFromSpec(ctx context.Context, fileName, uid string) (*Pod, error) {
	yamlFile, err := ioutil.ReadFile(fileName)
	if err != nil {
		fmt.Printf("Error reading YAML file: %s\n", err)
		return nil, err
	}
	p, con, err := ParseYamlFile(yamlFile)

	if err != nil {
		fmt.Println("Foolish human: ", err)
		return nil, err
	}

	p.Metadata.Name = defaultPodNamePrefix + p.Metadata.Name + uid

	podInfo, err := (*c.runtimeClient).RunPodSandbox(ctx, &criapi.RunPodSandboxRequest{Config: p})

	if err != nil {
		fmt.Println("Much Wow, Much Foolish: ", err)
		return nil, err
	}

	containers := []*Container{}

	for _, contain := range con {
		c.PullImage(ctx, contain.Image.Image)
		cconfig, err := loadContainerConfig(c.baseContainerConfig)

		if err != nil {
			fmt.Println("error reading in default pod file")
		}

		cconfig.Image.Image = contain.Image.Image
		cconfig.Command = contain.Command
		cconfig.Metadata.Name = contain.Metadata.Name
		_, containerID, err := c.CreateContainer(podInfo.PodSandboxId, &contain, p)
		if err != nil {
			fmt.Println("error creating container: ", err)
			continue
		}
		containers = append(containers,
			&Container{
				name:        contain.Metadata.Name,
				imageName:   contain.Image.Image,
				containerID: containerID,
			})
	}

	pod := &Pod{
		name:       p.Metadata.Name,
		podID:      podInfo.PodSandboxId,
		containers: containers,
	}
	return pod, nil
}

// CreatePod will create a Pod with no containers to be used later
func (c *Runtime) CreatePod(ctx context.Context, name string) (*Pod, error) {
	pconfig, err := loadPodSandboxConfig(c.baseSandboxConfig)
	if err != nil {
		fmt.Println("Error reading pod sandbox config: ", err)
		return nil, err
	}
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

	var config criapi.PodSandboxConfig

	if err := json.NewDecoder(f).Decode(&config); err != nil {
		return criapi.PodSandboxConfig{}, err
	}
	return config, nil
}

func loadContainerConfig(path string) (criapi.ContainerConfig, error) {
	f, err := openFile(path)
	if err != nil {
		return criapi.ContainerConfig{}, err
	}
	defer f.Close()

	var config criapi.ContainerConfig

	if err := json.NewDecoder(f).Decode(&config); err != nil {
		return criapi.ContainerConfig{}, err
	}
	return config, nil
}
