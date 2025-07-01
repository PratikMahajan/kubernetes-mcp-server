package updates

import (
	"context"
	"fmt"

	"github.com/manusa/kubernetes-mcp-server/pkg/kubernetes"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// GetClusterVersion retrieves the OpenShift cluster version by reading the
// ClusterVersion named "version" that the OpenShift cluster-version-operator
// maintains. The version string is typically something like "4.15.9".
//
// It expects to be executed against an OpenShift cluster. If executed against a
// non-OpenShift cluster the returned error will reflect the missing resource.
func GetClusterVersion(ctx context.Context, k *kubernetes.Kubernetes) (string, error) {
	gvk := schema.GroupVersionKind{Group: "config.openshift.io", Version: "v1", Kind: "ClusterVersion"}

	// In OpenShift there is a single ClusterVersion object named "version" that
	// is cluster-scoped (not namespaced).
	obj, err := k.ResourcesGet(ctx, &gvk, "", "version")
	if err != nil {
		return "", fmt.Errorf("cannot retrieve OpenShift ClusterVersion resource: %w", err)
	}

	// Parse the version from status.desired.version or fall back to the first
	// entry in status.history[].version.
	status, ok := obj.Object["status"].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("unexpected ClusterVersion payload: missing status")
	}

	// First, try the canonical field.
	if desired, ok := status["desired"].(map[string]interface{}); ok {
		if v, ok := desired["version"].(string); ok && v != "" {
			return v, nil
		}
	}

	// Fallback to history[0].version
	if history, ok := status["history"].([]interface{}); ok && len(history) > 0 {
		if entry, ok := history[0].(map[string]interface{}); ok {
			if v, ok := entry["version"].(string); ok && v != "" {
				return v, nil
			}
		}
	}

	return "", fmt.Errorf("version field not found in ClusterVersion status")
}
