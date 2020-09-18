package cri

import (
	"context"
	"fmt"
	"time"

	"github.com/ghodss/yaml"
	v1 "k8s.io/api/core/v1"
	criapi "k8s.io/cri-api/pkg/apis/runtime/v1alpha2"
)

// maxMsgSize use 16MB as the default message size limit.
// grpc library default is 4MB
const maxMsgSize = 1024 * 1024 * 16

// getContextWithTimeout returns a context with timeout.
func getContextWithTimeout(timeout time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), timeout)
}

// ParseYamlFile parses a pods yaml file
func ParseYamlFile(file []byte) (*criapi.PodSandboxConfig, []criapi.ContainerConfig, error) {

	var spec v1.Pod
	err := yaml.Unmarshal(file, &spec)
	if err != nil {
		panic(err.Error())
	}
	pod, err := loadPodSandboxConfig(defaultSandboxConfig)
	if err != nil {
		return nil, nil, err
	}
	containers := []criapi.ContainerConfig{}

	pod.Metadata.Name = spec.Name

	for _, container := range spec.Spec.Containers {
		c, err := loadContainerConfig(defaultContainerConfig)
		if err != nil {
			return nil, nil, err
		}
		c.Args = container.Args
		c.Command = container.Command
		c.Image.Image = container.Image
		c.Metadata.Name = container.Name

		fmt.Println("image to start: ", container.Image)

		containers = append(containers, c)
	}

	return &pod, containers, nil
}
