package menu

import (
	"context"
	"fmt"
	"github.com/jinzhu/copier"
	"go-zero-admin/application/applet/rpc/pb"
	"go-zero-admin/pkg/ctxJwt"

	"go-zero-admin/application/applet/api/internal/svc"
	"go-zero-admin/application/applet/api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetMenuAuthorityLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetMenuAuthorityLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetMenuAuthorityLogic {
	return &GetMenuAuthorityLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetMenuAuthorityLogic) GetMenuAuthority(req *types.GetMenuAuthorityRequest) (resp *types.GetMenuAuthorityResponse, err error) {
	authorityId := ctxJwt.GetJwtDataAuthorityId(l.ctx)
	authority, err := l.svcCtx.AppletMenuRPC.GetMenuAuthority(l.ctx, &pb.GetMenuAuthorityRequest{AuthorityId: int64(authorityId)})
	fmt.Println("------------ authority", authority)
	if err != nil {
		return nil, err
	}
	//var sysMenus []types.SysMenu
	//for _, v := range authority.SysMenuList {
	//	fmt.Println("---- v", v.)
	//
	//}

	var typesGetMenuAuthorityResponse types.GetMenuAuthorityResponse
	var sysMenus []types.SysMenu
	_ = copier.Copy(&sysMenus, authority.SysMenuList)
	fmt.Println("------------ sysMenus", sysMenus)
	typesGetMenuAuthorityResponse.SysMenuList = sysMenus

	return &typesGetMenuAuthorityResponse, err
}
