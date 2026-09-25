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

type DeleteApisByIdsLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewDeleteApisByIdsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteApisByIdsLogic {
	return &DeleteApisByIdsLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

func (l *DeleteApisByIdsLogic) DeleteApisByIds(in *pb.DeleteApisByIdsRequest) (*pb.NoDataResponse, error) {
	ids, err := accessutil.UniqueIDs(in.Ids)
	if err != nil {
		return nil, err
	}
	if len(ids) == 0 {
		return nil, xerr.NewErrCodeMsg(xerr.REUQEST_PARAM_ERROR, "请选择要删除的API")
	}
	err = accessutil.PolicyTransaction(l.svcCtx, func(tx *gorm.DB) error {
		var apis []model.SysApi
		if err := tx.Where("id IN ?", ids).Find(&apis).Error; err != nil {
			return err
		}
		if len(apis) != len(ids) {
			return xerr.NewErrCodeMsg(xerr.REUQEST_PARAM_ERROR, "部分API不存在，删除未执行")
		}
		for _, api := range apis {
			if err := tx.Where("ptype = ? AND v1 = ? AND v2 = ?", "p", api.Path, api.Method).Delete(&gormadapter.CasbinRule{}).Error; err != nil {
				return err
			}
		}
		return tx.Unscoped().Where("id IN ?", ids).Delete(&model.SysApi{}).Error
	})
	return &pb.NoDataResponse{}, err
}
