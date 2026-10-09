// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package permission

import (
	"context"
	"github.com/jinzhu/copier"
	"go-zero-admin/application/applet/rpc/pb"
	"go-zero-admin/pkg/ctxJwt"

	"go-zero-admin/application/applet/api/internal/svc"
	"go-zero-admin/application/applet/api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetPermissionSnapshotLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetPermissionSnapshotLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetPermissionSnapshotLogic {
	return &GetPermissionSnapshotLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetPermissionSnapshotLogic) GetPermissionSnapshot(req *types.GetPermissionSnapshotRequest) (resp *types.GetPermissionSnapshotResponse, err error) {
	actor := ctxJwt.GetJwtData(l.ctx)
	out, err := l.svcCtx.AppletPermissionRPC.GetPermissionSnapshot(l.ctx, &pb.GetPermissionSnapshotRequest{Actor: &pb.SessionRequest{UserID: actor.ID, AuthorityId: actor.AuthorityId, SessionVersion: actor.SessionVersion, SessionID: actor.SessionID}})
	if err != nil {
		return nil, err
	}
	resp = &types.GetPermissionSnapshotResponse{}
	err = copier.Copy(resp, out)
	return resp, err
}
