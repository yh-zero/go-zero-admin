// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package appauth

import (
	"context"

	"go-zero-admin/application/admin/api/internal/svc"
	"go-zero-admin/application/admin/api/internal/types"
	"go-zero-admin/application/app/rpc/apppb"
	"go-zero-admin/pkg/xerr"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListMyFeedbackLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewListMyFeedbackLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListMyFeedbackLogic {
	return &ListMyFeedbackLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ListMyFeedbackLogic) ListMyFeedback(req *types.AppContentListReq) (resp *types.AppFeedbackListResp, err error) {
	if l.svcCtx.AppAuthRpc == nil {
		return nil, xerr.NewErrCode(xerr.SERVICE_UNAVAILABLE)
	}
	r, err := l.svcCtx.AppAuthRpc.ListMyFeedback(l.ctx, &apppb.AppContentListRequest{
		AccessToken: appToken(l.ctx), Page: &apppb.PageRequest{Page: req.Page, PageSize: req.PageSize},
	})
	if err != nil {
		return nil, err
	}
	list := make([]types.AppFeedbackInfo, 0, len(r.List))
	for _, row := range r.List {
		list = append(list, MapAppFeedback(row))
	}
	return &types.AppFeedbackListResp{List: list, Total: r.GetPage().GetTotal()}, nil
}

func MapAppFeedback(row *apppb.AppFeedbackInfo) types.AppFeedbackInfo {
	return types.AppFeedbackInfo{
		Id: row.GetId(), UserId: row.GetUserId(), Account: row.GetAccount(), Nickname: row.GetNickname(),
		Content: row.GetContent(), Reply: row.GetReply(), Status: row.GetStatus(),
		CreatedAt: row.GetCreatedAt(), UpdatedAt: row.GetUpdatedAt(),
	}
}
