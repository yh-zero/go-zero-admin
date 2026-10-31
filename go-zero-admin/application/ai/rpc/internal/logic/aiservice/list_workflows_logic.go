package aiservicelogic

import (
	"context"

	"github.com/zeromicro/go-zero/core/logx"
	"go-zero-admin/application/ai/rpc/aipb"
	"go-zero-admin/application/ai/rpc/internal/svc"
)

type ListWorkflowsLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewListWorkflowsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListWorkflowsLogic {
	return &ListWorkflowsLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}
func (l *ListWorkflowsLogic) ListWorkflows(in *aipb.ListWorkflowsRequest) (*aipb.ListWorkflowsResponse, error) {
	if err := requireAiActor(in.Actor); err != nil {
		return nil, err
	}
	page, size, err := aiPagination(in.Page)
	if err != nil {
		return nil, err
	}
	rows, total, err := l.svcCtx.AiWorkflowModel.ListOwned(l.ctx, in.Actor.UserId, page, size)
	if err != nil {
		return nil, aiModelError(err)
	}
	list := make([]*aipb.WorkflowInfo, 0, len(rows))
	for _, row := range rows {
		list = append(list, workflowProto(row))
	}
	return &aipb.ListWorkflowsResponse{List: list, Page: &aipb.PageResponse{Total: total}}, nil
}
