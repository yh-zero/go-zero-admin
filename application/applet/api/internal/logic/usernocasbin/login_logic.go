package usernocasbin

import (
	"context"

	"go-zero-admin/application/applet/api/internal/svc"
	"go-zero-admin/application/applet/api/internal/types"
	"go-zero-admin/application/applet/rpc/pb"
	"go-zero-admin/pkg/result/xerr"

	"github.com/jinzhu/copier"
	"github.com/zeromicro/go-zero/core/logx"
)

type LoginLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewLoginLogic(ctx context.Context, svcCtx *svc.ServiceContext) *LoginLogic {
	return &LoginLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *LoginLogic) Login(req *types.LoginRequest) (resp *types.LoginResponse, err error) {
	if err := consumeCaptcha(l.ctx, req.CaptchaId, req.Captcha, l.svcCtx.BizRedis); err != nil {
		return nil, err
	}

	// 获取用户信息
	userInfoRPC, err := l.svcCtx.AppletUserRPC.GetUserInfo(l.ctx, &pb.GetUserInfoRequest{UserName: req.UserName, Password: req.Password})
	if err != nil {
		logx.Errorf("GetUserInfo err: %v", err)
		return nil, err
	}
	var resUserInfo types.LoginResponse
	_ = copier.Copy(&resUserInfo.UserInfo, &userInfoRPC.UserInfo)

	// 获取token
	var tokenReq = pb.GetUserTokeRequest{}
	_ = copier.Copy(&tokenReq, userInfoRPC.UserInfo)
	tokenResp, err := l.svcCtx.AppletUserRPC.GetUserToke(l.ctx, &tokenReq)

	if err != nil {
		logx.Errorf("l.svcCtx.AppletRPC.GenerateToken err: %v", err)
		return nil, xerr.NewErrCodeMsg(900001, "内部错误：GenerateToken")
	}

	return &types.LoginResponse{
		AccessExpire: tokenResp.ExpiresAt,
		AccessToken:  tokenResp.Token,
		UserInfo:     resUserInfo.UserInfo,
	}, nil
}
