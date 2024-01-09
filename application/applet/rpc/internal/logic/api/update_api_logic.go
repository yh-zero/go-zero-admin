package apilogic

import (
	"context"
	gormadapter "github.com/casbin/gorm-adapter/v3"
	"github.com/jinzhu/copier"
	"github.com/pkg/errors"
	"go-zero-admin/application/applet/rpc/internal/model"
	"gorm.io/gorm"

	"go-zero-admin/application/applet/rpc/internal/svc"
	"go-zero-admin/application/applet/rpc/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

type UpdateApiLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewUpdateApiLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateApiLogic {
	return &UpdateApiLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 更新api
func (l *UpdateApiLogic) UpdateApi(in *pb.UpdateApiRequest) (*pb.NoDataResponse, error) {
	var modelSysApi model.SysApi
	var sysApi model.SysApi
	_ = copier.Copy(&sysApi, in.SysApi)
	err := l.svcCtx.DB.Where("id = ?", sysApi.ID).First(&modelSysApi).Error
	if modelSysApi.Path != sysApi.Path || modelSysApi.Method != sysApi.Method {
		if !errors.Is(l.svcCtx.DB.Where("path = ? AND method = ?", sysApi.Path, sysApi.Method).First(&model.SysApi{}).Error, gorm.ErrRecordNotFound) {
			return nil, errors.New("存在相同api路径或者请求method一样")
		}
	}
	if err != nil {
		return nil, err
	} else {
		err = l.UpdateCasbinApi(modelSysApi.Path, sysApi.Path, modelSysApi.Method, sysApi.Method)
		if err != nil {
			return nil, err
		}
		sysApi.DeletedAt.Valid = false
		err = l.svcCtx.DB.Save(&sysApi).Error
	}

	return &pb.NoDataResponse{}, nil
}

func (l *UpdateApiLogic) UpdateCasbinApi(oldPath string, newPath string, oldMethod string, newMethod string) error {
	err := l.svcCtx.DB.Model(&gormadapter.CasbinRule{}).Where("v1 = ? AND v2 = ?", oldPath, oldMethod).Updates(map[string]interface{}{
		"v1": newPath,
		"v2": newMethod,
	}).Error
	csb := l.svcCtx.Config.CasbinConf.MustNewCasbinWithRedisWatcher(l.svcCtx.Config.DB.DataSource, l.svcCtx.Config.BizRedis)
	err = csb.LoadPolicy()
	if err != nil {
		return err
	}
	return err
}
