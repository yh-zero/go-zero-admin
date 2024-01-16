package casbin

import (
	"context"
	"fmt"
	"strconv"

	"github.com/zeromicro/go-zero/core/logx"
	"go-zero-admin/application/applet/api/internal/svc"
	"go-zero-admin/application/applet/api/internal/types"
)

type GetPathByAuthorityIdLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetPathByAuthorityIdLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetPathByAuthorityIdLogic {
	return &GetPathByAuthorityIdLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetPathByAuthorityIdLogic) GetPathByAuthorityId(req *types.GetPathByAuthorityIdRequest) (resp *types.GetPathByAuthorityIdResponse, err error) {

	csb := l.svcCtx.Config.CasbinConf.MustNewCasbinWithRedisWatcher(l.svcCtx.Config.DB.DataSource, l.svcCtx.Config.BizRedis)

	authorityId := strconv.FormatInt(req.AuthorityId, 10)
	casbinList := csb.GetFilteredPolicy(0, authorityId)
	fmt.Println("===== GetFilteredPolicy casbinList", casbinList)
	var casbinInfoList []types.CasbinInfo
	for _, v := range casbinList {
		casbinInfoList = append(casbinInfoList, types.CasbinInfo{
			Path:   v[1],
			Method: v[2],
		})
	}

	return &types.GetPathByAuthorityIdResponse{
		List: casbinInfoList,
	}, err
}

//func (l *GetPathByAuthorityIdLogic) GetPathByAuthorityId(req *types.GetPathByAuthorityIdRequest) (resp *types.GetPathByAuthorityIdResponse, err error) {
//	authorityId := ctxJwt.GetJwtDataAuthorityId(l.ctx)
//
//	getPathByAuthorityId, err := l.svcCtx.AppletCasbinRPC.GetPathByAuthorityId(l.ctx, &pb.GetPathByAuthorityIdRequest{AuthorityId: int64(authorityId)})
//	if err != nil {
//		logx.Errorf("l.svcCtx.AppletCasbinRPC.GetPathByAuthorityId err: %v", err)
//		return nil, err
//	}
//	var casbinInfoList []types.CasbinInfo
//
//	// 可以复制 也可以下面的方式遍历
//	//_ = copier.Copy(&casbinInfoList, getPathByAuthorityId.CasbinInfoList)
//
//	// 遍历原始切片，解引用指针对象并存入新的切片中
//	for _, item := range getPathByAuthorityId.CasbinInfoList {
//		casbinInfoList = append(casbinInfoList, types.CasbinInfo{
//			Path:   item.Path,
//			Method: item.Method,
//		})
//	}
//
//	return &types.GetPathByAuthorityIdResponse{
//		List: casbinInfoList,
//	}, err
//}
