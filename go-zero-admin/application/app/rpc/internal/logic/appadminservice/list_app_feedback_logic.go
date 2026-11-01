package appadminservicelogic

import (
	"context"

	"go-zero-admin/application/app/rpc/apppb"
	"go-zero-admin/application/app/rpc/internal/svc"
	"go-zero-admin/pkg/xerr"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListAppFeedbackLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewListAppFeedbackLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListAppFeedbackLogic {
	return &ListAppFeedbackLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *ListAppFeedbackLogic) ListAppFeedback(in *apppb.AppFeedbackListRequest) (*apppb.AppFeedbackListResponse, error) {
	if err := requireActor(in.Actor); err != nil {
		return nil, err
	}
	page, size, err := pagination(in.Page)
	if err != nil {
		return nil, err
	}
	if in.UserId < 0 || in.Status != "" && in.Status != "pending" && in.Status != "resolved" {
		return nil, xerr.NewErrCode(xerr.REQUEST_PARAM_ERROR)
	}
	rows, total, err := l.svcCtx.AppFeedbackModel.ListPage(l.ctx, in.UserId, in.Status, page, size)
	if err != nil {
		return nil, xerr.NewErrCode(xerr.DB_ERROR)
	}
	list := make([]*apppb.AppFeedbackInfo, 0, len(rows))
	for _, r := range rows {
		list = append(list, &apppb.AppFeedbackInfo{Id: r.Id, UserId: r.UserId, Account: r.Account, Nickname: r.Nickname, Content: r.Content, Reply: r.Reply, Status: r.Status, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt})
	}
	return &apppb.AppFeedbackListResponse{List: list, Page: &apppb.PageResponse{Total: total}}, nil
}
