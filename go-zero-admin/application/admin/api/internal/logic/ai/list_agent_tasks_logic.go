package ai

import (
	"context"
	"go-zero-admin/pkg/xerr"

	"github.com/zeromicro/go-zero/core/logx"
	"go-zero-admin/application/admin/api/internal/svc"
	"go-zero-admin/application/admin/api/internal/types"
	"go-zero-admin/application/ai/rpc/aipb"
)

type ListAgentTasksLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewListAgentTasksLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListAgentTasksLogic {
	return &ListAgentTasksLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}
func (l *ListAgentTasksLogic) ListAgentTasks(req *types.AiListTasksReq) (*types.AiListTasksResp, error) {
	if err := l.svcCtx.Validator.StructCtx(l.ctx, req); err != nil {
		return nil, xerr.NewErrCode(xerr.REQUEST_PARAM_ERROR)
	}
	actor, err := actorForAi(l.ctx, l.svcCtx)
	if err != nil {
		return nil, err
	}
	out, err := l.svcCtx.AiRpc.ListAgentTasks(l.ctx, &aipb.ListAgentTasksRequest{Actor: actor, Page: &aipb.PageRequest{Page: req.Page, PageSize: req.PageSize}, Status: req.Status, WorkflowId: req.WorkflowId})
	if err != nil {
		return nil, err
	}
	list := make([]*types.AiTaskInfo, 0, len(out.List))
	for _, item := range out.List {
		list = append(list, mapAiTask(item))
	}
	return &types.AiListTasksResp{List: list, Total: out.GetPage().GetTotal()}, nil
}
