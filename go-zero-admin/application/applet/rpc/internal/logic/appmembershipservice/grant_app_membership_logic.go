package appmembershipservicelogic

import (
	"context"
	"errors"
	"go-zero-admin/application/applet/rpc/internal/logic/accessutil"
	appaccount "go-zero-admin/application/applet/rpc/internal/logic/appaccountservice"
	"go-zero-admin/application/applet/rpc/internal/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"strings"
	"time"

	"go-zero-admin/application/applet/rpc/internal/svc"
	"go-zero-admin/application/applet/rpc/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

type GrantAppMembershipLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGrantAppMembershipLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GrantAppMembershipLogic {
	return &GrantAppMembershipLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *GrantAppMembershipLogic) GrantAppMembership(in *pb.AppGrantRequest) (*pb.AppGrantInfo, error) {
	if in == nil {
		return nil, membershipError("请求参数不能为空")
	}
	if err := appaccount.AuthorizeAdmin(l.ctx, l.svcCtx, in.Actor, grantsPath, "POST"); err != nil {
		return nil, err
	}
	if in.UserID <= 0 || in.PlanID <= 0 || in.Days < 1 || in.Days > 3650 || !requestIDPattern.MatchString(in.RequestID) {
		return nil, membershipError("用户、方案、天数或请求标识无效")
	}
	reason, err := validateReason(in.Reason)
	if err != nil {
		return nil, err
	}
	var result *pb.AppGrantInfo
	err = l.svcCtx.DB.WithContext(l.ctx).Transaction(func(tx *gorm.DB) error {
		if err := accessutil.LockAdminGuard(tx); err != nil {
			return err
		}
		// Legacy administrator logout updates this row without the role guard.
		// Hold it so session revalidation and the grant commit are serialized.
		var administrator model.SysUser
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&administrator, in.Actor.UserID).Error; err != nil {
			return err
		}
		if err := appaccount.AuthorizeAdmin(l.ctx, appaccount.WithDB(l.svcCtx, tx), in.Actor, grantsPath, "POST"); err != nil {
			return err
		}
		// Every grant and revocation locks its App user before reading grants.
		var user model.AppUser
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&user, in.UserID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return membershipError("App用户不存在")
			}
			return err
		}
		var existing model.AppMembershipGrant
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("request_id = ?", in.RequestID).Take(&existing).Error
		if err == nil {
			if existing.UserID != in.UserID || existing.PlanID != in.PlanID || existing.Days != in.Days || existing.Reason != reason {
				return membershipError("请求标识已被其他开通参数使用")
			}
			account, err := userAccount(tx, user.ID)
			if err != nil {
				return err
			}
			result, err = grantProto(&existing, account, membershipNow())
			return err
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		if user.Status != "active" {
			return membershipError("仅可为已审核且启用的App用户开通会员")
		}
		var plan model.AppMembershipPlan
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND code = ? AND version = ? AND enabled = ?", in.PlanID, "vip", 1, true).First(&plan).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return membershipError("VIP方案不存在或已停用")
			}
			return err
		}
		rights, err := entitlements(plan.Entitlements)
		if err != nil {
			return err
		}
		if len(rights) != 1 || rights[0] != profileExport {
			return membershipError("VIP方案权益配置无效")
		}
		now := membershipNow()
		start := now
		var latest model.AppMembershipGrant
		// A locking read sees renewals committed while the user lock was held by
		// another transaction, even under MySQL REPEATABLE READ.
		err = tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("user_id = ? AND revoked_at IS NULL AND ends_at > ?", user.ID, now).Order("ends_at DESC, id DESC").Take(&latest).Error
		if err == nil {
			start = latest.EndsAt
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		end := start.Add(time.Duration(in.Days) * 24 * time.Hour)
		if end.Year() > 9999 {
			return membershipError("会员有效期超出支持范围")
		}
		grant := model.AppMembershipGrant{UserID: user.ID, PlanID: plan.ID, PlanCode: plan.Code, PlanVersion: plan.Version, Entitlements: strings.TrimSpace(plan.Entitlements), RequestID: in.RequestID, Days: in.Days, StartsAt: start, EndsAt: end, GrantedBy: in.Actor.UserID, Reason: reason, CreatedAt: now}
		if err := tx.Create(&grant).Error; err != nil {
			return err
		}
		account, err := userAccount(tx, user.ID)
		if err != nil {
			return err
		}
		result, err = grantProto(&grant, account, now)
		if err != nil {
			return err
		}
		return auditMembership(l.ctx, tx, in.Actor, "grant", grantsPath, "POST", &grant)
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}
