package organizationlogic

import (
	"context"

	"go-zero-admin/application/applet/rpc/internal/svc"
	"go-zero-admin/application/applet/rpc/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetMembershipLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetMembershipLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetMembershipLogic {
	return &GetMembershipLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *GetMembershipLogic) GetMembership(in *pb.OrganizationIDRequest) (*pb.MembershipResponse, error) {
	return (store{ctx: l.ctx, db: l.svcCtx.DB.DB}).membership(in)
}
