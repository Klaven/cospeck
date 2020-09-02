package runtime

import (
	cri "k8s.io/cri-api/pkg/apis/runtime/v1alpha2"
)

const (
	defaultPodImage        = "k8s.gcr.io/pause:3.1"
	defaultPodNamePrefix   = "pod"
	defaultSandboxConfig   = "contrib/sandbox_config.json"
	defaultContainerConfig = "contrib/container_config.json"
)

var (
	pconfigGlobal cri.PodSandboxConfig
	cconfigGlobal cri.ContainerConfig
)

// CRIDriver is an implementation of the driver interface for using k8s Container Runtime Interface.
// This uses the provided client library which abstracts using the gRPC APIs directly.
type CRIDriver struct {
	criSocketAddress string
	runtimeClient    *cri.RuntimeServiceClient
	imageClient      *cri.ImageServiceClient
	pconfig          cri.PodSandboxConfig
	cconfig          cri.ContainerConfig
}

// CRIContainer is an implementation of the container metadata needed for CRI implementation
type CRIContainer struct {
	name        string
	imageName   string
	cmdOverride string
	state       string
	process     string
	trace       bool
	podID       string
}
