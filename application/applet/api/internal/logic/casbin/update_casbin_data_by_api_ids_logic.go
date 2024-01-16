package casbin

import (
	"context"
	"fmt"
	"github.com/jinzhu/copier"
	"github.com/pkg/errors"
	"go-zero-admin/application/applet/api/internal/types"
	"go-zero-admin/application/applet/rpc/pb"
	"strconv"

	"github.com/zeromicro/go-zero/core/logx"
	"go-zero-admin/application/applet/api/internal/svc"
)

type UpdateCasbinDataByApiIdsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewUpdateCasbinDataByApiIdsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateCasbinDataByApiIdsLogic {
	return &UpdateCasbinDataByApiIdsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *UpdateCasbinDataByApiIdsLogic) UpdateCasbinDataByApiIds(req *types.UpdateCasbinDataByApiIdsRequest) (resp *types.MessageResponse, err error) {
	var rpcReq pb.UpdateCasbinDataByApiIdsRequest
	_ = copier.Copy(&rpcReq, req)
	apiList, err := l.svcCtx.AppletCasbinRPC.UpdateCasbinDataByApiIds(l.ctx, &rpcReq)
	if err != nil {
		logx.Errorf("l.svcCtx.AppletCasbinRPC.UpdateCasbinDataByApiIds err: %v", err)
		return nil, err
	}

	authorityId := strconv.FormatInt(req.AuthorityId, 10)

	// 权限去重处理
	deduplicateMap := make(map[string]bool)
	rules := make([][]string, 0, len(apiList.SysApis))
	for _, v := range apiList.SysApis {
		key := authorityId + v.Path + v.Method
		if _, ok := deduplicateMap[key]; !ok {
			deduplicateMap[key] = true
			rules = append(rules, []string{authorityId, v.Path, v.Method})
		}
	}

	csb := l.svcCtx.Config.CasbinConf.MustNewCasbinWithRedisWatcher(l.svcCtx.Config.DB.DataSource, l.svcCtx.Config.BizRedis)
	if _, err = csb.RemoveFilteredPolicy(0, authorityId); err != nil {
		return nil, errors.Wrap(err, "删除策略失败")
	}
	fmt.Println("=========shanchu")
	if len(rules) > 0 {
		success, err := csb.AddPolicies(rules)
		if !success {
			return nil, errors.New("存在相同api,添加失败,请联系管理员")
		}

		fmt.Println("---------- success", success)
		fmt.Println("---------- err", err)
		fmt.Println("---------- rules", rules)

	}
	fmt.Println("=========add")

	return &types.MessageResponse{
		Message: "修改成功！",
	}, err
}

//func (l *UpdateCasbinDataByApiIdsLogic) UpdateCasbinDataByApiIds(req *types.UpdateCasbinDataByApiIdsRequest) (resp *types.MessageResponse, err error) {
//	var rpcReq pb.UpdateCasbinDataByApiIdsRequest
//	_ = copier.Copy(&rpcReq, req)
//	_, err = l.svcCtx.AppletCasbinRPC.UpdateCasbinDataByApiIds(l.ctx, &rpcReq)
//	if err != nil {
//		logx.Errorf("l.svcCtx.AppletCasbinRPC.UpdateCasbinDataByApiIds err: %v", err)
//		return nil, err
//	}
//
//	return &types.MessageResponse{
//		Message: "修改成功！",
//	}, nil
//}
