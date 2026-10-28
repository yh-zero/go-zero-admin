package menulogic

import (
	"context"
	"github.com/zeromicro/go-zero/core/logx"
	"go-zero-admin/application/applet/rpc/internal/logic/accessutil"
	"go-zero-admin/application/applet/rpc/internal/svc"
	"go-zero-admin/application/applet/rpc/pb"
	"gorm.io/gorm"
	"strconv"
)

type GetMenuAuthorityLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetMenuAuthorityLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetMenuAuthorityLogic {
	return &GetMenuAuthorityLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

func (l *GetMenuAuthorityLogic) GetMenuAuthority(in *pb.GetMenuAuthorityRequest) (*pb.GetMenuAuthorityResponse, error) {
	out := &pb.GetMenuAuthorityResponse{SysMenuList: []*pb.SysMenu{}}
	err := accessutil.PermissionRead(l.svcCtx.DB.WithContext(l.ctx), func(tx *gorm.DB) error {
		e, err := accessutil.LoadPermissionEdit(tx, in.AuthorityId, "menu")
		if err != nil {
			return err
		}
		out.Revision = e.Revision
		selected := map[int64]bool{}
		for _, id := range e.MenuIds {
			selected[id] = true
		}
		var visit func([]*pb.SysBaseMenu)
		visit = func(list []*pb.SysBaseMenu) {
			for _, m := range list {
				children := m.Children
				visit(children)
				if selected[m.ID] {
					m.Children = nil
					out.SysMenuList = append(out.SysMenuList, &pb.SysMenu{SysBaseMenu: m, ID: m.ID, MenuId: strconv.FormatInt(m.ID, 10), AuthorityId: in.AuthorityId})
				}
			}
		}
		visit(e.Menus)
		return nil
	})
	return out, err
}
