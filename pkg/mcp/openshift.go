package mcp

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	openshiftupdates "github.com/manusa/kubernetes-mcp-server/pkg/openshift/updates"
	"github.com/manusa/kubernetes-mcp-server/pkg/output"
	"github.com/redhat-developer/kubernetes-mcp/pkg/kubernetes"
)

// initOpenShift returns OpenShift-specific tools. If the connected cluster is not
// an OpenShift cluster it returns an empty slice so that the caller can safely
// concatenate the result without additional checks.
func (s *Server) initOpenShift() []server.ServerTool {
	if !s.k.IsOpenShift(context.Background()) {
		return nil
	}

	return []server.ServerTool{
		{Tool: mcp.NewTool("cluster_version_get",
			mcp.WithDescription("Get the OpenShift cluster version (for example '4.15.9')"),
			// Tool annotations
			mcp.WithTitleAnnotation("OpenShift: Cluster Version"),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithOpenWorldHintAnnotation(true),
		), Handler: s.openShiftClusterVersion},
		{Tool: mcp.NewTool("cluster_available_updates_list",
			mcp.WithDescription("List the OpenShift cluster available updates (version and image)"),
			// Tool annotations
			mcp.WithTitleAnnotation("OpenShift: Available Updates"),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithOpenWorldHintAnnotation(true),
		), Handler: s.openShiftAvailableUpdates},
		{Tool: mcp.NewTool("cluster_capabilities_get",
			mcp.WithDescription("Get the OpenShift cluster capabilities (enabled or known)"),
			mcp.WithString("type", mcp.Description("Type of capabilities to retrieve ('enabled' or 'known'). If not provided, both are returned")),
			// Tool annotations
			mcp.WithTitleAnnotation("OpenShift: Capabilities"),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithOpenWorldHintAnnotation(true),
		), Handler: s.openShiftCapabilities},
		{Tool: mcp.NewTool("cluster_update_history_list",
			mcp.WithDescription("List the OpenShift cluster update history"),
			// Tool annotations
			mcp.WithTitleAnnotation("OpenShift: Update History"),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithOpenWorldHintAnnotation(true),
		), Handler: s.openShiftUpdateHistory},
	}
}

func (s *Server) initOcCli() []server.ServerTool {
	return []server.ServerTool{
		{
			Tool: mcp.NewTool("oc_cli_exec",
				mcp.WithDescription("Execute an oc command")).
				WithParameter(mcp.NewParameter("command",
					"Command and its arguments as a list of strings, e.g., [\"get\", \"pods\"]").WithType(mcp.ParameterTypeArray)).
				WithTitleAnnotation("OpenShift: Execute OC Command").
				WithReadOnlyHintAnnotation(false).
				WithDestructiveHintAnnotation(true).
				WithOpenWorldHintAnnotation(true),
			Handler: s.ocCliExec,
		},
	}
}

func (s *Server) openShiftClusterVersion(ctx context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	version, err := openshiftupdates.GetClusterVersion(ctx, s.k.Derived(ctx))
	if err != nil {
		return NewTextResult("", fmt.Errorf("failed to get OpenShift cluster version: %v", err)), nil
	}
	return NewTextResult(version, nil), nil
}

func (s *Server) openShiftAvailableUpdates(ctx context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	updates, err := openshiftupdates.GetAvailableUpdates(ctx, s.k.Derived(ctx))
	if err != nil {
		return NewTextResult("", fmt.Errorf("failed to get OpenShift available updates: %v", err)), nil
	}
	yamlOut, err := output.MarshalYaml(updates)
	if err != nil {
		return NewTextResult("", fmt.Errorf("failed to marshal available updates: %v", err)), nil
	}
	return NewTextResult(fmt.Sprintf("Available updates (YAML format):\n%s", yamlOut), nil), nil
}

func (s *Server) openShiftCapabilities(ctx context.Context, ctr mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	caps, err := openshiftupdates.GetCapabilities(ctx, s.k.Derived(ctx))
	if err != nil {
		return NewTextResult("", fmt.Errorf("failed to get OpenShift capabilities: %v", err)), nil
	}
	capTypeArg := ""
	if v := ctr.GetArguments()["type"]; v != nil {
		capTypeArg, _ = v.(string)
	}
	var out interface{}
	switch capTypeArg {
	case "enabled":
		out = caps.Enabled
	case "known":
		out = caps.Known
	default:
		out = caps
	}
	yamlOut, err := output.MarshalYaml(out)
	if err != nil {
		return NewTextResult("", fmt.Errorf("failed to marshal capabilities: %v", err)), nil
	}
	return NewTextResult(fmt.Sprintf("Cluster capabilities (YAML format):\n%s", yamlOut), nil), nil
}

func (s *Server) openShiftUpdateHistory(ctx context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	history, err := openshiftupdates.GetUpdateHistory(ctx, s.k.Derived(ctx))
	if err != nil {
		return NewTextResult("", fmt.Errorf("failed to get OpenShift update history: %v", err)), nil
	}
	yamlOut, err := output.MarshalYaml(history)
	if err != nil {
		return NewTextResult("", fmt.Errorf("failed to marshal update history: %v", err)), nil
	}
	return NewTextResult(fmt.Sprintf("Cluster update history (YAML format):\n%s", yamlOut), nil), nil
}

func (s *Server) ocCliExec(ctx context.Context, ctr mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var args []string
	err := ctr.Parameters.Bind("command", &args)
	if err != nil {
		return modelcontext.ErrorToolResult(err.Error()), nil
	}

	output, err := s.kube.ExecuteOcCommand(ctx, args...)
	if err != nil {
		return modelcontext.ErrorToolResult(output + "\n" + err.Error()), nil
	}

	return modelcontext.TextToolResult(output), nil
}
