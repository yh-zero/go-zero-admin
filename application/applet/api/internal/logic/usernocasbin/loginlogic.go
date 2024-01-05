package usernocasbin

import (
	"context"
	"fmt"
	"strconv"

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
	remoteAddrIp := l.ctx.Value("RemoteAddr") // 获取id 存redis
	IsDev := l.ctx.Value("Isdev")
	if IsDev != strconv.Itoa(1) {
		fmt.Println("GetIP(remoteAddrIp)", GetIP(remoteAddrIp))
		cacheCode, err := GetActivationCache(GetIP(remoteAddrIp), l.svcCtx.BizRedis)
		fmt.Println("cacheCode", cacheCode)
		if err != nil {
			logx.Errorf("getActivationCache：获取redis中的验证码错误 error: %v", err)
			return nil, err
		}
		if cacheCode != req.Captcha {
			logx.Errorf("验证码错误 cacheCode: %s != reqCaptcha: %s", cacheCode, req.Captcha)
			return nil, xerr.NewErrCode(xerr.CAPTCHA_ERROR)
		}
	}
	// 获取用户信息
	userInfoRPC, err := l.svcCtx.AppletUserRPC.GetUserInfo(l.ctx, &pb.GetUserInfoRequest{UserName: req.UserName, Password: req.Password})
	if err != nil {
		logx.Errorf("GetUserInfo err: %v", err)
		return nil, xerr.NewErrCode(xerr.USER_PASSWORD_ERROR)
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
