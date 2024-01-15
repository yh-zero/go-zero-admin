package user

import (
	"context"

	"go-zero-admin/application/applet/api/internal/svc"
	"go-zero-admin/application/applet/api/internal/types"
	"go-zero-admin/application/applet/rpc/pb"

	"github.com/jinzhu/copier"
	"github.com/zeromicro/go-zero/core/logx"
)

type RegisterLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewRegisterLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RegisterLogic {
	return &RegisterLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *RegisterLogic) Register(req *types.RegisterRequest) (resp *types.RegisterResponse, err error) {
	var pbUserInfo pb.UserInfo
	_ = copier.Copy(&pbUserInfo, req)
	_, err = l.svcCtx.AppletUserRPC.Register(l.ctx, &pb.RegisterRequest{UserInfo: &pbUserInfo, AuthorityIds: req.AuthorityIds})
	if err != nil {
		logx.Errorf("l.svcCtx.AppletUserRPC.Register err: %v", err)
		return nil, err
	}

	return
}
