package agentlogic

import (
	"context"
	"encoding/json"
	"errors"
	"unicode/utf8"

	"go-zero-admin/application/ai/rpc/internal/svc"
	"go-zero-admin/application/ai/rpc/pb"
	appletpb "go-zero-admin/application/applet/rpc/pb"
	"go-zero-admin/pkg/agenttools"
	"go-zero-admin/pkg/aiagent"
)

func toolPermission(ctx context.Context, s *svc.ServiceContext, a *pb.SessionRequest, spec agenttools.Spec) error {
	if err := authorize(ctx, s, a, "/v1/ai/runs", "POST"); err != nil {
		return err
	}
	if spec.SessionOnly {
		return nil
	}
	return enforce(ctx, s, a, spec.Path, "GET")
}

func availableTools(ctx context.Context, s *svc.ServiceContext, a *pb.SessionRequest) ([]aiagent.Tool, error) {
	tools := make([]aiagent.Tool, 0, len(agenttools.Specs))
	for _, spec := range agenttools.Specs {
		if err := toolPermission(ctx, s, a, spec); err != nil {
			if errors.Is(err, errPermissionDenied) {
				continue
			}
			return nil, err
		}
		spec := spec
		tools = append(tools, aiagent.Tool{ToolDefinition: aiagent.ToolDefinition{Name: spec.Name, Description: spec.Description, Parameters: json.RawMessage(spec.Schema)}, Call: func(ctx context.Context, arguments json.RawMessage) (aiagent.ToolOutput, error) {
			if err := toolPermission(ctx, s, a, spec); err != nil {
				return aiagent.ToolOutput{}, err
			}
			var object map[string]json.RawMessage
			if len(arguments) > 4096 || !utf8.Valid(arguments) || json.Unmarshal(arguments, &object) != nil || object == nil {
				return aiagent.ToolOutput{}, aiagent.ErrToolArguments
			}
			if s.AppletAgentToolsRPC == nil {
				return aiagent.ToolOutput{}, aiagent.ErrToolFailed
			}
			request := &appletpb.AgentToolRequest{Actor: businessActor(a), ArgumentsJson: string(arguments)}
			var result *appletpb.AgentToolResult
			var err error
			switch spec.Name {
			case "query_audit_logs":
				result, err = s.AppletAgentToolsRPC.QueryAgentAudit(ctx, request)
			case "get_file_status":
				result, err = s.AppletAgentToolsRPC.GetAgentFileStatus(ctx, request)
			case "list_my_devices":
				result, err = s.AppletAgentToolsRPC.ListAgentDevices(ctx, request)
			default:
				return aiagent.ToolOutput{}, aiagent.ErrUnknownTool
			}
			if err != nil {
				if ctx.Err() != nil {
					return aiagent.ToolOutput{}, ctx.Err()
				}
				return aiagent.ToolOutput{}, aiagent.ErrToolFailed
			}
			if err := toolPermission(ctx, s, a, spec); err != nil {
				return aiagent.ToolOutput{}, err
			}
			maxBytes := s.AgentConfig.MaxToolOutputBytes
			if maxBytes == 0 {
				maxBytes = 16384
			}
			if result == nil || len(result.Content) > maxBytes || len(result.Summary) > 1024 || result.Count < 0 || result.Count > 1000000 || !utf8.ValidString(result.Content) || !utf8.ValidString(result.Summary) || !json.Valid([]byte(result.Content)) {
				return aiagent.ToolOutput{}, aiagent.ErrBudgetExceeded
			}
			return aiagent.ToolOutput{Content: result.Content, Summary: result.Summary, Count: int(result.Count)}, nil
		}})
	}
	return tools, nil
}
