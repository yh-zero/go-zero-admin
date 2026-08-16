package casbinlogic

import (
	"context"
	"strconv"

	"go-zero-admin/application/applet/rpc/internal/svc"
	"go-zero-admin/application/applet/rpc/pb"

	"github.com/pkg/errors"
	"github.com/zeromicro/go-zero/core/logx"
)

type UpdateCasbinDataLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewUpdateCasbinDataLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateCasbinDataLogic {
	return &UpdateCasbinDataLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 更新一个角色的对应的casbin数据
func (l *UpdateCasbinDataLogic) UpdateCasbinData(in *pb.UpdateCasbinDataRequest) (*pb.NoDataResponse, error) {
	authorityId := strconv.FormatInt(in.AuthorityId, 10)

	// 做权限去重处理
	deduplicateMap := make(map[string]bool)
	rules := [][]string{}
	for _, v := range in.CasbinInfoList {
		key := authorityId + v.Path + v.Method
		if _, ok := deduplicateMap[key]; !ok {
			deduplicateMap[key] = true
			rules = append(rules, []string{authorityId, v.Path, v.Method})
		}
	}

	// 记录旧策略 更新失败时回滚 避免该角色权限全丢
	oldRules := l.svcCtx.Casbin.GetFilteredPolicy(0, authorityId)
	if _, err := l.svcCtx.Casbin.RemoveFilteredPolicy(0, authorityId); err != nil {
		return nil, err
	}

	// 传空列表表示清空该角色全部权限
	if len(rules) > 0 {
		success, err := l.svcCtx.Casbin.AddPolicies(rules)
		if err != nil || !success {
			logx.WithContext(l.ctx).Errorf("UpdateCasbinData AddPolicies err: %v authorityId: %s", err, authorityId)
			// 回滚旧策略
			if len(oldRules) > 0 {
				if _, rbErr := l.svcCtx.Casbin.AddPolicies(oldRules); rbErr != nil {
					logx.WithContext(l.ctx).Errorf("UpdateCasbinData rollback err: %v authorityId: %s", rbErr, authorityId)
				}
			}
			return nil, errors.New("添加策略失败,请联系管理员")
		}
	}

	return &pb.NoDataResponse{}, nil
}
