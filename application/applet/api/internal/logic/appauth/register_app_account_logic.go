// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package appauth

import (
	"context"
	"go-zero-admin/application/applet/api/internal/appbridge"
	"go-zero-admin/application/applet/rpc/pb"

	"go-zero-admin/application/applet/api/internal/svc"
	"go-zero-admin/application/applet/api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type RegisterAppAccountLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 邮箱或手机号注册，等待管理员审核
func NewRegisterAppAccountLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RegisterAppAccountLogic {
	return &RegisterAppAccountLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *RegisterAppAccountLogic) RegisterAppAccount(req *types.AppRegisterRequest) (resp *types.AppUser, err error) {
	reply, err := l.svcCtx.AppAccountRPC.RegisterAppAccount(l.ctx, &pb.AppRegisterRequest{Account: req.Account, Password: req.Password, Nickname: req.Nickname, EmailCode: req.EmailCode, IP: appbridge.Request(l.ctx).IP})
	if err != nil {
		return nil, err
	}
	return appbridge.Copy[types.AppUser](reply)
}
