package userservicelogic

import (
	"context"

	"github.com/zeromicro/go-zero/core/logx"

	adminpb "go-zero-admin/application/admin/rpc/adminpb"
	"go-zero-admin/application/admin/rpc/internal/model"
	"go-zero-admin/application/admin/rpc/internal/svc"
	"go-zero-admin/pkg/xerr"
)

type GetUserListLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetUserListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetUserListLogic {
	return &GetUserListLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

func (l *GetUserListLogic) GetUserList(in *adminpb.GetUserListRequest) (*adminpb.GetUserListResponse, error) {
	rows, total, err := l.svcCtx.SysUserModel.ListUserPage(l.ctx, model.UserListFilter{
		Actor:    model.UserActor{UserId: in.GetActor().GetUserId(), DeptId: in.GetActor().GetDeptId()},
		Username: in.Username, Nickname: in.Nickname, DeptId: in.DeptId, Status: int64(in.Status),
	}, in.GetPage().GetPage(), in.GetPage().GetPageSize())
	if err != nil {
		l.Errorf("query users error: %v", err)
		return nil, passwordMutationError(err)
	}
	list := make([]*adminpb.UserInfo, 0, len(rows))
	ids := make([]int64, 0, len(rows))
	for _, r := range rows {
		ids = append(ids, r.Id)
	}
	roles, err := l.svcCtx.SysUserModel.ListUserRoleIds(l.ctx, ids)
	if err != nil {
		return nil, xerr.NewErrCode(xerr.DB_ERROR)
	}
	for _, r := range rows {
		list = append(list, &adminpb.UserInfo{
			Id: r.Id, Username: r.Username, Nickname: r.Nickname,
			RealName: r.RealName, Email: r.Email, Phone: r.Phone,
			DeptId: r.DeptId, AvatarFileId: r.AvatarFileId,
			Gender: r.Gender, Status: r.Status, Remark: r.Remark,
			LastLoginAt: r.LastLoginAt, LastLoginIP: r.LastLoginIp,
			CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt,
			RoleIds: roles[r.Id],
		})
	}
	return &adminpb.GetUserListResponse{List: list, Page: &adminpb.PageResponse{Total: total}}, nil
}
