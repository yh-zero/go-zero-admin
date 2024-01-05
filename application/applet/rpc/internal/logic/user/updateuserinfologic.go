package userlogic

import (
	"context"
	"github.com/zeromicro/go-zero/core/logx"
	"go-zero-admin/application/applet/rpc/internal/model"
	"go-zero-admin/application/applet/rpc/internal/svc"
	"go-zero-admin/application/applet/rpc/pb"
	"reflect"
	"time"
)

type UpdateUserInfoLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewUpdateUserInfoLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateUserInfoLogic {
	return &UpdateUserInfoLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 修改用户信息
func (l *UpdateUserInfoLogic) UpdateUserInfo(in *pb.UpdateUserInfoRequest) (*pb.UpdateUserInfoResponse, error) {
	updates := map[string]interface{}{
		"updated_at": time.Now(),
	}

	// 字段名称与数据库列名的映射关系
	fieldToColumn := map[string]string{
		"NickName":  "nick_name",
		"HeaderImg": "header_img",
		"Phone":     "phone",
		"Email":     "email",
		"SideMode":  "side_mode",
		"Enable":    "enable",
	}

	userInfoValue := reflect.ValueOf(in.UserInfo)
	if userInfoValue.Kind() == reflect.Ptr {
		userInfoValue = userInfoValue.Elem()
	}

	for fieldName, columnName := range fieldToColumn {
		fieldValue := userInfoValue.FieldByName(fieldName)
		if fieldValue.IsValid() {
			switch fieldValue.Kind() {
			case reflect.String:
				if fieldValue.String() != "" {
					updates[columnName] = fieldValue.Interface()
				}
			case reflect.Int, reflect.Int64, reflect.Uint, reflect.Uint64:
				if fieldValue.Int() != 0 {
					updates[columnName] = fieldValue.Interface()
				}
			}
		}
	}

	err := l.svcCtx.DB.Model(&model.SysUser{}).
		Select("updated_at", "nick_name", "header_img", "phone", "email", "side_mode", "enable").
		Where("id = ?", in.UserInfo.ID).
		Updates(updates).Error

	return &pb.UpdateUserInfoResponse{}, err
}

//if in.UserInfo.NickName != "" {
//	updates["nick_name"] = in.UserInfo.NickName
//}
//if in.UserInfo.HeaderImg != "" {
//	updates["header_img"] = in.UserInfo.HeaderImg
//}
//if in.UserInfo.Phone != "" {
//	updates["phone"] = in.UserInfo.Phone
//}
//if in.UserInfo.Email != "" {
//	updates["email"] = in.UserInfo.Email
//}
//if in.UserInfo.SideMode != "" {
//	updates["side_mode"] = in.UserInfo.SideMode
//}
//if in.UserInfo.Enable != 0 {
//	updates["enable"] = in.UserInfo.Enable
//}
