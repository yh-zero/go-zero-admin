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
func (l *DeleteApisByIdsLogic) DeleteApisByIds(in *pb.DeleteApisByIdsRequest) (*pb.DeleteApisByIdsResponse, error) {
	csb := l.svcCtx.Config.CasbinConf.MustNewCasbinWithRedisWatcher(l.svcCtx.Config.DB.DataSource, l.svcCtx.Config.BizRedis)

	var apis []model.SysApi
	err := l.svcCtx.DB.Find(&apis, "id in ?", in.Ids).Delete(&apis).Error
	if err != nil {
		return nil, err
	} else {
		for _, sysApi := range apis {
			_, err = csb.RemoveFilteredPolicy(1, sysApi.Path, sysApi.Method)
			//_, err = casbin.ClearCasbin(1, sysApi.Path, sysApi.Method)
			if err != nil {
				return nil, err
			}
		}
	}
	return &pb.DeleteApisByIdsResponse{}, nil
}
