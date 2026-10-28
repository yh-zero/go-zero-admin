// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package appaccount

import (
	"context"
	"go-zero-admin/application/applet/api/internal/appbridge"
	"go-zero-admin/application/applet/rpc/pb"

	"go-zero-admin/application/applet/api/internal/svc"
	"go-zero-admin/application/applet/api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetAppMemberExampleLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 会员示例：导出本人资料
func NewGetAppMemberExampleLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetAppMemberExampleLogic {
	return &GetAppMemberExampleLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetAppMemberExampleLogic) GetAppMemberExample(req *types.AppUserIDRequest) (resp *types.AppExample, err error) {
	reply, err := l.svcCtx.AppMembershipRPC.GetAppMemberExample(l.ctx, &pb.AppExampleRequest{AccessToken: appbridge.Request(l.ctx).Token, UserID: req.UserID})
	if err != nil {
		return nil, err
	}
	return appbridge.Copy[types.AppExample](reply)
}
