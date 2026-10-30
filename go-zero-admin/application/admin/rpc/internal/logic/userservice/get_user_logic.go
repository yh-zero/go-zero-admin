package userservicelogic

import (
	"context"

	"github.com/zeromicro/go-zero/core/logx"

	adminpb "go-zero-admin/application/admin/rpc/adminpb"
	"go-zero-admin/application/admin/rpc/internal/svc"
	"go-zero-admin/pkg/xerr"
)

type GetUserLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetUserLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetUserLogic {
	return &GetUserLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

func (l *GetUserLogic) GetUser(in *adminpb.GetUserRequest) (*adminpb.GetUserResponse, error) {
	user, err := l.svcCtx.SysUserModel.FindOne(l.ctx, uint64(in.UserId))
	if err != nil {
		return nil, xerr.NewErrCode(xerr.USER_NOT_FOUND)
	}
	var deptID, avatarID int64
	if user.DeptId.Valid {
		deptID = user.DeptId.Int64
	}
	if user.AvatarFileId.Valid {
		avatarID = user.AvatarFileId.Int64
	}
	info := &adminpb.UserInfo{
		Id: int64(user.Id), Username: user.Username, Nickname: user.Nickname,
		DeptId: deptID, AvatarFileId: avatarID, Gender: int32(user.Gender), Status: int32(user.Status),
		CreatedAt: user.CreatedAt.Unix(), UpdatedAt: user.UpdatedAt.Unix(),
	}
	info.RealName = user.RealName
	info.Email = user.Email
	info.Phone = user.Phone
	info.Remark = user.Remark
	if user.LastLoginAt.Valid {
		info.LastLoginAt = user.LastLoginAt.Time.Unix()
	}
	info.LastLoginIP = user.LastLoginIp
	// 查部门名
	if deptID > 0 {
		dept, err := l.svcCtx.SysDeptModel.FindOne(l.ctx, uint64(deptID))
		if err == nil {
			info.DeptName = dept.Name
		}
	}
	return &adminpb.GetUserResponse{User: info}, nil
}
