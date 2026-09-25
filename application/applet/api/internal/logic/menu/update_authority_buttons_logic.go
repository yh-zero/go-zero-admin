package menu

import (
	"context"
	"github.com/zeromicro/go-zero/core/logx"
	"go-zero-admin/application/applet/api/internal/svc"
	"go-zero-admin/application/applet/api/internal/types"
	"go-zero-admin/application/applet/rpc/pb"
)

type UpdateAuthorityButtonsLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewUpdateAuthorityButtonsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateAuthorityButtonsLogic {
	return &UpdateAuthorityButtonsLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

func (l *UpdateAuthorityButtonsLogic) UpdateAuthorityButtons(req *types.UpdateAuthorityButtonsRequest) (*types.MessageResponse, error) {
	_, err := l.svcCtx.AppletMenuRPC.UpdateAuthorityButtons(l.ctx, &pb.UpdateAuthorityButtonsRequest{AuthorityId: req.AuthorityId, MenuBtnIds: req.MenuBtnIds})
	if err != nil {
		return nil, err
	}
	return &types.MessageResponse{Message: "按钮授权已保存"}, nil
}
