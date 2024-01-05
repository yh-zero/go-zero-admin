package user

import (
	"context"
	"fmt"
	"github.com/jinzhu/copier"
	"go-zero-admin/application/applet/rpc/pb"

	"go-zero-admin/application/applet/api/internal/svc"
	"go-zero-admin/application/applet/api/internal/types"

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
	register, err := l.svcCtx.AppletUserRPC.Register(l.ctx, &pb.RegisterRequest{UserInfo: &pbUserInfo, AuthorityIds: req.AuthorityIds})
	fmt.Println("-------- register", register)
	if err != nil {
		logx.Errorf("l.svcCtx.AppletUserRPC.Register err: %v", err)
		return nil, err
	}

	return
}
