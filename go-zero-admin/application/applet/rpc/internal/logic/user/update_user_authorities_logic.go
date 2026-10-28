package userlogic

import (
	"context"
	"github.com/zeromicro/go-zero/core/logx"
	"go-zero-admin/application/applet/rpc/internal/svc"
	"go-zero-admin/application/applet/rpc/pb"
)

type UpdateUserAuthoritiesLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewUpdateUserAuthoritiesLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateUserAuthoritiesLogic {
	return &UpdateUserAuthoritiesLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

// 兼容旧RPC入口，同样复用事务更新；不再隐式把第一个角色设为默认角色。
func (l *UpdateUserAuthoritiesLogic) UpdateUserAuthorities(in *pb.UpdateUserAuthoritiesRequest) (*pb.NoDataResponse, error) {
	return NewUpdateUserInfoLogic(l.ctx, l.svcCtx).UpdateUserInfo(&pb.UpdateUserInfoRequest{UserInfo: &pb.UserInfo{ID: in.ID}, AuthorityIds: in.AuthorityIds, UpdateAuthorities: true})
}
