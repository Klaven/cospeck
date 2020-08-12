package utils

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
)

// CrictlRunner make this suck less
func CrictlRunner(args ...string) (string, error) {
	cmd := exec.Command("crictl", args...)
	cmd.Env = append(os.Environ())
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		fmt.Println(err)
		return "", err
	}
	output := out.String()
	//fmt.Printf("Output: %q\n", output)
	return output, nil
}

// DockerRunner make this suck less
func DockerRunner(args ...string) (string, error) {
	cmd := exec.Command("docker", args...)
	cmd.Env = append(os.Environ())
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		fmt.Println(err)
		return "", err
	}
	output := out.String()
	//fmt.Printf("Output: %q\n", output)
	return output, nil
}

// KubectlRunner make this suck.... less...
func KubectlRunner(args ...string) (string, error) {
	args = append(args, "--namespace=performance-tests" /*, "-o=json"*/)
	cmd := exec.Command("kubectl", args...)
	cmd.Env = append(os.Environ(), "KUBECONFIG=/var/run/kubernetes/admin.kubeconfig")
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		fmt.Println(err)
		return "", err
	}
	output := out.String()
	//fmt.Printf("Output: %q\n", output)
	return output, nil
}
