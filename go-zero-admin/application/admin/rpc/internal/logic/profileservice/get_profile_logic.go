package profileservicelogic

import (
	"context"

	"github.com/zeromicro/go-zero/core/logx"

	adminpb "go-zero-admin/application/admin/rpc/adminpb"
	"go-zero-admin/application/admin/rpc/internal/svc"
	"go-zero-admin/pkg/xerr"
)

type GetProfileLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetProfileLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetProfileLogic {
	return &GetProfileLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

func (l *GetProfileLogic) GetProfile(in *adminpb.GetProfileRequest) (*adminpb.GetProfileResponse, error) {
	user, err := l.svcCtx.SysUserModel.FindOne(l.ctx, uint64(in.Actor.UserId))
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
	if user.LastLoginAt.Valid {
		info.LastLoginAt = user.LastLoginAt.Time.Unix()
	}
	info.LastLoginIP = user.LastLoginIp
	if deptID > 0 {
		dept, err := l.svcCtx.SysDeptModel.FindOne(l.ctx, uint64(deptID))
		if err == nil {
			info.DeptName = dept.Name
		}
	}
	return &adminpb.GetProfileResponse{User: info}, nil
}
