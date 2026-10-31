package user

import (
	"context"
	"go-zero-admin/application/admin/api/internal/middleware"

	"go-zero-admin/application/admin/api/internal/svc"
	"go-zero-admin/application/admin/api/internal/types"
	adminpb "go-zero-admin/application/admin/rpc/adminpb"

	"github.com/zeromicro/go-zero/core/logx"
)

type UpdateUserLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewUpdateUserLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateUserLogic {
	return &UpdateUserLogic{Logger: logx.WithContext(ctx), ctx: ctx, svcCtx: svcCtx}
}

func (l *UpdateUserLogic) UpdateUser(req *types.UpdateUserReq) (*types.EmptyResp, error) {
	_, err := l.svcCtx.UserRpc.UpdateUser(l.ctx, &adminpb.UpdateUserRequest{
		Actor: middleware.ActorFromContext(l.ctx), RoleIds: req.RoleIds, UpdateRoles: req.RoleIds != nil,
		UserId: req.Id, Nickname: req.Nickname,
		RealName: req.RealName, Email: req.Email, Phone: req.Phone,
		DeptId: req.DeptId, AvatarFileId: req.AvatarFileId,
		Gender: req.Gender, Status: req.Status, Remark: req.Remark,
	})
	if err != nil {
		return nil, err
	}
	return &types.EmptyResp{}, nil
}
