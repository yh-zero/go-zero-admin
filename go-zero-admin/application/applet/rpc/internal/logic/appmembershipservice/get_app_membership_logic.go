package appmembershipservicelogic

import (
	"context"
	appaccount "go-zero-admin/application/applet/rpc/internal/logic/appaccountservice"

	"go-zero-admin/application/applet/rpc/internal/svc"
	"go-zero-admin/application/applet/rpc/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetAppMembershipLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetAppMembershipLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetAppMembershipLogic {
	return &GetAppMembershipLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *GetAppMembershipLogic) GetAppMembership(in *pb.AppAccessRequest) (*pb.AppMembershipInfo, error) {
	if in == nil {
		return nil, membershipError("请求参数不能为空")
	}
	user, _, err := appaccount.ValidateAccess(l.ctx, l.svcCtx, in.AccessToken)
	if err != nil {
		return nil, err
	}
	return membershipAt(l.svcCtx.DB.WithContext(l.ctx), user.ID, membershipNow())
}
