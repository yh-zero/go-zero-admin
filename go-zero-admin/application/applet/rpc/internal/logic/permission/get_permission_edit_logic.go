package permissionlogic

import (
	"context"
	"go-zero-admin/application/applet/rpc/internal/logic/accessutil"
	"gorm.io/gorm"

	"go-zero-admin/application/applet/rpc/internal/svc"
	"go-zero-admin/application/applet/rpc/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetPermissionEditLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetPermissionEditLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetPermissionEditLogic {
	return &GetPermissionEditLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *GetPermissionEditLogic) GetPermissionEdit(in *pb.GetPermissionEditRequest) (*pb.GetPermissionEditResponse, error) {
	var out *pb.GetPermissionEditResponse
	err := accessutil.PermissionRead(l.svcCtx.DB.WithContext(l.ctx), func(tx *gorm.DB) error {
		var err error
		out, err = accessutil.LoadPermissionEdit(tx, in.AuthorityId, in.Kind)
		return err
	})
	return out, err
}
