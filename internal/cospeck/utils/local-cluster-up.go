package utils

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
)

// TODO: low priority, but it would be nice to be able to automatically do this, might take a play from kubeadm phases

// ClusterUpRunner make this suck less
func ClusterUp(args ...string) (string, error) {
	args = append(args, "./hack/cluster-up.sh")
	cmd := exec.Command("bash", args...)
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
