// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package organization

import (
	"context"

	"go-zero-admin/application/applet/api/internal/svc"
	"go-zero-admin/application/applet/api/internal/types"
	"go-zero-admin/application/applet/rpc/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

type UpdateMembershipLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 设置人员部门岗位归属
func NewUpdateMembershipLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateMembershipLogic {
	return &UpdateMembershipLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *UpdateMembershipLogic) UpdateMembership(req *types.Membership) (resp *types.MessageResponse, err error) {
	_, err = l.svcCtx.AppletOrganizationRPC.UpdateMembership(l.ctx, &pb.MembershipRequest{UserID: req.UserId, DepartmentId: req.DepartmentId, PositionIds: req.PositionIds})
	if err != nil {
		return nil, err
	}
	return &types.MessageResponse{Message: "更新成功"}, nil
}
