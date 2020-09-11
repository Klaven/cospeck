package cmd

import (
	"context"
	"fmt"
	"regexp"
	"time"

	"github.com/Klaven/cospeck/internal/cospeck/utils"
	"github.com/Klaven/cospeck/internal/runtime/cri"
	"github.com/Klaven/cospeck/internal/tests"
	"github.com/spf13/cobra"
)

// Flags represent cmd line flags
type Flags struct {
	Runtime       string
	CreateCluster bool
}

// RootCmd is the root command builder thing
func RootCmd() *cobra.Command {
	globalFlags := &Flags{}
	testFlags := &tests.TestFlags{}

	cmd := &cobra.Command{
		Use:   "cospeck",
		Short: "A container runtime speed test",
		Run:   runContainerCmd,
	}

	// subcommands
	cmd.AddCommand(testCmd(globalFlags, testFlags))
	cmd.AddCommand(nodeBusterCmd(globalFlags))

	// Flags
	cmd.PersistentFlags().StringVarP(&globalFlags.Runtime, "runtime", "r", "/var/run/crio/crio.sock", "Runtime to use default: /var/run/crio/crio.sock")
	// really I would like to take the kubernetes cluster out of it eventually. but right now it makes some things easy
	cmd.PersistentFlags().BoolP("create-runtime", "c", true, "Create a cluster")

	return cmd
}

func runContainerCmd(cmd *cobra.Command, args []string) {

	rt, err := cri.NewRuntime("/var/run/crio/crio.sock", 30*time.Second)

	if err != nil {
		fmt.Println(err)
		return
	}

	ctx := context.Background()

	rt.Clean(ctx)

	ct, err := rt.CreatePodAndContainer(ctx, "nginx-pod", "docker.io/library/alpine:latest", "sleep 5000", false)

	if err != nil {
		fmt.Println(err)
		fmt.Println("error here fool")
		return
	}

	_, err = rt.Run(ctx, *(ct.GetContainer("nginx-pod")))

	if err != nil {
		fmt.Println("error starting container you dumb dumb: ", err)
	}

	time.Sleep(30 * time.Second)

	duration, err := rt.StopPod(ctx, ct)
	if err != nil {
		fmt.Println("duration:", duration)
		fmt.Println(err)
	}
	/*
		duration, err = rt.Remove(ctx, ct)
		if err != nil {
			fmt.Println(thing)
			fmt.Println("duration:", duration)
			fmt.Println(err)
		}
	*/
	rt.RemovePod(ctx, ct)

	rt.Clean(ctx)
}

func rootCmd(cmd *cobra.Command, args []string) {
	fmt.Println("Running tests")
	var res string
	var err error

	//TODO: check to make sure namesapce is cleaned up first (and maybe should create the namespace, failing if it exists)
	//TODO: fail if not clean

	if res, err = utils.KubectlRunner("create", "deployment", "test-deployment", "--image=nginx"); err != nil {
		fmt.Println("Failed to create deployment")
		return
	}

	tries := 1
	time.Sleep(1000 * time.Millisecond)
	for tries < 5 {
		fmt.Println("checking deployment")
		if res, err = utils.KubectlRunner("get", "pods", "--field-selector=status.phase=Running", "--no-headers=true"); err != nil {
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

	if res, err = utils.KubectlRunner("scale", "deployment/test-deployment", "--replicas=90"); err != nil {
		return
	}

	fmt.Println("scaled deploy: ", res)

	// check deployment

	if res, err = utils.KubectlRunner("wait", "pods", "--for=condition=ready", "--all", "--timeout=5m"); err != nil {
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

	if res, err = utils.CrictlRunner("stats"); err != nil {
		fmt.Println("crictl stats error")
		return
	}

	fmt.Println("got current specs:")
	fmt.Println(res)
	fmt.Println("Cleanup time!")

	_, _ = utils.Cleanup()
}
