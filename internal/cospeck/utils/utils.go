package utils

// Cleanup the cluster
func Cleanup() (string, error) {
	return KubectlRunner("kubectl", "delete", "deployment", "test-deployment")
}
