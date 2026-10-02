package sessionmanagelogic

import (
	"context"
	"errors"
	"github.com/gofrs/uuid/v5"
	"go-zero-admin/application/applet/rpc/internal/logic/accessutil"
	userlogic "go-zero-admin/application/applet/rpc/internal/logic/user"
	"go-zero-admin/application/applet/rpc/internal/model"
	"go-zero-admin/pkg/audit"
	"go-zero-admin/pkg/orm"
	"go-zero-admin/pkg/result/xerr"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"time"

	"go-zero-admin/application/applet/rpc/internal/svc"
	"go-zero-admin/application/applet/rpc/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

type RevokeDeviceSessionLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewRevokeDeviceSessionLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RevokeDeviceSessionLogic {
	return &RevokeDeviceSessionLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *RevokeDeviceSessionLogic) RevokeDeviceSession(in *pb.RevokeDeviceSessionRequest) (*pb.NoDataResponse, error) {
	if in == nil || in.Actor == nil || in.Actor.UserID <= 0 {
		return nil, xerr.NewErrCode(xerr.REUQEST_PARAM_ERROR)
	}
	if _, err := uuid.FromString(in.ID); err != nil {
		return nil, xerr.NewErrCode(xerr.REUQEST_PARAM_ERROR)
	}
	err := l.svcCtx.DB.WithContext(l.ctx).Transaction(func(tx *gorm.DB) error {
		if err := accessutil.LockAdminGuard(tx); err != nil {
			return err
		}
		var caller model.SysUser
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&caller, in.Actor.UserID).Error; err != nil {
			return err
		}
		valid, err := userlogic.NewCheckSessionLogic(l.ctx, &svc.ServiceContext{DB: &orm.DB{DB: tx}}).CheckSession(in.Actor)
		if err != nil {
			return err
		}
		if !valid.Valid {
			return xerr.NewErrCode(xerr.TOKEN_EXPIRE_ERROR)
		}
		query := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", in.ID)
		if in.SelfOnly || in.Actor.AuthorityId != accessutil.AdminAuthorityID {
			query = query.Where("user_id = ?", in.Actor.UserID)
		}
		var target model.SysDeviceSession
		if err = query.First(&target).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return xerr.NewErrCodeMsg(xerr.REUQEST_PARAM_ERROR, "设备会话不存在或无权操作")
			}
			return err
		}
		if target.RevokedAt != nil {
			return nil
		}
		if err = tx.Model(&target).Update("revoked_at", time.Now().UTC()).Error; err != nil {
			return err
		}
		return audit.Record(l.ctx, tx, audit.Event{Module: "session", Action: "revokeDeviceSession", Object: target.ID})
	})
	if err != nil {
		return nil, err
	}
	return &pb.NoDataResponse{}, nil
}
