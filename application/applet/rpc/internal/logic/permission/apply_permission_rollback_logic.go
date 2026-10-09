package permissionlogic

import (
	"context"

	"go-zero-admin/application/applet/rpc/internal/logic/accessutil"
	"go-zero-admin/pkg/audit"
	"gorm.io/gorm"
	"strconv"

	"go-zero-admin/application/applet/rpc/internal/svc"
	"go-zero-admin/application/applet/rpc/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

type ApplyPermissionRollbackLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewApplyPermissionRollbackLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ApplyPermissionRollbackLogic {
	return &ApplyPermissionRollbackLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *ApplyPermissionRollbackLogic) ApplyPermissionRollback(in *pb.ApplyPermissionRollbackRequest) (*pb.NoDataResponse, error) {
	if in == nil || in.Version == "" {
		return nil, historyError("缺少回滚预览，请重新确认")
	}
	err := accessutil.PolicyTransaction(l.ctx, l.svcCtx, func(tx *gorm.DB) error {
		if err := accessutil.RequirePermissionRevision(tx, in.ExpectedRevision); err != nil {
			return err
		}
		row, err := loadChange(tx, in.ID)
		if err != nil {
			return err
		}
		revision, err := accessutil.PermissionRevision(tx)
		if err != nil {
			return err
		}
		if in.Version != rollbackVersion(row, revision) {
			return historyError("回滚预览已变更，请重新确认")
		}
		before, after, err := checkRollback(tx, row)
		if err != nil {
			return err
		}
		if err = restoreGrant(tx, row.AuthorityID, row.Kind, before, after); err != nil {
			return err
		}
		return audit.Record(l.ctx, tx, audit.Event{Module: "permission", Action: "rollback", Object: strconv.FormatInt(row.ID, 10)})
	})
	return &pb.NoDataResponse{}, err
}
