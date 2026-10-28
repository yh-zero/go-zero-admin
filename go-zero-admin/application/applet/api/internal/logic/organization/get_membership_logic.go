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

type GetMembershipLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 读取人员部门岗位归属
func NewGetMembershipLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetMembershipLogic {
	return &GetMembershipLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetMembershipLogic) GetMembership(req *types.OrganizationUserRequest) (resp *types.Membership, err error) {
	result, err := l.svcCtx.AppletOrganizationRPC.GetMembership(l.ctx, &pb.OrganizationIDRequest{ID: req.UserId})
	if err != nil {
		return nil, err
	}
	return &types.Membership{UserId: result.UserID, DepartmentId: result.DepartmentId, PositionIds: append([]int64{}, result.PositionIds...)}, nil
}
