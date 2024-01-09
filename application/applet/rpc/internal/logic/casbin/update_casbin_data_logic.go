package casbinlogic

import (
	"context"
	"fmt"
	"github.com/pkg/errors"
	"strconv"

	"go-zero-admin/application/applet/rpc/internal/svc"
	"go-zero-admin/application/applet/rpc/pb"

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
	fmt.Println("authorityId", authorityId)
	_, err := l.ClearCasbin(0, authorityId)
	if err != nil {
		return nil, err
	}
	//做权限去重处理
	rules := [][]string{}
	deduplicateMap := make(map[string]bool)
	for _, v := range in.CasbinInfoList {
		key := authorityId + v.Path + v.Method
		if _, ok := deduplicateMap[key]; !ok {
			deduplicateMap[key] = true
			rules = append(rules, []string{authorityId, v.Path, v.Method})
		}
	}
	csb := l.svcCtx.Config.CasbinConf.MustNewCasbinWithRedisWatcher(l.svcCtx.Config.DB.DataSource, l.svcCtx.Config.BizRedis)
	success, err := csb.AddPolicies(rules)
	if !success {
		return nil, errors.New("存在相同api,添加失败,请联系管理员")
	}

	return &pb.NoDataResponse{}, err
}

func (l *UpdateCasbinDataLogic) ClearCasbin(v int, p ...string) (bool, error) {
	csb := l.svcCtx.Config.CasbinConf.MustNewCasbinWithRedisWatcher(l.svcCtx.Config.DB.DataSource, l.svcCtx.Config.BizRedis)
	success, err := csb.RemoveFilteredPolicy(v, p...)
	return success, err
}
