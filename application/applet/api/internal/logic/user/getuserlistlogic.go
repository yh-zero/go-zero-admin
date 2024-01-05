package user

import (
	"context"
	"github.com/jinzhu/copier"
	"go-zero-admin/application/applet/rpc/pb"

	"go-zero-admin/application/applet/api/internal/svc"
	"go-zero-admin/application/applet/api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetUserListLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetUserListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetUserListLogic {
	return &GetUserListLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetUserListLogic) GetUserList(req *types.GetUserListRequest) (resp *types.GetUserListResponse, err error) {
	var page pb.PageRequest
	if req.PageNo == 0 {
		req.PageNo = l.svcCtx.Config.Page.PageNo
	}
	if req.PageSize == 0 {
		req.PageSize = l.svcCtx.Config.Page.PageSize
	}
	page.PageNo = req.PageNo
	page.PageSize = req.PageSize

	getUserList, err := l.svcCtx.AppletUserRPC.GetUserList(l.ctx, &pb.GetUserListRequest{PageRequest: &page})
	if err != nil {
		logx.Errorf("l.svcCtx.AppletUserRPC.GetUserList err: %v", err)
		return nil, err
	}
	var typesUserInfoList []types.UserInfo
	_ = copier.Copy(&typesUserInfoList, getUserList.UserInfoList)

	return &types.GetUserListResponse{
		List: typesUserInfoList,
		PageResponse: types.PageResponse{
			Total:    getUserList.Total,
			PageNo:   page.PageNo,
			PageSize: page.PageSize,
		},
	}, nil
}
