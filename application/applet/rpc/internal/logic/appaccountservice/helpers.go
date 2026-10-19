package appaccountservicelogic

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	casbinlogic "go-zero-admin/application/applet/rpc/internal/logic/casbin"
	userlogic "go-zero-admin/application/applet/rpc/internal/logic/user"
	"go-zero-admin/application/applet/rpc/internal/model"
	"go-zero-admin/application/applet/rpc/internal/svc"
	"go-zero-admin/application/applet/rpc/pb"
	"go-zero-admin/pkg/appauth"
	"go-zero-admin/pkg/orm"
	"go-zero-admin/pkg/result/xerr"
	"gorm.io/gorm"
)

func badInput(message string) error { return xerr.NewErrCodeMsg(xerr.REUQEST_PARAM_ERROR, message) }
func unauthorized() error           { return xerr.NewErrCode(xerr.TOKEN_EXPIRE_ERROR) }
func WithDB(s *svc.ServiceContext, tx *gorm.DB) *svc.ServiceContext {
	copy := *s
	copy.DB = &orm.DB{DB: tx}
	return &copy
}
func ValidateAccess(ctx context.Context, s *svc.ServiceContext, token string) (*model.AppUser, *model.AppSession, error) {
	if s == nil || s.DB == nil {
		return nil, nil, errors.New("App authentication unavailable")
	}
	claims, err := appauth.Parse(s.Config.AppJwtAuth, token, "access")
	if err != nil {
		return nil, nil, unauthorized()
	}
	var user model.AppUser
	if err = s.DB.WithContext(ctx).Where("id = ? AND status = ? AND session_version = ?", claims.UserID, "active", claims.SessionVersion).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, unauthorized()
		}
		return nil, nil, err
	}
	var session model.AppSession
	if err = s.DB.WithContext(ctx).Where("id = ? AND user_id = ? AND session_version = ? AND revoked_at IS NULL AND expires_at > ?", claims.SessionID, user.ID, user.SessionVersion, time.Now().UTC()).First(&session).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, unauthorized()
		}
		return nil, nil, err
	}
	return &user, &session, nil
}
func AuthorizeAdmin(ctx context.Context, s *svc.ServiceContext, actor *pb.SessionRequest, path, method string) error {
	if actor == nil || actor.UserID <= 0 || actor.AuthorityId <= 0 || actor.SessionVersion <= 0 || actor.SessionID == "" {
		return unauthorized()
	}
	if s == nil || s.DB == nil || s.Casbin == nil {
		return errors.New("administrator authorization unavailable")
	}
	valid, err := userlogic.NewCheckSessionLogic(ctx, s).CheckSession(actor)
	if err != nil {
		return err
	}
	if !valid.Valid {
		return unauthorized()
	}
	allowed, err := casbinlogic.NewEnforceLogic(ctx, s).Enforce(&pb.EnforceRequest{AuthorityId: strconv.FormatInt(actor.AuthorityId, 10), Path: path, Method: method})
	if err != nil {
		return err
	}
	if !allowed.Pass {
		return badInput("无权执行此操作")
	}
	return nil
}
func timestamp(t time.Time) string { return t.UTC().Format(time.RFC3339Nano) }
func nickname(raw string) (string, error) {
	value := strings.TrimSpace(raw)
	if !utf8.ValidString(value) || utf8.RuneCountInString(value) < 1 || utf8.RuneCountInString(value) > 64 || strings.ContainsAny(value, "\r\n\x00") {
		return "", badInput("昵称须为1至64个字符")
	}
	return value, nil
}
func userInfo(ctx context.Context, s *svc.ServiceContext, user *model.AppUser) (*pb.AppUserInfo, error) {
	var identity model.AppUserIdentity
	if err := s.DB.WithContext(ctx).Select("identifier", "kind", "verified_at").Where("user_id = ?", user.ID).First(&identity).Error; err != nil {
		return nil, err
	}
	return &pb.AppUserInfo{ID: user.ID, Account: identity.Identifier, IdentityKind: identity.Kind, IdentityVerified: identity.VerifiedAt != nil, RegistrationIP: user.RegistrationIP, RegistrationRegion: user.RegistrationRegion, Nickname: user.Nickname, Status: user.Status, ReviewReason: user.ReviewReason, CreatedAt: timestamp(user.CreatedAt), UpdatedAt: timestamp(user.UpdatedAt)}, nil
}
