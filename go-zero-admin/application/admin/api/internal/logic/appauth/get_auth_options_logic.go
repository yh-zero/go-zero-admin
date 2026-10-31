package appauth

import (
	"context"
	"github.com/zeromicro/go-zero/core/logx"
	"go-zero-admin/application/admin/api/internal/svc"
	"go-zero-admin/application/admin/api/internal/types"
	"go-zero-admin/application/app/rpc/apppb"
	"go-zero-admin/pkg/xerr"
)

type GetAuthOptionsLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetAuthOptionsLogic(ctx context.Context, s *svc.ServiceContext) *GetAuthOptionsLogic {
	return &GetAuthOptionsLogic{ctx, s, logx.WithContext(ctx)}
}
func (l *GetAuthOptionsLogic) GetAuthOptions() (*types.AppAuthOptionsResp, error) {
	if l.svcCtx.AppAuthRpc == nil {
		return nil, xerr.NewErrCode(xerr.SERVICE_UNAVAILABLE)
	}

	r, err := l.svcCtx.AppAuthRpc.GetAuthOptions(l.ctx, &apppb.AppAuthOptionsRequest{})
	if err != nil {
		return nil, err
	}
	return &types.AppAuthOptionsResp{EmailVerificationRequired: r.EmailVerificationRequired, EmailCodeEnabled: r.EmailCodeEnabled, PhoneEnabled: r.PhoneEnabled}, nil
}
