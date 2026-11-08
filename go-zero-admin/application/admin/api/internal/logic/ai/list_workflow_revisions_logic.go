// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package ai

import (
	"context"

	"go-zero-admin/application/admin/api/internal/svc"
	"go-zero-admin/application/admin/api/internal/types"
	"go-zero-admin/application/ai/rpc/aipb"
	"go-zero-admin/pkg/xerr"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListWorkflowRevisionsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewListWorkflowRevisionsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListWorkflowRevisionsLogic {
	return &ListWorkflowRevisionsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ListWorkflowRevisionsLogic) ListWorkflowRevisions(req *types.AiListWorkflowRevisionsReq) (resp *types.AiListWorkflowRevisionsResp, err error) {
	if err := l.svcCtx.Validator.StructCtx(l.ctx, req); err != nil {
		return nil, xerr.NewErrCode(xerr.REQUEST_PARAM_ERROR)
	}
	actor, err := actorForAi(l.ctx, l.svcCtx)
	if err != nil {
		return nil, err
	}
	out, err := l.svcCtx.AiRpc.ListWorkflowRevisions(l.ctx, &aipb.ListWorkflowRevisionsRequest{Actor: actor, WorkflowId: req.Id, Page: &aipb.PageRequest{Page: req.Page, PageSize: req.PageSize}})
	if err != nil {
		return nil, err
	}
	list := make([]*types.AiWorkflowRevisionInfo, 0, len(out.List))
	for _, row := range out.List {
		list = append(list, mapAiWorkflowRevision(row))
	}
	return &types.AiListWorkflowRevisionsResp{List: list, Total: out.GetPage().GetTotal()}, nil
}
