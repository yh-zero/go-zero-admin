package organizationlogic

import (
	"context"

	"go-zero-admin/application/applet/rpc/internal/svc"
	"go-zero-admin/application/applet/rpc/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

type UpdateMembershipLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewUpdateMembershipLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateMembershipLogic {
	return &UpdateMembershipLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *UpdateMembershipLogic) UpdateMembership(in *pb.MembershipRequest) (*pb.NoDataResponse, error) {
	return &pb.NoDataResponse{}, (store{ctx: l.ctx, db: l.svcCtx.DB.DB}).updateMembership(in)
}
