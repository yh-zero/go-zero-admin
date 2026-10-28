package permissionlogic

import (
	"context"

	"go-zero-admin/application/applet/rpc/internal/logic/accessutil"
	"gorm.io/gorm"

	"go-zero-admin/application/applet/rpc/internal/svc"
	"go-zero-admin/application/applet/rpc/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

type PreviewPermissionRollbackLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewPreviewPermissionRollbackLogic(ctx context.Context, svcCtx *svc.ServiceContext) *PreviewPermissionRollbackLogic {
	return &PreviewPermissionRollbackLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *PreviewPermissionRollbackLogic) PreviewPermissionRollback(in *pb.PreviewPermissionRollbackRequest) (*pb.PreviewPermissionRollbackResponse, error) {
	if in == nil {
		return nil, historyError("历史ID无效")
	}
	out := &pb.PreviewPermissionRollbackResponse{}
	err := accessutil.PermissionRead(l.svcCtx.DB.WithContext(l.ctx), func(tx *gorm.DB) error {
		row, err := loadChange(tx, in.ID)
		if err != nil {
			return err
		}
		out.Change, err = changePB(row)
		if err != nil {
			return err
		}
		revision, err := accessutil.PermissionRevision(tx)
		if err != nil {
			return err
		}
		out.Version = rollbackVersion(row, revision)
		before, after, err := checkRollback(tx, row)
		if err != nil {
			out.Reason = err.Error()
			return nil
		}
		if err = tx.SavePoint("permission_rollback_preview").Error; err != nil {
			return err
		}
		validation := accessutil.ValidateProtectedPermissionChange(tx, func(tx *gorm.DB) error { return restoreGrant(tx, row.AuthorityID, row.Kind, before, after) })
		if err = tx.RollbackTo("permission_rollback_preview").Error; err != nil {
			return err
		}
		if validation != nil {
			out.Reason = validation.Error()
			return nil
		}
		out.Allowed = true
		return nil
	})
	return out, err
}
