package apilogic

import (
	"context"

	"go-zero-admin/application/applet/rpc/internal/logic/accessutil"
	"go-zero-admin/application/applet/rpc/internal/svc"
	"go-zero-admin/application/applet/rpc/pb"
	"go-zero-admin/data/api/generated"

	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
)

type ApplyApiSyncLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewApplyApiSyncLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ApplyApiSyncLogic {
	return &ApplyApiSyncLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}
func (l *ApplyApiSyncLogic) ApplyApiSync(in *pb.ApplyApiSyncRequest) (*pb.ApplyApiSyncResponse, error) {
	resources, err := readSwaggerResources(generated.Swagger)
	if err != nil {
		return nil, err
	}
	var result *pb.ApplyApiSyncResponse
	err = l.svcCtx.DB.WithContext(l.ctx).Transaction(func(tx *gorm.DB) error {
		// Serialize concurrent syncs without changing role or policy data.
		if err := accessutil.LockAdminGuard(tx); err != nil {
			return err
		}
		var err error
		result, err = applyApiSync(tx, resources, in)
		return err
	})
	return result, err
}
