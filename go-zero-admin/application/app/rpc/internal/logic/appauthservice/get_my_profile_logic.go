package appauthservicelogic

import (
	"context"

	"github.com/zeromicro/go-zero/core/logx"

	apppb "go-zero-admin/application/app/rpc/apppb"
	"go-zero-admin/application/app/rpc/internal/svc"
	"go-zero-admin/pkg/apptoken"
	"go-zero-admin/pkg/xerr"
)

type GetMyProfileLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetMyProfileLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetMyProfileLogic {
	return &GetMyProfileLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

// GetMyProfile 查自己资料（Phase 1：数据隔离，user_id 只来自 token）
func (l *GetMyProfileLogic) GetMyProfile(in *apppb.AppProfileRequest) (*apppb.AppProfileResponse, error) {
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
	return &apppb.AppProfileResponse{User: mapAppUser(user)}, nil
}
