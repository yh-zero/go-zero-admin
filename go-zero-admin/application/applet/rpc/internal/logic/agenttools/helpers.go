package agenttoolslogic

import (
	"context"
	"errors"
	"strconv"
	"time"

	casbinlogic "go-zero-admin/application/applet/rpc/internal/logic/casbin"
	userlogic "go-zero-admin/application/applet/rpc/internal/logic/user"
	"go-zero-admin/application/applet/rpc/internal/svc"
	"go-zero-admin/application/applet/rpc/pb"
	"go-zero-admin/pkg/agenttools"
	"go-zero-admin/pkg/result/xerr"
)

func session(ctx context.Context, s *svc.ServiceContext, actor *pb.SessionRequest) error {
	if actor == nil || actor.UserID <= 0 || actor.AuthorityId <= 0 || actor.SessionVersion <= 0 {
		return xerr.NewErrCode(xerr.TOKEN_EXPIRE_ERROR)
	}
	if s == nil || s.DB == nil {
		return errors.New("business session service unavailable")
	}
	response, err := userlogic.NewCheckSessionLogic(ctx, s).CheckSession(actor)
	if err != nil {
		return err
	}
	if !response.Valid {
		return xerr.NewErrCode(xerr.TOKEN_EXPIRE_ERROR)
	}
	return nil
}

// enforcePermission requires a session validated by the current authorization
// phase. executeTool runs the phase both before and after the business read.
func enforcePermission(ctx context.Context, s *svc.ServiceContext, actor *pb.SessionRequest, path, method string) error {
	if s.Casbin == nil {
		return errors.New("business permission service unavailable")
	}
	response, err := casbinlogic.NewEnforceLogic(ctx, s).Enforce(&pb.EnforceRequest{AuthorityId: strconv.FormatInt(actor.AuthorityId, 10), Path: path, Method: method})
	if err != nil {
		return err
	}
	if !response.Pass {
		return xerr.NewErrCodeMsg(xerr.REUQEST_PARAM_ERROR, "无权使用此AI能力")
	}
	return nil
}

func toolPermission(ctx context.Context, s *svc.ServiceContext, actor *pb.SessionRequest, spec agenttools.Spec) error {
	if err := session(ctx, s, actor); err != nil {
		return err
	}
	if err := enforcePermission(ctx, s, actor, "/v1/ai/runs", "POST"); err != nil {
		return err
	}
	if spec.SessionOnly {
		return nil
	}
	return enforcePermission(ctx, s, actor, spec.Path, "GET")
}

func executeTool(ctx context.Context, s *svc.ServiceContext, request *pb.AgentToolRequest, spec agenttools.Spec, call func() (agenttools.Output, error)) (*pb.AgentToolResult, error) {
	if request == nil {
		return nil, xerr.NewErrCode(xerr.REUQEST_PARAM_ERROR)
	}
	if err := toolPermission(ctx, s, request.Actor, spec); err != nil {
		return nil, err
	}
	result, err := call()
	if err != nil {
		if errors.Is(err, agenttools.ErrArguments) {
			return nil, xerr.NewErrCodeMsg(xerr.REUQEST_PARAM_ERROR, "AI工具参数无效")
		}
		return nil, err
	}
	// The model must not receive data if login or grants were revoked during SQL.
	if err := toolPermission(ctx, s, request.Actor, spec); err != nil {
		return nil, err
	}
	return &pb.AgentToolResult{Content: result.Content, Summary: result.Summary, Count: int64(result.Count)}, nil
}

func timestamp(value time.Time) string { return value.UTC().Format(time.RFC3339Nano) }
