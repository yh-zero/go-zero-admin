package userlogic

import (
	"context"
	"errors"
	"github.com/zeromicro/go-zero/core/logx"
	"go-zero-admin/application/applet/rpc/internal/model"
	"go-zero-admin/application/applet/rpc/internal/svc"
	"go-zero-admin/application/applet/rpc/pb"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type UpdateUserInfoLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewUpdateUserInfoLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateUserInfoLogic {
	return &UpdateUserInfoLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}
func (l *UpdateUserInfoLogic) UpdateUserInfo(in *pb.UpdateUserInfoRequest) (*pb.NoDataResponse, error) {
	if in.GetUserInfo() == nil || in.UserInfo.ID <= 0 {
		return nil, userError("用户ID无效")
	}
	updates, err := userUpdates(in)
	if err != nil {
		return nil, err
	}
	err = l.svcCtx.DB.WithContext(l.ctx).Transaction(func(tx *gorm.DB) error {
		var user model.SysUser
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&user, in.UserInfo.ID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return userError("用户不存在")
			}
			return err
		}
		if in.UpdateAuthorities || updates["authority_id"] != nil {
			ids := in.AuthorityIds
			if !in.UpdateAuthorities {
				if err := tx.Model(&model.SysUserAuthority{}).Where("sys_user_id = ?", user.ID).Pluck("sys_authority_authority_id", &ids).Error; err != nil {
					return err
				}
			}
			defaultID := user.AuthorityId
			if value, ok := updates["authority_id"]; ok {
				defaultID = value.(int64)
			}
			ids, err = validateUserAuthorities(tx, ids, defaultID)
			if err != nil {
				return err
			}
			if in.UpdateAuthorities {
				if err := replaceUserAuthorities(tx, user.ID, ids); err != nil {
					return err
				}
			}
		}
		if len(updates) == 0 {
			return nil
		}
		return tx.Model(&user).Updates(updates).Error
	})
	if err != nil {
		return nil, err
	}
	return &pb.NoDataResponse{}, nil
}
func userUpdates(in *pb.UpdateUserInfoRequest) (map[string]any, error) {
	values := map[string]any{}
	for _, field := range in.UpdateFields {
		switch field {
		case "nickName":
			values["nick_name"] = in.UserInfo.NickName
		case "phone":
			values["phone"] = in.UserInfo.Phone
		case "email":
			values["email"] = in.UserInfo.Email
		case "headerImg":
			values["header_img"] = in.UserInfo.HeaderImg
		case "sideMode":
			values["side_mode"] = in.UserInfo.SideMode
		case "authorityId":
			if in.UserInfo.AuthorityId <= 0 {
				return nil, userError("默认角色ID无效")
			}
			values["authority_id"] = in.UserInfo.AuthorityId
		case "enable":
			if in.UserInfo.Enable != 1 && in.UserInfo.Enable != 2 {
				return nil, userError("用户状态只能为1或2")
			}
			values["enable"] = in.UserInfo.Enable
		default:
			return nil, userError("不支持更新该用户字段")
		}
	}
	return values, nil
}
