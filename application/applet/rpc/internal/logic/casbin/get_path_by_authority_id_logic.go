package casbinlogic

import (
	"context"
	"strconv"

	"go-zero-admin/application/applet/rpc/internal/svc"
	"go-zero-admin/application/applet/rpc/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetPathByAuthorityIdLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetPathByAuthorityIdLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetPathByAuthorityIdLogic {
	return &GetPathByAuthorityIdLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 根据角色id获取对应的casbin数据
func (l *GetPathByAuthorityIdLogic) GetPathByAuthorityId(in *pb.GetPathByAuthorityIdRequest) (*pb.GetPathByAuthorityIdResponse, error) {
	csb := l.svcCtx.Config.CasbinConf.MustNewCasbinWithRedisWatcher(l.svcCtx.Config.DB.DataSource, l.svcCtx.Config.BizRedis)

	authorityId := strconv.Itoa(int(in.AuthorityId))
	list := csb.GetFilteredPolicy(0, authorityId)
	//var casbinInfo []pb.CasbinInfo
	var pbGetPathByAuthorityId pb.GetPathByAuthorityIdResponse
	for _, v := range list {
		pbGetPathByAuthorityId.CasbinInfoList = append(pbGetPathByAuthorityId.CasbinInfoList, &pb.CasbinInfo{
			Path:   v[1],
			Method: v[2],
		})
	}
	return &pbGetPathByAuthorityId, nil
}
