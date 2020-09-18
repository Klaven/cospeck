package cri

import (
	"context"
	"time"

	v1 "k8s.io/api/core/v1"
	"k8s.io/api/rbac/v1beta1"
	"k8s.io/client-go/kubernetes/scheme"
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
	decode := scheme.Codecs.UniversalDeserializer().Decode
	obj, _, err := decode([]byte(file), nil, nil)

	if err != nil {
		return nil, nil, err
	}

	pod := &criapi.PodSandboxConfig{}
	containers := []criapi.ContainerConfig{}
	// now use switch over the type of the object
	// and match each type-case
	switch o := obj.(type) {
	case *v1.Pod:
		pod.Metadata.Name = o.Name

		for _, container := range o.Spec.Containers {
			c := criapi.ContainerConfig{
				Args:    container.Args,
				Command: container.Command,
				Image: &criapi.ImageSpec{
					Image: container.Image,
				},
				Metadata: &criapi.ContainerMetadata{
					Name: container.Name,
				},
			}

			containers = append(containers, c)
		}
		// o is a pod
	case *v1beta1.Role:
		// o is the actual role Object with all fields etc
	case *v1beta1.RoleBinding:
	case *v1beta1.ClusterRole:
	case *v1beta1.ClusterRoleBinding:
	case *v1.ServiceAccount:
	default:
		//o is unknown for us
	}

	return pod, containers, nil
}
