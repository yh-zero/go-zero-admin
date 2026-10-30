package user

import (
	"context"

	"go-zero-admin/application/admin/api/internal/svc"
	"go-zero-admin/application/admin/api/internal/types"
	adminpb "go-zero-admin/application/admin/rpc/adminpb"

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

func (l *GetUserListLogic) GetUserList(req *types.GetUserListReq) (resp *types.GetUserListResp, err error) {
	rpcResp, err := l.svcCtx.UserRpc.GetUserList(l.ctx, &adminpb.GetUserListRequest{
		Page:     &adminpb.PageRequest{Page: req.Page, PageSize: req.PageSize},
		Username: req.Username,
		Nickname: req.Nickname,
		DeptId:   req.DeptId,
		Status:   req.Status,
	})
	if err != nil {
		return nil, err
	}
	list := make([]*types.UserInfo, 0, len(rpcResp.List))
	for _, u := range rpcResp.List {
		list = append(list, &types.UserInfo{
			Id: u.Id, Username: u.Username, Nickname: u.Nickname,
			RealName: u.RealName, Email: u.Email, Phone: u.Phone,
			DeptId: u.DeptId, DeptName: u.DeptName,
			AvatarFileId: u.AvatarFileId, Gender: u.Gender, Status: u.Status,
			Remark: u.Remark, LastLoginAt: u.LastLoginAt, LastLoginIp: u.LastLoginIP,
			CreatedAt: u.CreatedAt, UpdatedAt: u.UpdatedAt,
		})
	}
	return &types.GetUserListResp{List: list, Total: rpcResp.Page.Total}, nil
}
