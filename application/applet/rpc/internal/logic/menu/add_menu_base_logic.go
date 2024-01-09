package menulogic

import (
	"context"
	"github.com/jinzhu/copier"
	"github.com/pkg/errors"
	"go-zero-admin/application/applet/rpc/internal/model"
	"gorm.io/gorm"

	"go-zero-admin/application/applet/rpc/internal/svc"
	"go-zero-admin/application/applet/rpc/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

type AddMenuBaseLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewAddMenuBaseLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AddMenuBaseLogic {
	return &AddMenuBaseLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 添加系统基础菜单列表
func (l *AddMenuBaseLogic) AddMenuBase(in *pb.AddMenuBaseRequest) (*pb.NoDataResponse, error) {
	if !errors.Is(l.svcCtx.DB.Where("name = ?", in.SysBaseMenu.Name).First(&model.SysBaseMenu{}).Error, gorm.ErrRecordNotFound) {
		//return nil, errors.Wrapf(xerr.NewErrCode(xerr.REPEAT_NAME_ERROR), "添加菜单名字重复错误")
		return nil, errors.New("存在重复name，请修改name")
	}
	var baseMenu model.SysBaseMenu
	_ = copier.Copy(&baseMenu, in.SysBaseMenu)
	return &pb.NoDataResponse{}, l.svcCtx.DB.Omit("deleted_at").Create(&baseMenu).Error
}
