package organizationlogic

import (
	"context"

	"go-zero-admin/application/applet/rpc/internal/svc"
	"go-zero-admin/application/applet/rpc/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

type UpdatePositionLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewUpdatePositionLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdatePositionLogic {
	return &UpdatePositionLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *UpdatePositionLogic) UpdatePosition(in *pb.PositionRequest) (*pb.NoDataResponse, error) {
	_, err := (store{ctx: l.ctx, db: l.svcCtx.DB.DB}).savePosition(in, false)
	return &pb.NoDataResponse{}, err
}
