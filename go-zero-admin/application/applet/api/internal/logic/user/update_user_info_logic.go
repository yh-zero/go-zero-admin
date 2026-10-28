package user

import (
	"context"
	"github.com/zeromicro/go-zero/core/logx"
	"go-zero-admin/application/applet/api/internal/svc"
	"go-zero-admin/application/applet/api/internal/types"
	"go-zero-admin/application/applet/rpc/pb"
	"go-zero-admin/pkg/ctxJwt"
	"go-zero-admin/pkg/result/xerr"
)

type UpdateUserInfoLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewUpdateUserInfoLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateUserInfoLogic {
	return &UpdateUserInfoLogic{Logger: logx.WithContext(ctx), ctx: ctx, svcCtx: svcCtx}
}
func (l *UpdateUserInfoLogic) UpdateUserInfo(req *types.UpdateUserInfoRequest) (*types.MessageResponse, error) {
	if req.Enable != nil && *req.Enable == 2 && req.ID == ctxJwt.GetJwtDataID(l.ctx) {
		return nil, xerr.NewErrCodeMsg(xerr.REUQEST_PARAM_ERROR, "不能冻结当前登录用户")
	}
	if _, err := l.svcCtx.AppletUserRPC.UpdateUserInfo(l.ctx, userUpdateRequest(req)); err != nil {
		return nil, err
	}
	return &types.MessageResponse{Message: "更新成功"}, nil
}

// 显式字段列表保留“未传”和“传空值”的区别。
func userUpdateRequest(req *types.UpdateUserInfoRequest) *pb.UpdateUserInfoRequest {
	user := &pb.UserInfo{ID: req.ID}
	input := &pb.UpdateUserInfoRequest{UserInfo: user, AuthorityIds: req.AuthorityIds, UpdateAuthorities: req.AuthorityIds != nil}
	setString := func(name string, value *string, destination *string) {
		if value != nil {
			*destination = *value
			input.UpdateFields = append(input.UpdateFields, name)
		}
	}
	setString("nickName", req.NickName, &user.NickName)
	setString("phone", req.Phone, &user.Phone)
	setString("email", req.Email, &user.Email)
	setString("headerImg", req.HeaderImg, &user.HeaderImg)
	setString("sideMode", req.SideMode, &user.SideMode)
	if req.Enable != nil {
		user.Enable = *req.Enable
		input.UpdateFields = append(input.UpdateFields, "enable")
	}
	if req.AuthorityId != nil {
		user.AuthorityId = *req.AuthorityId
		input.UpdateFields = append(input.UpdateFields, "authorityId")
	}
	return input
}
