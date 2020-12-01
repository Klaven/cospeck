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

	criapi "github.com/Klaven/cospeck/cri"
	"github.com/Klaven/cospeck/internal/runtime"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
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
	baseSandboxConfig   *criapi.PodSandboxConfig
	baseContainerConfig *criapi.ContainerConfig
	timeout             time.Duration
	baseYaml            []byte
}

var _ runtime.Runtime = &Runtime{}
var _ runtime.Runtime = (*Runtime)(nil)

// NewCRIRuntime creates an instance of the CRI runtime
func NewCRIRuntime(path string, timeout time.Duration, podSandboxConfigReader, containerConfigReader *io.Reader) (*Runtime, error) {
	if path == "" {
		return nil, fmt.Errorf("socket path unspecified")
	}
	var sandboxFile io.Reader
	if podSandboxConfigReader == nil {
		sandboxFile, _ = os.Open(defaultSandboxConfig)
	} else {
		sandboxFile = *podSandboxConfigReader
	}
	bsc, err := loadPodSandboxConfig(sandboxFile)

	var containerFile io.Reader
	if containerConfigReader == nil {
		containerFile, _ = os.Open(defaultContainerConfig)
	} else {
		containerFile = *containerConfigReader
	}
	bcc, err := loadContainerConfig(containerFile)

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

// GetRuntimeClient get runtime client
func (r *Runtime) GetRuntimeClient() *criapi.RuntimeServiceClient {
	return r.runtimeClient
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
func (r *Runtime) Info(ctx context.Context) (string, error) {
	version, err := (*r.runtimeClient).Version(ctx, &criapi.VersionRequest{})
	if err != nil {
		return "", err
	}

	info := "CRI Client runtime (Version: " + version.GetVersion() + ", API Version: " + version.GetRuntimeApiVersion() + " Runtime" + version.GetRuntimeName() + version.GetRuntimeVersion() + " )"

	return info, nil
}

// Path returns the binary (or socket) path related to the runtime in use
func (r *Runtime) Path() string {
	return r.criSocketAddress
}

// pullImage pulls an image
func (r *Runtime) pullImage(ctx context.Context, image string) error {
	if status, err := (*r.imageClient).ImageStatus(ctx, &criapi.ImageStatusRequest{Image: &criapi.ImageSpec{Image: image}}); err != nil || status.Image == nil {
		if _, err := (*r.imageClient).PullImage(ctx, &criapi.PullImageRequest{Image: &criapi.ImageSpec{Image: image}}); err != nil {
			return err
		}
	}

	if status, err := (*r.imageClient).ImageStatus(ctx, &criapi.ImageStatusRequest{Image: &criapi.ImageSpec{Image: defaultPauseImage}}); err != nil || status.Image == nil {
		if _, err := (*r.imageClient).PullImage(ctx, &criapi.PullImageRequest{Image: &criapi.ImageSpec{Image: defaultPauseImage}}); err != nil {
			return err
		}
	}
	return nil
}

// CreateContainer creates a container in the specified pod
func (r *Runtime) CreateContainer(podSandBoxID string, config *criapi.ContainerConfig, sandboxConfig *criapi.PodSandboxConfig) (time.Duration, string, error) {
	start := time.Now()
	ctx, cancel := getContextWithTimeout(r.timeout)
	defer cancel()

	resp, err := (*r.runtimeClient).CreateContainer(ctx, &criapi.CreateContainerRequest{
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
func (r *Runtime) CreatePodAndContainerFromSpec(ctx context.Context, fileName, uid string) (runtime.Pod, error) {
	yamlFile, err := ioutil.ReadFile(fileName)
	if err != nil {
		fmt.Printf("Error reading YAML file: %s\n", err)
		return nil, err
	}
	p, con, err := ParseYamlFile(yamlFile)

	if err != nil {
		fmt.Println("Error Parsing Yaml: ", err)
		return nil, err
	}

	p.Metadata.Name = defaultPodNamePrefix + p.Metadata.Name + uid

	podInfo, err := (*r.runtimeClient).RunPodSandbox(ctx, &criapi.RunPodSandboxRequest{Config: p})

	if err != nil {
		fmt.Println("Error Running Pod Sadbox: ", err)
		return nil, err
	}

	containers := []runtime.Container{}

	for _, contain := range con {
		r.pullImage(ctx, contain.Image.Image)
		clone := proto.Clone(r.baseContainerConfig)
		cconfig := criapi.ContainerConfig{}
		proto.Merge(&cconfig, clone)

		cconfig.Image.Image = contain.Image.Image
		cconfig.Command = contain.Command
		cconfig.Metadata.Name = contain.Metadata.Name
		_, containerID, err := r.CreateContainer(podInfo.PodSandboxId, &cconfig, p)
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
func (r *Runtime) CreatePod(ctx context.Context, name string) (*Pod, error) {

	clone := proto.Clone(r.baseSandboxConfig)
	pconfig := criapi.PodSandboxConfig{}
	proto.Merge(&pconfig, clone)

	pconfig.Metadata.Name = defaultPodNamePrefix + name

	podInfo, err := (*r.runtimeClient).RunPodSandbox(ctx, &criapi.RunPodSandboxRequest{Config: &pconfig})
	if err != nil {
		return nil, err
	}
	return &Pod{
		name:  pconfig.Metadata.Name,
		podID: podInfo.PodSandboxId,
	}, nil
}

// Clean will clean the operating environment of a specific runtime
func (r *Runtime) Clean(ctx context.Context) error {

	respp, err := (*r.runtimeClient).ListPodSandbox(ctx, &criapi.ListPodSandboxRequest{Filter: &criapi.PodSandboxFilter{}})
	if err != nil {
		return err
	}

	pods := respp.GetItems()

	for _, pod := range pods {
		_, err = (*r.runtimeClient).StopPodSandbox(ctx, &criapi.StopPodSandboxRequest{PodSandboxId: pod.Id})
		if err != nil {
			log.Errorf("Error deleting pod %s, %v", pod.Id, err)
		}
		_, err = (*r.runtimeClient).RemovePodSandbox(ctx, &criapi.RemovePodSandboxRequest{PodSandboxId: pod.Id})
		if err != nil {
			log.Errorf("Error deleting pod %s, %v", pod.Id, err)
		}
	}

	log.Infof("CRI cleanup complete.")
	return nil
}

// Run will execute a container using the cri runtime
func (r *Runtime) Run(ctx context.Context, ctr runtime.Container) (time.Duration, error) {
	start := time.Now()
	_, err := (*r.runtimeClient).StartContainer(ctx, &criapi.StartContainerRequest{ContainerId: ctr.ContainerID()})
	elapsed := time.Since(start)
	return elapsed, err
}

// Stop will stop/kill a container will not stop a pod
func (r *Runtime) Stop(ctx context.Context, ctr *Container) (string, time.Duration, error) {
	start := time.Now()
	resp, err := (*r.runtimeClient).ListContainers(ctx, &criapi.ListContainersRequest{Filter: &criapi.ContainerFilter{Id: ctr.ContainerID()}})
	if err != nil {
		return "", 0, nil
	}

	containers := resp.GetContainers()
	for _, ctr := range containers {
		podID := ctr.GetPodSandboxId()
		_, err := (*r.runtimeClient).StopContainer(ctx, &criapi.StopContainerRequest{ContainerId: ctr.GetId(), Timeout: 0})
		if err != nil {
			log.Errorf("Error Stoping container %v", err)
			return "", 0, nil
		}
		_, err = (*r.runtimeClient).StopPodSandbox(ctx, &criapi.StopPodSandboxRequest{PodSandboxId: podID})
		if err != nil {
			log.Errorf("Error Stoping pod %v", err)
			return "", 0, nil
		}
	}
	elapsed := time.Since(start)
	return "", elapsed, nil
}

// StopPod a pod, will stop all containers in the pod
func (r *Runtime) StopPod(ctx context.Context, pod *runtime.Pod, file string) (time.Duration, error) {
	start := time.Now()
	_, err := (*r.runtimeClient).StopPodSandbox(ctx, &criapi.StopPodSandboxRequest{PodSandboxId: (*pod).PodID()})
	if err != nil {
		log.Errorf("Error Stoping pod %v", err)
		return 0, nil
	}

	elapsed := time.Since(start)
	return elapsed, nil
}

// RemovePod will remove a pod sandbox
func (r *Runtime) RemovePod(ctx context.Context, pod *runtime.Pod, file string) (time.Duration, error) {

	start := time.Now()

	_, err := (*r.runtimeClient).RemovePodSandbox(ctx, &criapi.RemovePodSandboxRequest{PodSandboxId: (*pod).PodID()})
	if err != nil {
		log.Errorf("Error deleting pod %v", err)
		return 0, nil
	}

	elapsed := time.Since(start)
	return elapsed, nil
}

// Close allows the runtime to free any resources/close any
// connections
func (r *Runtime) Close() error {
	return nil
}

// PID returns daemon process id
func (r *Runtime) PID() (int, error) {
	return 0, errors.New("not implemented")
}

// Wait blocks thread until container stop
func (r *Runtime) Wait(ctx context.Context, ctr Container) (string, time.Duration, error) {
	return "", 0, errors.New("not implemented")
}

// Stats returns stats data from daemon for container
func (r *Runtime) Stats(ctx context.Context, ctr Container) (io.ReadCloser, error) {
	return nil, errors.New("not implemented")
}

// ProcNames returns the list of process names contributing to mem/cpu usage during overhead benchmark
func (r *Runtime) ProcNames() []string {
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

func loadPodSandboxConfig(file io.Reader) (*criapi.PodSandboxConfig, error) {
	var config criapi.PodSandboxConfig

	if err := json.NewDecoder(file).Decode(&config); err != nil {
		return &criapi.PodSandboxConfig{}, err
	}
	return &config, nil
}

func loadContainerConfig(file io.Reader) (*criapi.ContainerConfig, error) {
	var config criapi.ContainerConfig

	if err := json.NewDecoder(file).Decode(&config); err != nil {
		return &criapi.ContainerConfig{}, err
	}
	return &config, nil
}
