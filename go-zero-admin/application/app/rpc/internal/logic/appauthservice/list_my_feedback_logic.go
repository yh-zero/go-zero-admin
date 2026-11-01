package appauthservicelogic

import (
	"context"

	"go-zero-admin/application/app/rpc/apppb"
	"go-zero-admin/application/app/rpc/internal/svc"
	"go-zero-admin/pkg/xerr"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListMyFeedbackLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewListMyFeedbackLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListMyFeedbackLogic {
	return &ListMyFeedbackLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *ListMyFeedbackLogic) ListMyFeedback(in *apppb.AppContentListRequest) (*apppb.AppFeedbackListResponse, error) {
	user, _, err := CurrentAppUser(l.ctx, l.svcCtx, in.AccessToken)
	if err != nil {
		return nil, err
	}
	page, size := in.GetPage().GetPage(), in.GetPage().GetPageSize()
	if page <= 0 {
		page = 1
	}
	if size <= 0 {
		size = 20
	}
	if size > 100 {
		size = 100
	}
	if page > 1000000 {
		return nil, xerr.NewErrCode(xerr.REQUEST_PARAM_ERROR)
	}
	rows, total, err := l.svcCtx.AppFeedbackModel.ListPage(l.ctx, int64(user.Id), "", page, size)
	if err != nil {
		return nil, xerr.NewErrCode(xerr.DB_ERROR)
	}
	list := make([]*apppb.AppFeedbackInfo, 0, len(rows))
	for _, r := range rows {
		list = append(list, &apppb.AppFeedbackInfo{Id: r.Id, UserId: r.UserId, Account: r.Account, Nickname: r.Nickname, Content: r.Content, Reply: r.Reply, Status: r.Status, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt})
	}
	return &apppb.AppFeedbackListResponse{List: list, Page: &apppb.PageResponse{Total: total}}, nil
}
