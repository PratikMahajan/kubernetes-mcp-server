package kubernetes

import (
	"context"
	"os/exec"
	"strings"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func (m *Manager) IsOpenShift(ctx context.Context) bool {
	// This method should be fast and not block (it's called at startup)
	timeoutSeconds := int64(5)
	if _, err := m.dynamicClient.Resource(schema.GroupVersionResource{
		Group:    "project.openshift.io",
		Version:  "v1",
		Resource: "projects",
	}).List(ctx, metav1.ListOptions{Limit: 1, TimeoutSeconds: &timeoutSeconds}); err == nil {
		return true
	}
	return false
}

// ExecuteOcCommand executes an oc command with the given arguments and returns the combined output.
func (m *Manager) ExecuteOcCommand(ctx context.Context, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "oc", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return strings.TrimSpace(string(output)), err
	}
	return strings.TrimSpace(string(output)), nil
}
