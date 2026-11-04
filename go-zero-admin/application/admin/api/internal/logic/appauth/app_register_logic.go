package appauth

import (
	"context"
	"github.com/zeromicro/go-zero/core/logx"
	"go-zero-admin/application/admin/api/internal/svc"
	"go-zero-admin/application/admin/api/internal/types"
	"go-zero-admin/application/app/rpc/apppb"
	"go-zero-admin/pkg/xerr"
)

type AppRegisterLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewAppRegisterLogic(ctx context.Context, s *svc.ServiceContext) *AppRegisterLogic {
	return &AppRegisterLogic{ctx, s, logx.WithContext(ctx)}
}
func (l *AppRegisterLogic) AppRegister(req *types.AppRegisterReq) (*types.EmptyResp, error) {
	if err := l.svcCtx.Validator.StructCtx(l.ctx, req); err != nil {
		return nil, xerr.NewErrCode(xerr.REQUEST_PARAM_ERROR)
	}
	if l.svcCtx.AppAuthRpc == nil {
		return nil, xerr.NewErrCode(xerr.SERVICE_UNAVAILABLE)
	}

	ip, ua := appRequestMeta(l.ctx)
	_, err := l.svcCtx.AppAuthRpc.AppRegister(l.ctx, &apppb.AppRegisterRequest{Account: req.Account, Password: req.Password, Nickname: req.Nickname, EmailCode: req.EmailCode, Ip: ip, UserAgent: ua})
	if err != nil {
		return nil, err
	}
	return &types.EmptyResp{}, nil
}
