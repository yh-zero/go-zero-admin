package apilogic

import (
	"context"
	gormadapter "github.com/casbin/gorm-adapter/v3"
	"github.com/zeromicro/go-zero/core/logx"
	"go-zero-admin/application/applet/rpc/internal/logic/accessutil"
	"go-zero-admin/application/applet/rpc/internal/model"
	"go-zero-admin/application/applet/rpc/internal/svc"
	"go-zero-admin/application/applet/rpc/pb"
	"go-zero-admin/pkg/result/xerr"
	"gorm.io/gorm"
)

type UpdateApiLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewUpdateApiLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateApiLogic {
	return &UpdateApiLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

func (l *UpdateApiLogic) UpdateApi(in *pb.UpdateApiRequest) (*pb.NoDataResponse, error) {
	if in.SysApi == nil {
		return nil, xerr.NewErrCodeMsg(xerr.REUQEST_PARAM_ERROR, "API不能为空")
	}
	path, method, err := accessutil.API(in.SysApi.Path, in.SysApi.Method)
	if err != nil {
		return nil, err
	}
	err = accessutil.PolicyTransaction(l.svcCtx, func(tx *gorm.DB) error {
		var old model.SysApi
		if err := accessutil.RequireID(tx, &old, in.SysApi.ID); err != nil {
			return err
		}
		if err := accessutil.Unique(tx, &model.SysApi{}, "id <> ? AND path = ? AND method = ?", old.ID, path, method); err != nil {
			return err
		}
		if old.Path != path || old.Method != method {
			var rules []gormadapter.CasbinRule
			if err := tx.Where("ptype = ? AND v1 = ? AND v2 = ?", "p", old.Path, old.Method).Find(&rules).Error; err != nil {
				return err
			}
			for _, rule := range rules {
				var count int64
				if err := tx.Model(&gormadapter.CasbinRule{}).Where("ptype = ? AND v0 = ? AND v1 = ? AND v2 = ?", "p", rule.V0, path, method).Count(&count).Error; err != nil {
					return err
				}
				if count > 0 {
					if err := tx.Delete(&rule).Error; err != nil {
						return err
					}
				} else {
					if err := tx.Model(&rule).Updates(map[string]interface{}{"v1": path, "v2": method}).Error; err != nil {
						return err
					}
				}
			}
		}
		return tx.Model(&old).Updates(map[string]interface{}{"path": path, "method": method, "api_group": in.SysApi.ApiGroup, "description": in.SysApi.Description}).Error
	})
	return &pb.NoDataResponse{}, err
}
