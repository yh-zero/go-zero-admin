package apilogic

import (
	"context"

	"go-zero-admin/application/applet/rpc/internal/model"
	"go-zero-admin/application/applet/rpc/internal/svc"
	"go-zero-admin/application/applet/rpc/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

//var casbin = casbinlogic.UpdateCasbinDataLogic{}

type DeleteApisByIdsLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewDeleteApisByIdsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteApisByIdsLogic {
	return &DeleteApisByIdsLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 删除多条api
func (l *DeleteApisByIdsLogic) DeleteApisByIds(in *pb.DeleteApisByIdsRequest) (*pb.NoDataResponse, error) {
	var apis []model.SysApi
	err := l.svcCtx.DB.Find(&apis, "id in ?", in.Ids).Delete(&apis).Error
	if err != nil {
		return nil, err
	}
	// 同步删除casbin对应策略 避免死策略残留
	for _, sysApi := range apis {
		if _, err = l.svcCtx.Casbin.RemoveFilteredPolicy(1, sysApi.Path, sysApi.Method); err != nil {
			logx.WithContext(l.ctx).Errorf("DeleteApisByIds RemoveFilteredPolicy err: %v path: %s method: %s", err, sysApi.Path, sysApi.Method)
			return nil, err
		}
	}
	return &pb.NoDataResponse{}, nil
}
