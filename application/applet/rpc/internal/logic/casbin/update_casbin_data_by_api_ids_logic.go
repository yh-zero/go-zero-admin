package casbinlogic

import (
	"context"
	"github.com/jinzhu/copier"
	"go-zero-admin/application/applet/rpc/internal/model"
	"go-zero-admin/application/applet/rpc/internal/svc"
	"go-zero-admin/application/applet/rpc/pb"

	"github.com/pkg/errors"
	"github.com/zeromicro/go-zero/core/logx"
)

type UpdateCasbinDataByApiIdsLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewUpdateCasbinDataByApiIdsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateCasbinDataByApiIdsLogic {
	return &UpdateCasbinDataByApiIdsLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 更新一个角色的对应的casbin数据 用api的ids 查数据
func (l *UpdateCasbinDataByApiIdsLogic) UpdateCasbinDataByApiIds(in *pb.UpdateCasbinDataByApiIdsRequest) (*pb.UpdateCasbinDataByApiIdsResponse, error) {
	// 根据 ApiIds 获取对应的数据
	var modelSysApis []model.SysApi
	err := l.svcCtx.DB.Where("id in ?", in.ApiIds).Find(&modelSysApis).Error
	if err != nil {
		return nil, errors.Wrap(err, "获取数据失败")
	}

	var pbSysApis []*pb.SysApi
	_ = copier.Copy(&pbSysApis, modelSysApis)

	return &pb.UpdateCasbinDataByApiIdsResponse{SysApis: pbSysApis}, err

}

//
//func (l *UpdateCasbinDataByApiIdsLogic) UpdateCasbinDataByApiIds(in *pb.UpdateCasbinDataByApiIdsRequest) (*pb.NoDataResponse, error) {
//	// 根据 ApiIds 获取对应的数据
//	var modelSysApis []model.SysApi
//	err := l.svcCtx.DB.Where("id in ?", in.ApiIds).Find(&modelSysApis).Error
//	fmt.Println("========= modelSysApis", modelSysApis)
//	if err != nil {
//		return nil, err
//	}
//
//	authorityId := strconv.FormatInt(in.AuthorityId, 10)
//
//	csb := l.svcCtx.Config.CasbinConf.MustNewCasbinWithRedisWatcher(l.svcCtx.Config.DB.DataSource, l.svcCtx.Config.BizRedis)
//	_, err = csb.RemoveFilteredPolicy(0, authorityId)
//
//	if err != nil {
//		return nil, err
//	}
//	//做权限去重处理
//	rules := [][]string{}
//	deduplicateMap := make(map[string]bool)
//	for _, v := range modelSysApis {
//		key := authorityId + v.Path + v.Method
//		if _, ok := deduplicateMap[key]; !ok {
//			deduplicateMap[key] = true
//			rules = append(rules, []string{authorityId, v.Path, v.Method})
//		}
//	}
//
//	success, err := csb.AddPolicies(rules)
//
//	if !success {
//		return nil, errors.New("存在相同api,添加失败,请联系管理员")
//	}
//
//	return &pb.NoDataResponse{}, err
//}
