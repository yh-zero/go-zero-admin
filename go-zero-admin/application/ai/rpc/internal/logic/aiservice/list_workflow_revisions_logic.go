package aiservicelogic

import (
	"context"

	"go-zero-admin/application/ai/rpc/aipb"
	"go-zero-admin/application/ai/rpc/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListWorkflowRevisionsLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewListWorkflowRevisionsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListWorkflowRevisionsLogic {
	return &ListWorkflowRevisionsLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *ListWorkflowRevisionsLogic) ListWorkflowRevisions(in *aipb.ListWorkflowRevisionsRequest) (*aipb.ListWorkflowRevisionsResponse, error) {
	if err := requireAiActor(in.Actor); err != nil {
		return nil, err
	}
	page, size, err := aiPagination(in.Page)
	if err != nil {
		return nil, err
	}
	rows, total, err := l.svcCtx.AiWorkflowModel.ListRevisionsOwned(l.ctx, in.WorkflowId, in.Actor.UserId, page, size)
	if err != nil {
		return nil, workflowModelError(err)
	}
	list := make([]*aipb.WorkflowRevisionInfo, 0, len(rows))
	for _, row := range rows {
		list = append(list, workflowRevisionProto(row))
	}
	return &aipb.ListWorkflowRevisionsResponse{List: list, Page: &aipb.PageResponse{Total: total}}, nil
}
