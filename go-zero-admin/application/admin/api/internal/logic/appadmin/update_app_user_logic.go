// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package appadmin

import (
	"context"

	"go-zero-admin/application/admin/api/internal/svc"
	"go-zero-admin/application/admin/api/internal/types"
	"go-zero-admin/application/app/rpc/apppb"
	"go-zero-admin/pkg/xerr"

	"github.com/zeromicro/go-zero/core/logx"
)

type UpdateAppUserLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewUpdateAppUserLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateAppUserLogic {
	return &UpdateAppUserLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *UpdateAppUserLogic) UpdateAppUser(req *types.AppUpdateUserReq) (resp *types.EmptyResp, err error) {
	if err := l.svcCtx.Validator.StructCtx(l.ctx, req); err != nil {
		return nil, xerr.NewErrCode(xerr.REQUEST_PARAM_ERROR)
	}
	if l.svcCtx.AppAdminRpc == nil {
		return nil, xerr.NewErrCode(xerr.SERVICE_UNAVAILABLE)
	}
	_, err = l.svcCtx.AppAdminRpc.UpdateAppUser(l.ctx, &apppb.AppUpdateUserRequest{
		Actor: appAdminActor(l.ctx), UserId: req.Id, Nickname: req.Nickname, Remark: req.Remark,
	})
	if err != nil {
		return nil, err
	}
	return &types.EmptyResp{}, nil
}
