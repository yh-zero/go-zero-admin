package menu

import (
	"context"
	"github.com/zeromicro/go-zero/core/logx"
	"go-zero-admin/application/applet/api/internal/svc"
	"go-zero-admin/application/applet/api/internal/types"
	"go-zero-admin/application/applet/rpc/pb"
)

type GetAuthorityButtonsLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetAuthorityButtonsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetAuthorityButtonsLogic {
	return &GetAuthorityButtonsLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

func (l *GetAuthorityButtonsLogic) GetAuthorityButtons(req *types.GetAuthorityButtonsRequest) (*types.GetAuthorityButtonsResponse, error) {
	result, err := l.svcCtx.AppletMenuRPC.GetAuthorityButtons(l.ctx, &pb.GetAuthorityButtonsRequest{AuthorityId: req.AuthorityId})
	if err != nil {
		return nil, err
	}
	return &types.GetAuthorityButtonsResponse{MenuBtnIds: result.MenuBtnIds}, nil
}
