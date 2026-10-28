package appmembershipservicelogic

import (
	"context"
	appaccount "go-zero-admin/application/applet/rpc/internal/logic/appaccountservice"

	"go-zero-admin/application/applet/rpc/internal/svc"
	"go-zero-admin/application/applet/rpc/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetAppMemberExampleLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetAppMemberExampleLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetAppMemberExampleLogic {
	return &GetAppMemberExampleLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *GetAppMemberExampleLogic) GetAppMemberExample(in *pb.AppExampleRequest) (*pb.AppExampleResponse, error) {
	if in == nil {
		return nil, membershipError("请求参数不能为空")
	}
	user, _, err := appaccount.ValidateAccess(l.ctx, l.svcCtx, in.AccessToken)
	if err != nil {
		return nil, err
	}
	if in.UserID != user.ID {
		return nil, membershipError("仅允许导出本人资料")
	}
	now := membershipNow()
	membership, err := membershipAt(l.svcCtx.DB.WithContext(l.ctx), user.ID, now)
	if err != nil {
		return nil, err
	}
	allowed := false
	for _, right := range membership.Entitlements {
		if right == profileExport {
			allowed = true
		}
	}
	if !membership.Active || !allowed {
		return nil, membershipError("需要有效的VIP资料导出权益")
	}
	account, err := userAccount(l.svcCtx.DB.WithContext(l.ctx), user.ID)
	if err != nil {
		return nil, err
	}
	return &pb.AppExampleResponse{UserID: user.ID, Account: account, Nickname: user.Nickname, GeneratedAt: timestamp(now)}, nil
}
