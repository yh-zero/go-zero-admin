package casbinlogic

import (
	"context"
	"github.com/zeromicro/go-zero/core/logx"
	"go-zero-admin/application/applet/rpc/internal/logic/accessutil"
	"go-zero-admin/application/applet/rpc/internal/svc"
	"go-zero-admin/application/applet/rpc/pb"
	"gorm.io/gorm"
)

type GetPathByAuthorityIdLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetPathByAuthorityIdLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetPathByAuthorityIdLogic {
	return &GetPathByAuthorityIdLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

func (l *GetPathByAuthorityIdLogic) GetPathByAuthorityId(in *pb.GetPathByAuthorityIdRequest) (*pb.GetPathByAuthorityIdResponse, error) {
	var out *pb.GetPathByAuthorityIdResponse
	err := accessutil.PermissionRead(l.svcCtx.DB.WithContext(l.ctx), func(tx *gorm.DB) error {
		e, err := accessutil.LoadPermissionEdit(tx, in.AuthorityId, "api")
		if err != nil {
			return err
		}
		out = &pb.GetPathByAuthorityIdResponse{Revision: e.Revision, CasbinInfoList: e.Policies}
		return nil
	})
	return out, err
}
