// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package menu

import (
	"context"
	"github.com/jinzhu/copier"
	"go-zero-admin/application/applet/rpc/pb"

	"go-zero-admin/application/applet/api/internal/svc"
	"go-zero-admin/application/applet/api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type PreviewMenuMoveLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewPreviewMenuMoveLogic(ctx context.Context, svcCtx *svc.ServiceContext) *PreviewMenuMoveLogic {
	return &PreviewMenuMoveLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *PreviewMenuMoveLogic) PreviewMenuMove(req *types.PreviewMenuMoveRequest) (resp *types.PreviewMenuMoveResponse, err error) {
	out, err := l.svcCtx.AppletMenuRPC.PreviewMenuMove(l.ctx, &pb.PreviewMenuMoveRequest{ID: req.ID, ParentId: req.ParentId})
	if err != nil {
		return nil, err
	}
	resp = &types.PreviewMenuMoveResponse{}
	err = copier.Copy(resp, out)
	return resp, err
}
