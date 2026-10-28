package menulogic

import (
	"context"
	"go-zero-admin/application/applet/rpc/internal/logic/accessutil"
	"gorm.io/gorm"

	"go-zero-admin/application/applet/rpc/internal/svc"
	"go-zero-admin/application/applet/rpc/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

type PreviewMenuMoveLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewPreviewMenuMoveLogic(ctx context.Context, svcCtx *svc.ServiceContext) *PreviewMenuMoveLogic {
	return &PreviewMenuMoveLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *PreviewMenuMoveLogic) PreviewMenuMove(in *pb.PreviewMenuMoveRequest) (*pb.PreviewMenuMoveResponse, error) {
	var out *pb.PreviewMenuMoveResponse
	err := accessutil.PermissionRead(l.svcCtx.DB.WithContext(l.ctx), func(tx *gorm.DB) error { var err error; out, err = menuMovePreview(tx, in.ID, in.ParentId); return err })
	return out, err
}
