package appadmin

import (
	"context"
	"github.com/zeromicro/go-zero/core/logx"
	"go-zero-admin/application/admin/api/internal/svc"
	"go-zero-admin/application/admin/api/internal/types"
	"go-zero-admin/application/app/rpc/apppb"
	"go-zero-admin/pkg/xerr"
)

type ReviewAppUserLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewReviewAppUserLogic(ctx context.Context, s *svc.ServiceContext) *ReviewAppUserLogic {
	return &ReviewAppUserLogic{ctx, s, logx.WithContext(ctx)}
}
func (l *ReviewAppUserLogic) ReviewAppUser(req *types.AppReviewReq) (*types.EmptyResp, error) {
	if l.svcCtx.AppAdminRpc == nil {
		return nil, xerr.NewErrCode(xerr.SERVICE_UNAVAILABLE)
	}

	_, err := l.svcCtx.AppAdminRpc.ReviewAppUser(l.ctx, &apppb.AppReviewRequest{Actor: appAdminActor(l.ctx), UserID: req.Id, Status: req.Status, Reason: req.Reason})
	if err != nil {
		return nil, err
	}
	return &types.EmptyResp{}, nil
}
