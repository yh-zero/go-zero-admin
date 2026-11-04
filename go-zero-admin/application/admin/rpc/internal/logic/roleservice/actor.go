package roleservicelogic

import (
	"context"
	"errors"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
	"go-zero-admin/application/admin/rpc/adminpb"
	"go-zero-admin/application/admin/rpc/internal/model"
	"go-zero-admin/application/admin/rpc/internal/svc"
	"go-zero-admin/pkg/xerr"
)

func requireRoleActor(ctx context.Context, s *svc.ServiceContext, actor *adminpb.Actor) error {
	if actor.GetUserId() <= 0 {
		return xerr.NewErrCode(xerr.PERMISSION_DENIED)
	}
	// 复用自查身份路径，只确认当前账号有效，不套用用户管理的数据范围门槛。
	user, err := s.SysUserModel.FindScopedUser(ctx, actor.GetUserId(), model.UserActor{UserId: actor.GetUserId(), OwnAccount: true})
	if errors.Is(err, model.ErrDataScopeDenied) || errors.Is(err, sqlx.ErrNotFound) {
		return xerr.NewErrCode(xerr.PERMISSION_DENIED)
	}
	if err != nil {
		return xerr.NewErrCode(xerr.DB_ERROR)
	}
	if user.Status != 1 || user.DeletedAt.Valid {
		return xerr.NewErrCode(xerr.PERMISSION_DENIED)
	}
	return nil
}
