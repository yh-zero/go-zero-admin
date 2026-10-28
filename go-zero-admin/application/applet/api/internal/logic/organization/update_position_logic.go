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

type UpdatePositionLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 更新岗位
func NewUpdatePositionLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdatePositionLogic {
	return &UpdatePositionLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *UpdatePositionLogic) UpdatePosition(req *types.Position) (resp *types.MessageResponse, err error) {
	_, err = l.svcCtx.AppletOrganizationRPC.UpdatePosition(l.ctx, &pb.PositionRequest{Position: positionInput(req)})
	if err != nil {
		return nil, err
	}
	return &types.MessageResponse{Message: "更新成功"}, nil
}
