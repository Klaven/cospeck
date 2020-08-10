package cmd

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"time"

	"github.com/spf13/cobra"
)

func RootCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "cospeck",
		Short: "A container runtime speed test",
		Run:   rootCmd,
	}
}

func rootCmd(cmd *cobra.Command, args []string) {
	fmt.Println("Running tests")
	var res string
	var err error

	//TODO: check to make sure namesapce is cleaned up first (and maybe should create the namespace, failing if it exists)
	//TODO: fail if not clean

	if res, err = kubectlRunner("create", "deployment", "test-deployment", "--image=nginx"); err != nil {
		fmt.Println("Failed to create deployment")
		return
	}

	tries := 1
	time.Sleep(1000 * time.Millisecond)
	for tries < 5 {
		fmt.Println("checking deployment")
		if res, err = kubectlRunner("get", "pods", "--field-selector=status.phase=Running", "--no-headers=true"); err != nil {
			return
		}

		fmt.Println(res)

		matched, _ := regexp.MatchString("1/1", res)

		if matched {
			break
		}
		fmt.Println("did not have the right deployment")

		time.Sleep(5000 * time.Millisecond)
		tries += 1
	}

	if tries >= 5 {
		fmt.Println("pod failed to deploy")
		return
	}

	// scale deployment

	if res, err = kubectlRunner("scale", "deployment/test-deployment", "--replicas=10"); err != nil {
		return
	}

	fmt.Println("scaled deploy: ", res)

	// check deployment

	if res, err = kubectlRunner("wait", "pods", "--for=condition=ready", "--all", "--timeout=5m"); err != nil {
		fmt.Println("waiting for pods error")
		return
	}

	fmt.Println("pods are ready")

	// check total memory

	/*
		if res, err = crictlRunner("get", "pods"); err != nil {
			return
		}
	*/

	/*
		var results map[string]interface{}
		json.Unmarshal([]byte(res), &results)
		fmt.Println(res)
	*/

	if res, err = crictlRunner("stats"); err != nil {
		fmt.Println("crictl stats error")
		return
	}

	fmt.Println("got current specs:")
	fmt.Println(res)
	fmt.Println("Cleanup time!")

	_, _ = cleanup()
}

func cleanup() (string, error) {
	return kubectlRunner("kubectl", "delete", "deployment", "test-deployment")
}

func kubectlRunner(args ...string) (string, error) {
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

func crictlRunner(args ...string) (string, error) {
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
