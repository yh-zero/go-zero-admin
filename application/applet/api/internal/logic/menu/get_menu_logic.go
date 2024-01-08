package menu

import (
	"context"
	"github.com/jinzhu/copier"
	"go-zero-admin/application/applet/rpc/pb"
	"go-zero-admin/pkg/ctxJwt"
	"go-zero-admin/pkg/result/xerr"

	"go-zero-admin/application/applet/api/internal/svc"
	"go-zero-admin/application/applet/api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetMenuLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetMenuLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetMenuLogic {
	return &GetMenuLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetMenuLogic) GetMenu(req *types.GetMenuRequest) (resp *types.GetMenuResponse, err error) {
	authorityId := ctxJwt.GetJwtDataAuthorityId(l.ctx)
	menus, err := l.svcCtx.AppletMenuRPC.GetMenuTree(l.ctx, &pb.GetMenuTreeRequest{AuthorityId: int64(authorityId)})
	if err != nil {
		logx.Errorf("MenuTreeGet err: %v", err)
		return nil, xerr.NewErrCodeMsg(900002, "内部错误：MenuTreeGet")
	}

	var getMenuResponse types.GetMenuResponse
	_ = copier.Copy(&getMenuResponse.Menus, menus.SysMenu)
	return &getMenuResponse, err
}
