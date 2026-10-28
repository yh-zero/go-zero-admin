package usernocasbin

import (
	"context"

	"go-zero-admin/application/applet/api/internal/svc"
	"go-zero-admin/application/applet/api/internal/types"
	"go-zero-admin/application/applet/rpc/pb"
	"go-zero-admin/pkg/audit"
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

// consumeCaptcha forwards to the RPC tier, which owns the Redis schema and
// atomic GET+DEL semantics. Wrong, missing, or malformed captchas collapse to
// a single business error so callers cannot distinguish the reason.
func (l *LoginLogic) consumeCaptcha(ctx context.Context, captchaId, captcha string) error {
	matched, err := l.svcCtx.AppletCaptchaRPC.VerifyCaptcha(ctx, &pb.VerifyCaptchaRequest{Id: captchaId, Answer: captcha})
	if err != nil {
		return err
	}
	if matched == nil || !matched.Matched {
		return xerr.NewErrCode(xerr.CAPTCHA_ERROR)
	}
	return nil
}

func (l *LoginLogic) Login(req *types.LoginRequest) (resp *types.LoginResponse, err error) {
	if err := l.consumeCaptcha(l.ctx, req.CaptchaId, req.Captcha); err != nil {
		return nil, err
	}

	// 获取用户信息
	userInfoRPC, err := l.svcCtx.AppletUserRPC.GetUserInfo(l.ctx, &pb.GetUserInfoRequest{UserName: req.UserName, Password: req.Password})
	if err != nil {
		logx.Errorf("GetUserInfo err: %v", err)
		return nil, err
	}
	var resUserInfo types.LoginResponse
	audit.SetActor(l.ctx, audit.Actor{ID: userInfoRPC.UserInfo.ID, Name: userInfoRPC.UserInfo.Username, AuthorityID: userInfoRPC.UserInfo.AuthorityId})
	_ = copier.Copy(&resUserInfo.UserInfo, &userInfoRPC.UserInfo)

	// 获取token
	var tokenReq = pb.GetUserTokeRequest{}
	_ = copier.Copy(&tokenReq, userInfoRPC.UserInfo)
	request := audit.RequestFromContext(l.ctx)
	tokenReq.IP, tokenReq.UserAgent = request.IP, request.UserAgent
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
