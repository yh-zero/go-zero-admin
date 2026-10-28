package menulogic

import (
	"context"
	"github.com/zeromicro/go-zero/core/logx"
	"go-zero-admin/application/applet/rpc/internal/logic/accessutil"
	"go-zero-admin/application/applet/rpc/internal/svc"
	"go-zero-admin/application/applet/rpc/pb"
	"gorm.io/gorm"
)

type AddMenuBaseLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewAddMenuBaseLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AddMenuBaseLogic {
	return &AddMenuBaseLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

func (l *AddMenuBaseLogic) AddMenuBase(in *pb.AddMenuBaseRequest) (*pb.NoDataResponse, error) {
	err := accessutil.AdminMenuTransaction(l.svcCtx.DB.WithContext(l.ctx), func(tx *gorm.DB) error {
		if in.SysBaseMenu != nil {
			in.SysBaseMenu.ID = 0
		}
		if err := validateMenu(tx, in.SysBaseMenu); err != nil {
			return err
		}
		menu := menuModel(in.SysBaseMenu)
		if err := tx.Omit("Parameters", "MenuBtn", "SysAuthoritys").Create(&menu).Error; err != nil {
			return err
		}
		return saveMenuRelations(tx, menu.ID, in.SysBaseMenu)
	})
	return &pb.NoDataResponse{}, accessutil.FriendlyDuplicate(err)
}
