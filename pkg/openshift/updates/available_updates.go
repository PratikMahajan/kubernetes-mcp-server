package updates

import (
    "context"
    "fmt"

    "github.com/manusa/kubernetes-mcp-server/pkg/kubernetes"
    "k8s.io/apimachinery/pkg/runtime/schema"
)

// Update represents an available update (or an entry in the update history)
// reported by the OpenShift ClusterVersion resource.
//
// Only the most relevant fields are included. The Image field may be empty
// when not provided by the ClusterVersion payload.
type Update struct {
    Version string `json:"version" yaml:"version"`
    Image   string `json:"image" yaml:"image"`
}

// Capabilities contains the lists of enabled and known capabilities that the
// ClusterVersion operator reports in the status.capabilities stanza.
//
// The Enabled slice lists the capabilities that are currently enabled in the
// cluster while Known lists all the capabilities recognised by the cluster
// (independently of whether they are enabled or not).
type Capabilities struct {
    Enabled []string `json:"enabledCapabilities" yaml:"enabledCapabilities"`
    Known   []string `json:"knownCapabilities" yaml:"knownCapabilities"`
}

// UpdateHistory extends Update with metadata about when the update was applied
// and its state.
//
// The time fields are kept as strings (RFC3339 format) to avoid introducing an
// extra time.Parse error path that provides little added value for callers that
// will usually just display the information back to the user.
type UpdateHistory struct {
    Update
    CompletionTime string `json:"completionTime,omitempty" yaml:"completionTime,omitempty"`
    StartedTime    string `json:"startedTime,omitempty" yaml:"startedTime,omitempty"`
    State          string `json:"state,omitempty" yaml:"state,omitempty"`
    Verified       bool   `json:"verified,omitempty" yaml:"verified,omitempty"`
}

// GetAvailableUpdates returns the list of updates that are available for the
// cluster. The information is obtained from the status.availableUpdates field
// of the ClusterVersion resource.
func GetAvailableUpdates(ctx context.Context, k *kubernetes.Kubernetes) ([]Update, error) {
    gvk := schema.GroupVersionKind{Group: "config.openshift.io", Version: "v1", Kind: "ClusterVersion"}

    obj, err := k.ResourcesGet(ctx, &gvk, "", "version")
    if err != nil {
        return nil, fmt.Errorf("cannot retrieve OpenShift ClusterVersion resource: %w", err)
    }

    status, ok := obj.Object["status"].(map[string]interface{})
    if !ok {
        return nil, fmt.Errorf("unexpected ClusterVersion payload: missing status")
    }

    availableUpdatesIface, ok := status["availableUpdates"].([]interface{})
    if !ok {
        // It is valid for a cluster to not have any available updates. In that
        // case return an empty slice instead of an error.
        return []Update{}, nil
    }

    updates := make([]Update, 0, len(availableUpdatesIface))
    for _, entry := range availableUpdatesIface {
        updMap, ok := entry.(map[string]interface{})
        if !ok {
            continue // skip malformed entries
        }
        upd := Update{}
        if v, ok := updMap["version"].(string); ok {
            upd.Version = v
        }
        if img, ok := updMap["image"].(string); ok {
            upd.Image = img
        }
        if upd.Version != "" {
            updates = append(updates, upd)
        }
    }
    return updates, nil
}

// GetCapabilities returns the enabled and known capabilities of the cluster as
// reported by the ClusterVersion operator.
func GetCapabilities(ctx context.Context, k *kubernetes.Kubernetes) (Capabilities, error) {
    gvk := schema.GroupVersionKind{Group: "config.openshift.io", Version: "v1", Kind: "ClusterVersion"}

    obj, err := k.ResourcesGet(ctx, &gvk, "", "version")
    if err != nil {
        return Capabilities{}, fmt.Errorf("cannot retrieve OpenShift ClusterVersion resource: %w", err)
    }

    status, ok := obj.Object["status"].(map[string]interface{})
    if !ok {
        return Capabilities{}, fmt.Errorf("unexpected ClusterVersion payload: missing status")
    }

    capsStatus, ok := status["capabilities"].(map[string]interface{})
    if !ok {
        // The capabilities block may be missing in older cluster versions.
        return Capabilities{}, fmt.Errorf("capabilities not found in ClusterVersion status")
    }

    var caps Capabilities

    if enabled, ok := capsStatus["enabledCapabilities"].([]interface{}); ok {
        caps.Enabled = convertInterfaceSliceToStringSlice(enabled)
    }
    if known, ok := capsStatus["knownCapabilities"].([]interface{}); ok {
        caps.Known = convertInterfaceSliceToStringSlice(known)
    }

    return caps, nil
}

// GetUpdateHistory returns the list describing the history of updates that the
// cluster has gone through, as reported by the ClusterVersion operator. The
// information is obtained from the status.history field.
func GetUpdateHistory(ctx context.Context, k *kubernetes.Kubernetes) ([]UpdateHistory, error) {
    gvk := schema.GroupVersionKind{Group: "config.openshift.io", Version: "v1", Kind: "ClusterVersion"}

    obj, err := k.ResourcesGet(ctx, &gvk, "", "version")
    if err != nil {
        return nil, fmt.Errorf("cannot retrieve OpenShift ClusterVersion resource: %w", err)
    }

    status, ok := obj.Object["status"].(map[string]interface{})
    if !ok {
        return nil, fmt.Errorf("unexpected ClusterVersion payload: missing status")
    }

    historyIface, ok := status["history"].([]interface{})
    if !ok {
        return []UpdateHistory{}, nil // no history found, return empty slice
    }

    history := make([]UpdateHistory, 0, len(historyIface))
    for _, entry := range historyIface {
        hMap, ok := entry.(map[string]interface{})
        if !ok {
            continue
        }
        uh := UpdateHistory{}
        if v, ok := hMap["version"].(string); ok {
            uh.Version = v
        }
        if img, ok := hMap["image"].(string); ok {
            uh.Image = img
        }
        if ct, ok := hMap["completionTime"].(string); ok {
            uh.CompletionTime = ct
        }
        if st, ok := hMap["startedTime"].(string); ok {
            uh.StartedTime = st
        }
        if state, ok := hMap["state"].(string); ok {
            uh.State = state
        }
        if verified, ok := hMap["verified"].(bool); ok {
            uh.Verified = verified
        }
        if uh.Version != "" {
            history = append(history, uh)
        }
    }
    return history, nil
}

// convertInterfaceSliceToStringSlice converts a slice of interface{} to a slice
// of strings ignoring items that are not strings.
func convertInterfaceSliceToStringSlice(input []interface{}) []string {
    ret := make([]string, 0, len(input))
    for _, item := range input {
        if s, ok := item.(string); ok {
            ret = append(ret, s)
        }
    }
    return ret
} 