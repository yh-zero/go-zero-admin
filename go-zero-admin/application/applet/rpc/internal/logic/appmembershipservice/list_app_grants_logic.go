package appmembershipservicelogic

import (
	"context"
	appaccount "go-zero-admin/application/applet/rpc/internal/logic/appaccountservice"
	"go-zero-admin/application/applet/rpc/internal/model"

	"go-zero-admin/application/applet/rpc/internal/svc"
	"go-zero-admin/application/applet/rpc/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListAppGrantsLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewListAppGrantsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListAppGrantsLogic {
	return &ListAppGrantsLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *ListAppGrantsLogic) ListAppGrants(in *pb.AppGrantListRequest) (*pb.AppGrantListResponse, error) {
	if in == nil {
		return nil, membershipError("请求参数不能为空")
	}
	if err := appaccount.AuthorizeAdmin(l.ctx, l.svcCtx, in.Actor, grantsPath, "GET"); err != nil {
		return nil, err
	}
	page, size := int64(1), int64(10)
	if in.Page != nil {
		page, size = in.Page.PageNo, in.Page.PageSize
	}
	if page < 1 || page > 1000000 || size < 1 || size > 100 || in.UserID < 0 {
		return nil, membershipError("分页或用户参数无效")
	}
	query := l.svcCtx.DB.WithContext(l.ctx).Model(&model.AppMembershipGrant{})
	if in.UserID > 0 {
		query = query.Where("user_id = ?", in.UserID)
	}
	result := &pb.AppGrantListResponse{List: []*pb.AppGrantInfo{}}
	if err := query.Count(&result.Total).Error; err != nil {
		return nil, err
	}
	var grants []model.AppMembershipGrant
	if err := query.Order("id DESC").Offset(int((page - 1) * size)).Limit(int(size)).Find(&grants).Error; err != nil {
		return nil, err
	}
	ids := make([]int64, 0, len(grants))
	for _, g := range grants {
		ids = append(ids, g.UserID)
	}
	accounts := map[int64]string{}
	if len(ids) > 0 {
		var identities []model.AppUserIdentity
		if err := l.svcCtx.DB.WithContext(l.ctx).Select("user_id", "identifier").Where("user_id IN ?", ids).Find(&identities).Error; err != nil {
			return nil, err
		}
		for _, identity := range identities {
			accounts[identity.UserID] = identity.Identifier
		}
	}
	now := membershipNow()
	for i := range grants {
		info, err := grantProto(&grants[i], accounts[grants[i].UserID], now)
		if err != nil {
			return nil, err
		}
		result.List = append(result.List, info)
	}
	return result, nil
}
