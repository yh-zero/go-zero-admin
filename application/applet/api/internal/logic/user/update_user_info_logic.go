package user

import (
	"context"
	"fmt"

	"go-zero-admin/application/applet/api/internal/svc"
	"go-zero-admin/application/applet/api/internal/types"
	"go-zero-admin/application/applet/rpc/pb"
	"go-zero-admin/pkg/result/xerr"

	"github.com/zeromicro/go-zero/core/logx"
)

type UpdateUserInfoLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewUpdateUserInfoLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateUserInfoLogic {
	return &UpdateUserInfoLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *UpdateUserInfoLogic) UpdateUserInfo(req *types.UpdateUserInfoRequest) (resp *types.MessageResponse, err error) {
	fmt.Println("-------  req.SideMode", req.SideMode)

	if len(req.AuthorityIds) != 0 {
		_, err = l.svcCtx.AppletUserRPC.UpdateUserAuthorities(l.ctx, &pb.UpdateUserAuthoritiesRequest{ID: req.ID, AuthorityIds: req.AuthorityIds})
		if err != nil {
			logx.Errorf("l.svcCtx.AppletUserRPC.UpdateUserAuthorities err: %v", err)
			return nil, xerr.NewErrCodeMsg(900000, "l.svcCtx.AppletUserRPC.UpdateUserAuthorities")
		}
	}
	userInfo := &pb.UserInfo{
		NickName:  req.NickName,
		SideMode:  req.SideMode,
		HeaderImg: req.HeaderImg,
		Phone:     req.Phone,
		Email:     req.Email,
		Enable:    req.Enable,
		ID:        req.ID,
	}
	_, err = l.svcCtx.AppletUserRPC.UpdateUserInfo(l.ctx, &pb.UpdateUserInfoRequest{UserInfo: userInfo})
	if err != nil {
		logx.Errorf("l.svcCtx.AppletUserRPC.UpdateUserInfo err: %v", err)
		return nil, err
	}

	return &types.MessageResponse{Message: "更新数据成功！"}, nil
}
