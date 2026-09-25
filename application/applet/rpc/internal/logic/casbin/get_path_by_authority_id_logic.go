package casbinlogic

import (
	"context"
	gormadapter "github.com/casbin/gorm-adapter/v3"
	"github.com/zeromicro/go-zero/core/logx"
	"go-zero-admin/application/applet/rpc/internal/logic/accessutil"
	"go-zero-admin/application/applet/rpc/internal/svc"
	"go-zero-admin/application/applet/rpc/pb"
	"strconv"
)

type GetPathByAuthorityIdLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetPathByAuthorityIdLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetPathByAuthorityIdLogic {
	return &GetPathByAuthorityIdLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

func (l *GetPathByAuthorityIdLogic) GetPathByAuthorityId(in *pb.GetPathByAuthorityIdRequest) (*pb.GetPathByAuthorityIdResponse, error) {
	if err := accessutil.RequireRole(l.svcCtx.DB.DB, in.AuthorityId); err != nil {
		return nil, err
	}
	var rules []gormadapter.CasbinRule
	if err := l.svcCtx.DB.Where("ptype = ? AND v0 = ?", "p", strconv.FormatInt(in.AuthorityId, 10)).Order("v1,v2").Find(&rules).Error; err != nil {
		return nil, err
	}
	result := &pb.GetPathByAuthorityIdResponse{CasbinInfoList: make([]*pb.CasbinInfo, 0, len(rules))}
	for _, rule := range rules {
		result.CasbinInfoList = append(result.CasbinInfoList, &pb.CasbinInfo{Path: rule.V1, Method: rule.V2})
	}
	return result, nil
}
