package userlogic

import (
	"context"
	"github.com/jinzhu/copier"
	"go-zero-admin/application/applet/rpc/internal/model"

	"go-zero-admin/application/applet/rpc/internal/svc"
	"go-zero-admin/application/applet/rpc/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetUserListLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetUserListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetUserListLogic {
	return &GetUserListLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 分页获取用户列表
func (l *GetUserListLogic) GetUserList(in *pb.GetUserListRequest) (*pb.GetUserListResponse, error) {
	offset := int(in.PageRequest.PageSize * (in.PageRequest.PageNo - 1))
	db := l.svcCtx.DB.Model(&model.SysUser{})
	var sysUserList []model.SysUser
	var total int64
	err := db.Count(&total).Error
	if err != nil {
		return nil, err
	}
	err = db.Limit(int(in.PageRequest.PageSize)).Offset(offset).Preload("Authorities").Preload("Authority").Find(&sysUserList).Error
	//var userInfoList []pb.UserInfo
	var pbGetUserList pb.GetUserListResponse
	_ = copier.Copy(&pbGetUserList.UserInfoList, sysUserList)
	pbGetUserList.Total = total

	return &pbGetUserList, nil
}
