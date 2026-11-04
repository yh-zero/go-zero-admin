package appauth

import (
	"context"
	"github.com/zeromicro/go-zero/core/logx"
	"go-zero-admin/application/admin/api/internal/svc"
	"go-zero-admin/application/admin/api/internal/types"
	"go-zero-admin/application/app/rpc/apppb"
	"go-zero-admin/pkg/xerr"
)

type SendEmailCodeLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewSendEmailCodeLogic(ctx context.Context, s *svc.ServiceContext) *SendEmailCodeLogic {
	return &SendEmailCodeLogic{ctx, s, logx.WithContext(ctx)}
}
func (l *SendEmailCodeLogic) SendEmailCode(req *types.AppSendEmailCodeReq) (*types.EmptyResp, error) {
	if err := l.svcCtx.Validator.StructCtx(l.ctx, req); err != nil {
		return nil, xerr.NewErrCode(xerr.REQUEST_PARAM_ERROR)
	}
	if l.svcCtx.AppAuthRpc == nil {
		return nil, xerr.NewErrCode(xerr.SERVICE_UNAVAILABLE)
	}

	ip, _ := appRequestMeta(l.ctx)
	_, err := l.svcCtx.AppAuthRpc.SendEmailCode(l.ctx, &apppb.AppSendEmailCodeRequest{Account: req.Account, IP: ip})
	if err != nil {
		return nil, err
	}
	return &types.EmptyResp{}, nil
}
