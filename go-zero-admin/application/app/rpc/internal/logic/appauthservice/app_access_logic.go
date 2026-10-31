package appauthservicelogic

import (
	"context"
	"errors"
	"github.com/zeromicro/go-zero/core/logx"
	apppb "go-zero-admin/application/app/rpc/apppb"
	"go-zero-admin/application/app/rpc/internal/model"
	"go-zero-admin/application/app/rpc/internal/svc"
	"go-zero-admin/pkg/jwtx"
	"go-zero-admin/pkg/xerr"
)

type AppAccessLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewAppAccessLogic(ctx context.Context, s *svc.ServiceContext) *AppAccessLogic {
	return &AppAccessLogic{ctx, s, logx.WithContext(ctx)}
}
func (l *AppAccessLogic) AppAccess(in *apppb.AppAccessRequest) (*apppb.AppAccessResponse, error) {
	u, _, err := CurrentAppUser(l.ctx, l.svcCtx, in.AccessToken)
	if err != nil {
		return nil, err
	}
	return &apppb.AppAccessResponse{User: mapAppUser(u)}, nil
}

// CurrentAppUser 同时校验令牌、当前会话和账户状态，不信任 JWT 的历史权限快照。
func CurrentAppUser(ctx context.Context, s *svc.ServiceContext, token string) (*model.AppUser, *jwtx.Session, error) {
	c, err := jwtx.ParseAppToken(s.Config.JwtAuth.AccessSecret, token)
	if err != nil || c.UserID <= 0 {
		return nil, nil, xerr.NewErrCode(xerr.TOKEN_EXPIRE_ERROR)
	}
	session, err := s.Sessions.Validate(ctx, c.SessionId, c.UserID, c.ID)
	if err != nil {
		return nil, nil, sessionError(err)
	}
	u, err := s.AppUserModel.FindOne(ctx, uint64(c.UserID))
	if errors.Is(err, model.ErrNotFound) {
		return nil, nil, xerr.NewErrCode(xerr.TOKEN_EXPIRE_ERROR)
	}
	if err != nil {
		return nil, nil, xerr.NewErrCode(xerr.DB_ERROR)
	}
	if u.DeletedAt.Valid || u.Status != "active" {
		_ = s.Sessions.RevokeAll(ctx, c.UserID)
		return nil, nil, xerr.NewErrCode(xerr.USER_DISABLED)
	}
	return u, session, nil
}
func sessionError(err error) error {
	if errors.Is(err, jwtx.ErrSessionInvalid) {
		return xerr.NewErrCode(xerr.TOKEN_EXPIRE_ERROR)
	}
	return xerr.NewErrCode(xerr.SERVER_COMMON_ERROR)
}
