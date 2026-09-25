package userlogic

import (
	"context"
	"errors"
	"github.com/jinzhu/copier"
	"github.com/zeromicro/go-zero/core/logx"
	"go-zero-admin/application/applet/rpc/internal/model"
	"go-zero-admin/application/applet/rpc/internal/svc"
	"go-zero-admin/application/applet/rpc/pb"
	"gorm.io/gorm"
)

type GetCurrentUserLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetCurrentUserLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetCurrentUserLogic {
	return &GetCurrentUserLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}
func (l *GetCurrentUserLogic) GetCurrentUser(in *pb.SessionRequest) (*pb.GetUserInfoResponse, error) {
	var user model.SysUser
	err := sessionQuery(l.svcCtx.DB.WithContext(l.ctx), in).Preload("Authority").Preload("Authorities").First(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, sessionExpired()
	}
	if err != nil {
		return nil, err
	}
	NewGetUserInfoLogic(l.ctx, l.svcCtx).UserAuthorityDefaultRouter(&user)
	out := &pb.UserInfo{}
	if err := copier.Copy(out, &user); err != nil {
		return nil, err
	}
	out.Password = ""
	return &pb.GetUserInfoResponse{UserInfo: out}, nil
}
