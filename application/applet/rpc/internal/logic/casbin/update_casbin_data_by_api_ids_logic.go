package casbinlogic

import (
	"context"
	"github.com/jinzhu/copier"
	"github.com/zeromicro/go-zero/core/logx"
	"go-zero-admin/application/applet/rpc/internal/logic/accessutil"
	"go-zero-admin/application/applet/rpc/internal/model"
	"go-zero-admin/application/applet/rpc/internal/svc"
	"go-zero-admin/application/applet/rpc/pb"
	"go-zero-admin/pkg/result/xerr"
	"gorm.io/gorm"
)

type UpdateCasbinDataByApiIdsLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewUpdateCasbinDataByApiIdsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateCasbinDataByApiIdsLogic {
	return &UpdateCasbinDataByApiIdsLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

func (l *UpdateCasbinDataByApiIdsLogic) UpdateCasbinDataByApiIds(in *pb.UpdateCasbinDataByApiIdsRequest) (*pb.UpdateCasbinDataByApiIdsResponse, error) {
	ids, err := accessutil.UniqueIDs(in.ApiIds)
	if err != nil {
		return nil, err
	}
	var apis []model.SysApi
	err = accessutil.PolicyTransaction(l.svcCtx, func(tx *gorm.DB) error {
		if len(ids) > 0 {
			if err := tx.Where("id IN ?", ids).Find(&apis).Error; err != nil {
				return err
			}
		}
		if len(apis) != len(ids) {
			return xerr.NewErrCodeMsg(xerr.REUQEST_PARAM_ERROR, "部分API ID不存在，授权未修改")
		}
		entries := make([]*pb.CasbinInfo, 0, len(apis))
		for _, api := range apis {
			entries = append(entries, &pb.CasbinInfo{Path: api.Path, Method: api.Method})
		}
		return replaceRolePolicies(tx, in.AuthorityId, entries)
	})
	if err != nil {
		return nil, err
	}
	result := &pb.UpdateCasbinDataByApiIdsResponse{}
	if err := copier.Copy(&result.SysApis, apis); err != nil {
		return nil, err
	}
	return result, nil
}
