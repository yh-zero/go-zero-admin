package casbinlogic

import (
	"context"
	"strconv"

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

	authorityId := strconv.FormatInt(in.AuthorityId, 10)

	if _, err = l.svcCtx.Casbin.RemoveFilteredPolicy(0, authorityId); err != nil {
		return nil, errors.Wrap(err, "删除策略失败")
	}

	// 做权限去重处理
	deduplicateMap := make(map[string]bool)
	rules := make([][]string, 0, len(modelSysApis))
	for _, v := range modelSysApis {
		key := authorityId + v.Path + v.Method
		if _, ok := deduplicateMap[key]; !ok {
			deduplicateMap[key] = true
			rules = append(rules, []string{authorityId, v.Path, v.Method})
		}
	}

	if len(rules) > 0 {
		success, err := l.svcCtx.Casbin.AddPolicies(rules)
		if err != nil || !success {
			return nil, errors.New("存在相同api,添加失败,请联系管理员")
		}
	}

	return &pb.UpdateCasbinDataByApiIdsResponse{SysApis: pbSysApis}, nil
}
