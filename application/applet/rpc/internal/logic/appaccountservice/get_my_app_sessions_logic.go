package appaccountservicelogic

import (
	"context"

	"go-zero-admin/application/applet/rpc/internal/svc"
	"go-zero-admin/application/applet/rpc/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetMyAppSessionsLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetMyAppSessionsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetMyAppSessionsLogic {
	return &GetMyAppSessionsLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *GetMyAppSessionsLogic) GetMyAppSessions(in *pb.AppAccessRequest) (*pb.AppSessionListResponse, error) {
	return mySessions(l.ctx, l.svcCtx, in)
}
