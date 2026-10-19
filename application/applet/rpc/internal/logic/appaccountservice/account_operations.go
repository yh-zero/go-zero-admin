package appaccountservicelogic

import (
	"context"
	"crypto/subtle"
	"errors"
	"net/netip"
	"strings"
	"time"

	"github.com/gofrs/uuid/v5"
	"go-zero-admin/application/applet/rpc/internal/model"
	"go-zero-admin/application/applet/rpc/internal/svc"
	"go-zero-admin/application/applet/rpc/pb"
	"go-zero-admin/pkg/appauth"
	"go-zero-admin/pkg/hash"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func register(ctx context.Context, s *svc.ServiceContext, in *pb.AppRegisterRequest) (*pb.AppUserInfo, error) {
	if in == nil {
		return nil, badInput("注册参数不能为空")
	}
	kind, account, err := appauth.NormalizeAccount(in.Account)
	if err != nil {
		return nil, badInput(err.Error())
	}
	if err = credentialLimit(ctx, s, "register", account, in.IP); err != nil {
		return nil, err
	}
	requestedNickname := in.Nickname
	if strings.TrimSpace(requestedNickname) == "" {
		requestedNickname = "App用户"
	}
	nick, err := nickname(requestedNickname)
	if err != nil {
		return nil, err
	}
	if err = hash.ValidatePassword(in.Password); err != nil {
		return nil, badInput(err.Error())
	}
	verifiedAt, err := verifyRegistrationEmail(ctx, s, kind, account, in.EmailCode)
	if err != nil {
		return nil, err
	}
	passwordHash, err := hash.BcryptHash(in.Password)
	if err != nil {
		return nil, err
	}
	user := &model.AppUser{Nickname: nick, Status: "pending", SessionVersion: 1}
	if ip, parseErr := netip.ParseAddr(strings.TrimSpace(in.IP)); parseErr == nil {
		user.RegistrationIP = ip.Unmap().String()
		user.RegistrationRegion = "未知"
		if s.IPLocation != nil {
			user.RegistrationRegion = s.IPLocation.Lookup(user.RegistrationIP)
		}
	}
	err = s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var count int64
		if err := tx.Model(&model.AppUserIdentity{}).Where("identifier = ?", account).Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			return badInput("该账号已注册")
		}
		if err := tx.Create(user).Error; err != nil {
			return err
		}
		if err := tx.Create(&model.AppUserIdentity{UserID: user.ID, Kind: kind, Identifier: account, PasswordHash: passwordHash, VerifiedAt: verifiedAt}).Error; err != nil {
			if strings.Contains(strings.ToLower(err.Error()), "unique") || strings.Contains(strings.ToLower(err.Error()), "duplicate") {
				return badInput("该账号已注册")
			}
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return userInfo(ctx, s, user)
}
func login(ctx context.Context, s *svc.ServiceContext, in *pb.AppLoginRequest) (*pb.AppAuthTokens, error) {
	if in == nil {
		return nil, badInput("登录参数不能为空")
	}
	_, account, err := appauth.NormalizeAccount(in.Account)
	if err != nil {
		return nil, badInput("账号或密码不正确")
	}
	if err = credentialLimit(ctx, s, "login", account, in.IP); err != nil {
		return nil, err
	}
	if len(in.Password) < 1 || len(in.Password) > 72 {
		return nil, badInput("账号或密码不正确")
	}
	var identity model.AppUserIdentity
	if err = s.DB.WithContext(ctx).Where("identifier = ?", account).First(&identity).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, badInput("账号或密码不正确")
		}
		return nil, err
	}
	if !hash.BcryptCheck(in.Password, identity.PasswordHash) {
		return nil, badInput("账号或密码不正确")
	}
	var result *pb.AppAuthTokens
	err = s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var user model.AppUser
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&user, identity.UserID).Error; err != nil {
			return err
		}
		if user.Status != "active" {
			return badInput("账号尚未开通或已停用，请联系管理员")
		}
		id, err := uuid.NewV4()
		if err != nil {
			return err
		}
		session := &model.AppSession{ID: id.String(), UserID: user.ID, SessionVersion: user.SessionVersion, RefreshGeneration: 1, IP: bounded(in.IP, 64), UserAgent: bounded(in.UserAgent, 512), ExpiresAt: time.Now().UTC().Add(time.Duration(s.Config.AppJwtAuth.RefreshExpire) * time.Second)}
		result, err = issueTokens(ctx, WithDB(s, tx), &user, session)
		if err != nil {
			return err
		}
		return tx.Create(session).Error
	})
	return result, err
}
func issueTokens(ctx context.Context, s *svc.ServiceContext, user *model.AppUser, session *model.AppSession) (*pb.AppAuthTokens, error) {
	accessExpiry := time.Now().UTC().Add(time.Duration(s.Config.AppJwtAuth.AccessExpire) * time.Second)
	if accessExpiry.After(session.ExpiresAt) {
		accessExpiry = session.ExpiresAt
	}
	access, err := appauth.Issue(s.Config.AppJwtAuth, user.ID, session.ID, user.SessionVersion, session.RefreshGeneration, "access", accessExpiry)
	if err != nil {
		return nil, err
	}
	refresh, err := appauth.Issue(s.Config.AppJwtAuth, user.ID, session.ID, user.SessionVersion, session.RefreshGeneration, "refresh", session.ExpiresAt)
	if err != nil {
		return nil, err
	}
	session.RefreshHash = appauth.Digest(refresh)
	info, err := userInfo(ctx, s, user)
	if err != nil {
		return nil, err
	}
	return &pb.AppAuthTokens{AccessToken: access, RefreshToken: refresh, ExpiresAt: accessExpiry.Unix(), RefreshExpiresAt: session.ExpiresAt.Unix(), User: info}, nil
}
func refresh(ctx context.Context, s *svc.ServiceContext, in *pb.AppRefreshRequest) (*pb.AppAuthTokens, error) {
	if in == nil {
		return nil, unauthorized()
	}
	claims, err := appauth.Parse(s.Config.AppJwtAuth, in.RefreshToken, "refresh")
	if err != nil {
		return nil, unauthorized()
	}
	var result *pb.AppAuthTokens
	replayed := false
	err = s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var user model.AppUser
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&user, claims.UserID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return unauthorized()
			}
			return err
		}
		var session model.AppSession
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND user_id = ?", claims.SessionID, user.ID).First(&session).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return unauthorized()
			}
			return err
		}
		now := time.Now().UTC()
		if user.Status != "active" || user.SessionVersion != claims.SessionVersion || session.SessionVersion != user.SessionVersion || session.RevokedAt != nil || !session.ExpiresAt.After(now) {
			return unauthorized()
		}
		digest := appauth.Digest(in.RefreshToken)
		if claims.Generation != session.RefreshGeneration || subtle.ConstantTimeCompare([]byte(digest), []byte(session.RefreshHash)) != 1 {
			replayed = true
			return tx.Model(&session).Update("revoked_at", now).Error
		}
		previous := session.RefreshHash
		session.RefreshGeneration++
		session.IP = bounded(in.IP, 64)
		session.UserAgent = bounded(in.UserAgent, 512)
		var err error
		result, err = issueTokens(ctx, WithDB(s, tx), &user, &session)
		if err != nil {
			return err
		}
		return tx.Model(&session).Updates(map[string]any{"refresh_hash": session.RefreshHash, "previous_refresh_hash": previous, "refresh_generation": session.RefreshGeneration, "ip": session.IP, "user_agent": session.UserAgent}).Error
	})
	if err != nil {
		return nil, err
	}
	if replayed {
		return nil, unauthorized()
	}
	return result, nil
}
func bounded(value string, max int) string {
	runes := []rune(strings.TrimSpace(value))
	if len(runes) > max {
		return string(runes[:max])
	}
	return string(runes)
}

func profile(ctx context.Context, s *svc.ServiceContext, in *pb.AppAccessRequest) (*pb.AppUserInfo, error) {
	if in == nil {
		return nil, unauthorized()
	}
	u, _, err := ValidateAccess(ctx, s, in.AccessToken)
	if err != nil {
		return nil, err
	}
	result, err := userInfo(ctx, s, u)
	if err != nil {
		return nil, err
	}
	if _, _, err = ValidateAccess(ctx, s, in.AccessToken); err != nil {
		return nil, err
	}
	return result, nil
}
func updateProfile(ctx context.Context, s *svc.ServiceContext, in *pb.AppProfileRequest) (*pb.AppUserInfo, error) {
	if in == nil {
		return nil, unauthorized()
	}
	nick, err := nickname(in.Nickname)
	if err != nil {
		return nil, err
	}
	var result *pb.AppUserInfo
	err = withAppSession(ctx, s, in.AccessToken, func(tx *gorm.DB, u *model.AppUser, _ *model.AppSession) error {
		if err := tx.Model(u).Update("nickname", nick).Error; err != nil {
			return err
		}
		var err error
		result, err = userInfo(ctx, WithDB(s, tx), u)
		return err
	})
	return result, err
}
func withAppSession(ctx context.Context, s *svc.ServiceContext, raw string, run func(*gorm.DB, *model.AppUser, *model.AppSession) error) error {
	claims, err := appauth.Parse(s.Config.AppJwtAuth, raw, "access")
	if err != nil {
		return unauthorized()
	}
	return s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var lock model.AppUser
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&lock, claims.UserID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return unauthorized()
			}
			return err
		}
		u, ss, err := ValidateAccess(ctx, WithDB(s, tx), raw)
		if err != nil {
			return err
		}
		return run(tx, u, ss)
	})
}
func logout(ctx context.Context, s *svc.ServiceContext, in *pb.AppLogoutRequest) (*pb.NoDataResponse, error) {
	if in == nil {
		return nil, unauthorized()
	}
	err := withAppSession(ctx, s, in.AccessToken, func(tx *gorm.DB, u *model.AppUser, ss *model.AppSession) error {
		query := tx.Model(&model.AppSession{}).Where("user_id = ? AND revoked_at IS NULL", u.ID)
		if !in.AllDevices {
			query = query.Where("id = ?", ss.ID)
		} else {
			if err := tx.Model(u).Update("session_version", gorm.Expr("session_version + 1")).Error; err != nil {
				return err
			}
		}
		return query.Update("revoked_at", time.Now().UTC()).Error
	})
	if err != nil {
		return nil, err
	}
	return &pb.NoDataResponse{}, nil
}
func sessions(ctx context.Context, s *svc.ServiceContext, userID int64, current string) (*pb.AppSessionListResponse, error) {
	var rows []model.AppSession
	if err := s.DB.WithContext(ctx).Select("id", "user_id", "ip", "user_agent", "created_at", "expires_at").Where("user_id = ? AND revoked_at IS NULL AND expires_at > ?", userID, time.Now().UTC()).Order("created_at DESC, id DESC").Find(&rows).Error; err != nil {
		return nil, err
	}
	result := &pb.AppSessionListResponse{List: make([]*pb.AppSessionInfo, 0, len(rows))}
	for _, r := range rows {
		result.List = append(result.List, &pb.AppSessionInfo{ID: r.ID, UserID: r.UserID, IP: r.IP, UserAgent: r.UserAgent, CreatedAt: timestamp(r.CreatedAt), ExpiresAt: timestamp(r.ExpiresAt), Current: r.ID == current})
	}
	return result, nil
}
func mySessions(ctx context.Context, s *svc.ServiceContext, in *pb.AppAccessRequest) (*pb.AppSessionListResponse, error) {
	if in == nil {
		return nil, unauthorized()
	}
	u, ss, err := ValidateAccess(ctx, s, in.AccessToken)
	if err != nil {
		return nil, err
	}
	result, err := sessions(ctx, s, u.ID, ss.ID)
	if err != nil {
		return nil, err
	}
	if _, _, err = ValidateAccess(ctx, s, in.AccessToken); err != nil {
		return nil, err
	}
	return result, nil
}
func revokeMine(ctx context.Context, s *svc.ServiceContext, in *pb.AppRevokeSelfRequest) (*pb.NoDataResponse, error) {
	if in == nil {
		return nil, unauthorized()
	}
	if in.ID == "" || len(in.ID) > 36 {
		return nil, badInput("会话参数无效")
	}
	err := withAppSession(ctx, s, in.AccessToken, func(tx *gorm.DB, u *model.AppUser, _ *model.AppSession) error {
		var target model.AppSession
		if err := tx.Where("id = ? AND user_id = ?", in.ID, u.ID).First(&target).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return badInput("会话不存在或无权操作")
			}
			return err
		}
		if target.RevokedAt != nil {
			return nil
		}
		return tx.Model(&target).Update("revoked_at", time.Now().UTC()).Error
	})
	if err != nil {
		return nil, err
	}
	return &pb.NoDataResponse{}, nil
}
