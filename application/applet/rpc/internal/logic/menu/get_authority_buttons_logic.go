package menulogic

import (
	"context"
	"github.com/zeromicro/go-zero/core/logx"
	"go-zero-admin/application/applet/rpc/internal/logic/accessutil"
	"go-zero-admin/application/applet/rpc/internal/svc"
	"go-zero-admin/application/applet/rpc/pb"
	"gorm.io/gorm"
)

type GetAuthorityButtonsLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetAuthorityButtonsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetAuthorityButtonsLogic {
	return &GetAuthorityButtonsLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

func (l *GetAuthorityButtonsLogic) GetAuthorityButtons(in *pb.GetAuthorityButtonsRequest) (*pb.GetAuthorityButtonsResponse, error) {
	var out *pb.GetAuthorityButtonsResponse
	err := accessutil.PermissionRead(l.svcCtx.DB.WithContext(l.ctx), func(tx *gorm.DB) error {
		e, err := accessutil.LoadPermissionEdit(tx, in.AuthorityId, "button")
		if err != nil {
			return err
		}
		out = &pb.GetAuthorityButtonsResponse{Revision: e.Revision, MenuBtnIds: e.MenuBtnIds}
		return nil
	})
	return out, err
}
