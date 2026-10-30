package user

import (
	"context"

	"go-zero-admin/application/admin/api/internal/svc"
	"go-zero-admin/application/admin/api/internal/types"
	adminpb "go-zero-admin/application/admin/rpc/adminpb"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetUserLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetUserLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetUserLogic {
	return &GetUserLogic{Logger: logx.WithContext(ctx), ctx: ctx, svcCtx: svcCtx}
}

func (l *GetUserLogic) GetUser(req *types.GetUserReq) (resp *types.GetUserResp, err error) {
	rpcResp, err := l.svcCtx.UserRpc.GetUser(l.ctx, &adminpb.GetUserRequest{UserId: req.Id})
	if err != nil {
		return nil, err
	}
	u := rpcResp.User
	return &types.GetUserResp{User: &types.UserInfo{
		Id: u.Id, Username: u.Username, Nickname: u.Nickname,
		RealName: u.RealName, Email: u.Email, Phone: u.Phone,
		DeptId: u.DeptId, DeptName: u.DeptName,
		AvatarFileId: u.AvatarFileId, Gender: u.Gender, Status: u.Status,
		Remark: u.Remark, LastLoginAt: u.LastLoginAt, LastLoginIp: u.LastLoginIP,
		CreatedAt: u.CreatedAt, UpdatedAt: u.UpdatedAt,
	}}, nil
}
