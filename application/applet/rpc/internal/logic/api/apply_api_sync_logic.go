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
	err = accessutil.PermissionTransaction(l.svcCtx.DB.WithContext(l.ctx), func(tx *gorm.DB) error {
		var err error
		result, err = applyApiSync(tx, resources, in)
		return err
	})
	return result, err
}
