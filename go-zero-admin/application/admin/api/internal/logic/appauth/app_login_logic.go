package appauth

import (
	"context"
	"fmt"
	"github.com/zeromicro/go-zero/core/logx"
	"go-zero-admin/application/admin/api/internal/middleware"
	"go-zero-admin/application/admin/api/internal/svc"
	"go-zero-admin/application/admin/api/internal/types"
	"go-zero-admin/application/app/rpc/apppb"
	"go-zero-admin/pkg/xerr"
)

type AppLoginLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewAppLoginLogic(ctx context.Context, s *svc.ServiceContext) *AppLoginLogic {
	return &AppLoginLogic{ctx, s, logx.WithContext(ctx)}
}
func (l *AppLoginLogic) AppLogin(req *types.AppLoginReq) (*types.AppTokenResp, error) {
	if l.svcCtx.AppAuthRpc == nil {
		return nil, xerr.NewErrCode(xerr.SERVICE_UNAVAILABLE)
	}

	ip, ua := appRequestMeta(l.ctx)
	r, err := l.svcCtx.AppAuthRpc.AppLogin(l.ctx, &apppb.AppLoginRequest{Account: req.Account, Password: req.Password, IP: ip, UserAgent: ua})
	if err != nil {
		return nil, err
	}
	if r.User != nil {
		middleware.MarkAuditIdentity(l.ctx, 0, fmt.Sprintf("app:%d", r.User.Id), 0)
	}
	return &types.AppTokenResp{AccessToken: r.AccessToken, ExpiresAt: r.ExpiresAt, RefreshToken: r.RefreshToken, RefreshExpiresAt: r.RefreshExpiresAt, SessionId: r.SessionId, User: MapAppUser(r.User)}, nil
}

func appToken(ctx context.Context) string {
	v, _ := ctx.Value(middleware.CtxKeyAppAccessToken).(string)
	return v
}
func appRequestMeta(ctx context.Context) (string, string) {
	ip, _ := ctx.Value(middleware.CtxKeyClientIP).(string)
	ua, _ := ctx.Value(middleware.CtxKeyUserAgent).(string)
	return ip, ua
}
func MapAppUser(u *apppb.AppUserInfo) types.AppUserInfo {
	if u == nil {
		return types.AppUserInfo{}
	}
	return types.AppUserInfo{Id: u.Id, Account: u.Account, IdentityKind: u.IdentityKind, Nickname: u.Nickname, AvatarFileId: u.AvatarFileId, AvatarUrl: u.AvatarURL, Status: u.Status, ReviewReason: u.ReviewReason, EmailVerified: u.EmailVerified, PhoneVerified: u.PhoneVerified, LastLoginAt: u.LastLoginAt, CreatedAt: u.CreatedAt}
}
func MapMembership(r *apppb.MembershipGrantInfo) types.MembershipGrantInfo {
	if r == nil {
		return types.MembershipGrantInfo{}
	}
	return types.MembershipGrantInfo{Id: r.Id, UserId: r.UserID, PlanCode: r.PlanCode, Benefits: r.Benefits, RequestId: r.RequestId, GrantedBy: r.GrantedBy, GrantedByAdmin: r.GrantedByAdmin, StartsAt: r.StartsAt, ExpiresAt: r.ExpiresAt, Revoked: r.Revoked, RevokedAt: r.RevokedAt, CreatedAt: r.CreatedAt}
}
