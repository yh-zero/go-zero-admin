package authority

import (
	"context"

	"go-zero-admin/application/applet/api/internal/svc"
	"go-zero-admin/application/applet/api/internal/types"
	"go-zero-admin/application/applet/rpc/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

type AddAuthorityMenuLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewAddAuthorityMenuLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AddAuthorityMenuLogic {
	return &AddAuthorityMenuLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *AddAuthorityMenuLogic) AddAuthorityMenu(req *types.AddAuthorityMenuRequest) (resp *types.MessageResponse, err error) {
	var authorityMenu pb.AddAuthorityMenuRequest
	authorityMenu.AuthorityId = req.AuthorityId
	authorityMenu.MenuIds = req.MenuIds

	_, err = l.svcCtx.AppletAuthorityRPC.AddAuthorityMenu(l.ctx, &authorityMenu)
	if err != nil {
		return nil, err
	}

	return &types.MessageResponse{Message: "添加成功"}, nil
}
