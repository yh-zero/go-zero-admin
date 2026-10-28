package appmembershipservicelogic

import (
	"context"
	"errors"
	"go-zero-admin/application/applet/rpc/internal/logic/accessutil"
	appaccount "go-zero-admin/application/applet/rpc/internal/logic/appaccountservice"
	"go-zero-admin/application/applet/rpc/internal/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"go-zero-admin/application/applet/rpc/internal/svc"
	"go-zero-admin/application/applet/rpc/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

type RevokeAppMembershipLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewRevokeAppMembershipLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RevokeAppMembershipLogic {
	return &RevokeAppMembershipLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *RevokeAppMembershipLogic) RevokeAppMembership(in *pb.AppGrantRevokeRequest) (*pb.NoDataResponse, error) {
	if in == nil {
		return nil, membershipError("请求参数不能为空")
	}
	if err := appaccount.AuthorizeAdmin(l.ctx, l.svcCtx, in.Actor, grantPath, "DELETE"); err != nil {
		return nil, err
	}
	if in.ID <= 0 {
		return nil, membershipError("授予记录ID无效")
	}
	reason, err := validateReason(in.Reason)
	if err != nil {
		return nil, err
	}
	err = l.svcCtx.DB.WithContext(l.ctx).Transaction(func(tx *gorm.DB) error {
		if err := accessutil.LockAdminGuard(tx); err != nil {
			return err
		}
		var administrator model.SysUser
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&administrator, in.Actor.UserID).Error; err != nil {
			return err
		}
		if err := appaccount.AuthorizeAdmin(l.ctx, appaccount.WithDB(l.svcCtx, tx), in.Actor, grantPath, "DELETE"); err != nil {
			return err
		}
		var grant model.AppMembershipGrant
		if err := tx.First(&grant, in.ID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return membershipError("会员授予记录不存在")
			}
			return err
		}
		var user model.AppUser
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&user, grant.UserID).Error; err != nil {
			return err
		}
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&grant, in.ID).Error; err != nil {
			return err
		}
		if grant.RevokedAt != nil {
			return nil
		}
		now := membershipNow()
		if err := tx.Model(&grant).Updates(map[string]any{"revoked_at": now, "revoked_by": in.Actor.UserID, "revoke_reason": reason}).Error; err != nil {
			return err
		}
		return auditMembership(l.ctx, tx, in.Actor, "revoke", grantPath, "DELETE", &grant)
	})
	if err != nil {
		return nil, err
	}
	return &pb.NoDataResponse{}, nil
}
