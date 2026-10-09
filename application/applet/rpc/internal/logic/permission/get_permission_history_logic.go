package permissionlogic

import (
	"context"

	"go-zero-admin/application/applet/rpc/internal/logic/accessutil"
	"go-zero-admin/application/applet/rpc/internal/model"
	"gorm.io/gorm"

	"go-zero-admin/application/applet/rpc/internal/svc"
	"go-zero-admin/application/applet/rpc/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetPermissionHistoryLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetPermissionHistoryLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetPermissionHistoryLogic {
	return &GetPermissionHistoryLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *GetPermissionHistoryLogic) GetPermissionHistory(in *pb.GetPermissionHistoryRequest) (*pb.GetPermissionHistoryResponse, error) {
	if in == nil || in.AuthorityId <= 0 || in.Kind != "" && !validKind(in.Kind) {
		return nil, historyError("历史查询参数无效")
	}
	page, size := int64(1), int64(20)
	if in.PageRequest != nil {
		if in.PageRequest.PageNo != 0 {
			page = in.PageRequest.PageNo
		}
		if in.PageRequest.PageSize != 0 {
			size = in.PageRequest.PageSize
		}
	}
	offset, limit, err := accessutil.Page(page, size)
	if err != nil {
		return nil, err
	}
	out := &pb.GetPermissionHistoryResponse{List: []*pb.PermissionChange{}}
	err = accessutil.PermissionRead(l.svcCtx.DB.WithContext(l.ctx), func(tx *gorm.DB) error {
		query := tx.Model(&model.SysPermissionChange{}).Where("authority_id = ?", in.AuthorityId)
		if in.Kind != "" {
			query = query.Where("kind = ?", in.Kind)
		}
		if err := query.Count(&out.Total).Error; err != nil {
			return err
		}
		var rows []model.SysPermissionChange
		if err := query.Order("id DESC").Offset(offset).Limit(limit).Find(&rows).Error; err != nil {
			return err
		}
		for _, row := range rows {
			change, err := changePB(row)
			if err != nil {
				return err
			}
			out.List = append(out.List, change)
		}
		return nil
	})
	return out, err
}
