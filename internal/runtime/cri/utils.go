package cri

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	criapi "github.com/Klaven/cospeck/cri"
	"github.com/ghodss/yaml"
	v1 "k8s.io/api/core/v1"
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
	r, err := os.Open(defaultSandboxConfig)
	if err != nil {
		return nil, nil, err
	}

	rc, err := os.Open(defaultContainerConfig)
	if err != nil {
		return nil, nil, err
	}

	return ParseYamlFileWithPodConfig(file, r, rc)
}

// ParseYamlFileWithPodConfig parses a pods yaml file
func ParseYamlFileWithPodConfig(file []byte, sandboxConfig, containerConfig io.Reader) (*criapi.PodSandboxConfig, []criapi.ContainerConfig, error) {

	var spec v1.Pod
	err := yaml.Unmarshal(file, &spec)
	if err != nil {
		panic(err.Error())
	}
	pod, err := loadPodSandboxConfig(sandboxConfig)
	if err != nil {
		fmt.Println("error loading pod sandbox config")
		return nil, nil, err
	}
	containers := []criapi.ContainerConfig{}

	pod.Metadata.Name = spec.Name

	for _, container := range spec.Spec.Containers {
		c, err := loadContainerConfig(containerConfig)
		if err != nil {
			fmt.Println("error loading container config")
			return nil, nil, err
		}
		c.Args = container.Args
		c.Command = container.Command
		c.Image.Image = container.Image
		c.Metadata.Name = container.Name

		fmt.Println("image to start: ", container.Image)

		containers = append(containers, *c)
	}

	return pod, containers, nil
}
