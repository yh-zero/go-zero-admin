package appaccountservicelogic

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"go-zero-admin/application/applet/rpc/internal/logic/accessutil"
	"go-zero-admin/application/applet/rpc/internal/model"
	"go-zero-admin/application/applet/rpc/internal/svc"
	"go-zero-admin/application/applet/rpc/pb"
	"go-zero-admin/pkg/audit"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const adminRoot = "/v1/sys/app-members/"

func listUsers(ctx context.Context, s *svc.ServiceContext, in *pb.AppAdminListRequest) (*pb.AppUserListResponse, error) {
	if in == nil {
		return nil, unauthorized()
	}
	if err := AuthorizeAdmin(ctx, s, in.Actor, adminRoot+"users", "GET"); err != nil {
		return nil, err
	}
	page, size, keyword := int64(1), int64(10), ""
	if in.Page != nil {
		page, size, keyword = in.Page.PageNo, in.Page.PageSize, strings.TrimSpace(in.Page.Keyword)
	}
	if page < 1 {
		page = 1
	}
	if size < 1 {
		size = 10
	}
	if size > 100 {
		size = 100
	}
	if page > 1000000 || utf8.RuneCountInString(keyword) > 254 {
		return nil, badInput("分页或搜索参数无效")
	}
	query := s.DB.WithContext(ctx).Model(&model.AppUser{}).Joins("JOIN app_user_identities i ON i.user_id = app_users.id")
	if in.Status != "" {
		switch in.Status {
		case "pending", "active", "rejected", "disabled":
		default:
			return nil, badInput("账号状态无效")
		}
		query = query.Where("app_users.status = ?", in.Status)
	}
	if keyword != "" {
		query = query.Where("i.identifier LIKE ? OR app_users.nickname LIKE ?", "%"+keyword+"%", "%"+keyword+"%")
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}
	var rows []struct {
		model.AppUser
		Account            string
		IdentityKind       string
		IdentityVerifiedAt *time.Time
	}
	if err := query.Select("app_users.*, i.identifier AS account, i.kind AS identity_kind, i.verified_at AS identity_verified_at").Order("app_users.id DESC").Offset(int((page - 1) * size)).Limit(int(size)).Scan(&rows).Error; err != nil {
		return nil, err
	}
	result := &pb.AppUserListResponse{Total: total, List: make([]*pb.AppUserInfo, 0, len(rows))}
	for _, r := range rows {
		result.List = append(result.List, &pb.AppUserInfo{ID: r.ID, Nickname: r.Nickname, Status: r.Status, ReviewReason: r.ReviewReason, Account: r.Account, IdentityKind: r.IdentityKind, IdentityVerified: r.IdentityVerifiedAt != nil, RegistrationIP: r.RegistrationIP, RegistrationRegion: r.RegistrationRegion, CreatedAt: timestamp(r.CreatedAt), UpdatedAt: timestamp(r.UpdatedAt)})
	}
	if err := AuthorizeAdmin(ctx, s, in.Actor, adminRoot+"users", "GET"); err != nil {
		return nil, err
	}
	return result, nil
}
func withAdmin(ctx context.Context, s *svc.ServiceContext, actor *pb.SessionRequest, path, method string, run func(*gorm.DB, *model.SysUser) error) error {
	if err := AuthorizeAdmin(ctx, s, actor, path, method); err != nil {
		return err
	}
	return s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := accessutil.LockAdminGuard(tx); err != nil {
			return err
		}
		var caller model.SysUser
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&caller, actor.UserID).Error; err != nil {
			return err
		}
		if err := AuthorizeAdmin(ctx, WithDB(s, tx), actor, path, method); err != nil {
			return err
		}
		return run(tx, &caller)
	})
}
func statusUser(ctx context.Context, s *svc.ServiceContext, in *pb.AppStatusRequest) (*pb.NoDataResponse, error) {
	if in == nil || in.UserID <= 0 {
		return nil, badInput("用户参数无效")
	}
	reason := strings.TrimSpace(in.Reason)
	if utf8.RuneCountInString(reason) > 500 || !utf8.ValidString(reason) {
		return nil, badInput("原因不能超过500个字符")
	}
	switch in.Status {
	case "active":
	case "disabled", "rejected":
		if reason == "" {
			return nil, badInput("拒绝或禁用账号须填写原因")
		}
	default:
		return nil, badInput("目标状态无效")
	}
	err := withAdmin(ctx, s, in.Actor, adminRoot+"status", "PUT", func(tx *gorm.DB, caller *model.SysUser) error {
		var user model.AppUser
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&user, in.UserID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return badInput("App用户不存在")
			}
			return err
		}
		if user.Status == in.Status {
			return nil
		}
		valid := (user.Status == "pending" && (in.Status == "active" || in.Status == "rejected")) || (user.Status == "active" && in.Status == "disabled") || (user.Status == "disabled" && in.Status == "active") || (user.Status == "rejected" && in.Status == "active")
		if !valid {
			return badInput("不允许此账号状态变更")
		}
		now := time.Now().UTC()
		if err := tx.Model(&user).Updates(map[string]any{"status": in.Status, "review_reason": reason, "reviewed_by": caller.ID, "reviewed_at": now, "session_version": gorm.Expr("session_version + 1")}).Error; err != nil {
			return err
		}
		if err := tx.Model(&model.AppSession{}).Where("user_id = ? AND revoked_at IS NULL", user.ID).Update("revoked_at", now).Error; err != nil {
			return err
		}
		return audit.Record(ctx, tx, audit.Event{ActorID: caller.ID, ActorName: caller.Username, AuthorityID: in.Actor.AuthorityId, Module: "app-members", Action: "setAppUserStatus:" + in.Status, Object: "app-user:" + strconv.FormatInt(user.ID, 10), Path: adminRoot + "status", Method: "PUT"})
	})
	if err != nil {
		return nil, err
	}
	return &pb.NoDataResponse{}, nil
}
func adminSessions(ctx context.Context, s *svc.ServiceContext, in *pb.AppAdminUserRequest) (*pb.AppSessionListResponse, error) {
	if in == nil || in.UserID <= 0 {
		return nil, badInput("用户参数无效")
	}
	if err := AuthorizeAdmin(ctx, s, in.Actor, adminRoot+"sessions", "GET"); err != nil {
		return nil, err
	}
	result, err := sessions(ctx, s, in.UserID, "")
	if err != nil {
		return nil, err
	}
	if err = AuthorizeAdmin(ctx, s, in.Actor, adminRoot+"sessions", "GET"); err != nil {
		return nil, err
	}
	return result, nil
}
func revokeAdminSession(ctx context.Context, s *svc.ServiceContext, in *pb.AppAdminRevokeRequest) (*pb.NoDataResponse, error) {
	if in == nil || in.ID == "" || len(in.ID) > 36 {
		return nil, badInput("会话参数无效")
	}
	err := withAdmin(ctx, s, in.Actor, adminRoot+"session", "DELETE", func(tx *gorm.DB, caller *model.SysUser) error {
		var session model.AppSession
		if err := tx.Where("id = ?", in.ID).First(&session).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return badInput("App会话不存在")
			}
			return err
		}
		var user model.AppUser
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&user, session.UserID).Error; err != nil {
			return err
		}
		if err := tx.Where("id = ?", in.ID).First(&session).Error; err != nil {
			return err
		}
		if session.RevokedAt != nil {
			return nil
		}
		if err := tx.Model(&session).Update("revoked_at", time.Now().UTC()).Error; err != nil {
			return err
		}
		return audit.Record(ctx, tx, audit.Event{ActorID: caller.ID, ActorName: caller.Username, AuthorityID: in.Actor.AuthorityId, Module: "app-members", Action: "revokeAppSession", Object: "app-user:" + strconv.FormatInt(session.UserID, 10) + "/session:" + session.ID, Path: adminRoot + "session", Method: "DELETE"})
	})
	if err != nil {
		return nil, err
	}
	return &pb.NoDataResponse{}, nil
}
