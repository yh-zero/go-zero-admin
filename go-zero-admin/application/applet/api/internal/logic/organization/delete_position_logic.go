// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package organization

import (
	"context"

	"go-zero-admin/application/applet/api/internal/svc"
	"go-zero-admin/application/applet/api/internal/types"
	"go-zero-admin/application/applet/rpc/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

type DeletePositionLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 删除未被使用的岗位
func NewDeletePositionLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeletePositionLogic {
	return &DeletePositionLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *DeletePositionLogic) DeletePosition(req *types.OrganizationIDRequest) (resp *types.MessageResponse, err error) {
	_, err = l.svcCtx.AppletOrganizationRPC.DeletePosition(l.ctx, &pb.OrganizationIDRequest{ID: req.Id})
	if err != nil {
		return nil, err
	}
	return &types.MessageResponse{Message: "删除成功"}, nil
}
