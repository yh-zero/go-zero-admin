package appauthservicelogic

import (
	"context"
	"time"

	"github.com/zeromicro/go-zero/core/logx"

	apppb "go-zero-admin/application/app/rpc/apppb"
	"go-zero-admin/application/app/rpc/internal/svc"
	"go-zero-admin/pkg/apptoken"
	"go-zero-admin/pkg/xerr"
)

type UpdateMyProfileLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewUpdateMyProfileLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateMyProfileLogic {
	return &UpdateMyProfileLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

// UpdateMyProfile 更新自己资料（Phase 1：user_id 只来自 token，禁止请求参数指定）
func (l *UpdateMyProfileLogic) UpdateMyProfile(in *apppb.AppProfileRequest) (*apppb.AppProfileResponse, error) {
	claims, err := apptoken.Claims(l.svcCtx.Config.JwtAuth.AccessSecret, in.AccessToken)
	if err != nil {
		return nil, err
	}
	if err := apptoken.MustActive(claims); err != nil {
		return nil, err
	}
	user, err := l.svcCtx.AppUserModel.FindOne(l.ctx, uint64(claims.UserID))
	if err != nil {
		return nil, xerr.NewErrCodeMsg(400, "用户不存在")
	}
	user.Nickname = in.Nickname
	user.UpdatedAt = time.Now()
	err = l.svcCtx.AppUserModel.Update(l.ctx, user)
	if err != nil {
		return nil, xerr.NewErrCode(xerr.DB_ERROR)
	}
	return &apppb.AppProfileResponse{User: mapAppUser(user)}, nil
}
