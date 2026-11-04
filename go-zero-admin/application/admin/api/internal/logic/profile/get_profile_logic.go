package profile

import (
	"context"
	"go-zero-admin/pkg/xerr"

	"go-zero-admin/application/admin/api/internal/middleware"
	"go-zero-admin/application/admin/api/internal/svc"
	"go-zero-admin/application/admin/api/internal/types"
	"go-zero-admin/application/admin/rpc/adminpb"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetProfileLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetProfileLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetProfileLogic {
	return &GetProfileLogic{Logger: logx.WithContext(ctx), ctx: ctx, svcCtx: svcCtx}
}

func (l *GetProfileLogic) GetProfile(req *types.GetProfileReq) (resp *types.GetProfileResp, err error) {
	if err := l.svcCtx.Validator.StructCtx(l.ctx, req); err != nil {
		return nil, xerr.NewErrCode(xerr.REQUEST_PARAM_ERROR)
	}
	actor := middleware.ActorFromContext(l.ctx)
	rpcResp, err := l.svcCtx.ProfileRpc.GetProfile(l.ctx, &adminpb.GetProfileRequest{Actor: actor})
	if err != nil {
		return nil, err
	}
	u := rpcResp.User
	return &types.GetProfileResp{User: &types.ProfileInfo{
		Id: u.Id, Username: u.Username, Nickname: u.Nickname, RealName: u.RealName,
		Email: u.Email, Phone: u.Phone, AvatarUrl: u.AvatarUrl, Gender: u.Gender, DeptName: u.DeptName,
		AvatarFileId: u.AvatarFileId, IsSuperAdmin: u.IsSuperAdmin,
	}}, nil
}
