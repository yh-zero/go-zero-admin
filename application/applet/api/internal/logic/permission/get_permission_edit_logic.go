// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package permission

import (
	"context"
	"github.com/jinzhu/copier"
	"go-zero-admin/application/applet/rpc/pb"

	"go-zero-admin/application/applet/api/internal/svc"
	"go-zero-admin/application/applet/api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetPermissionEditLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetPermissionEditLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetPermissionEditLogic {
	return &GetPermissionEditLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetPermissionEditLogic) GetPermissionEdit(req *types.GetPermissionEditRequest) (resp *types.GetPermissionEditResponse, err error) {
	out, err := l.svcCtx.AppletPermissionRPC.GetPermissionEdit(l.ctx, &pb.GetPermissionEditRequest{AuthorityId: req.AuthorityId, Kind: req.Kind})
	if err != nil {
		return nil, err
	}
	resp = &types.GetPermissionEditResponse{}
	err = copier.Copy(resp, out)
	return resp, err
}
