package appauthservicelogic

import (
	"context"

	"github.com/zeromicro/go-zero/core/logx"

	apppb "go-zero-admin/application/app/rpc/apppb"
	"go-zero-admin/application/app/rpc/internal/svc"
	"go-zero-admin/pkg/apptoken"
)

type AppAccessLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewAppAccessLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AppAccessLogic {
	return &AppAccessLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

// AppAccess 校验 token 有效性（Phase 2：零查库，status 看快照）
func (l *AppAccessLogic) AppAccess(in *apppb.AppAccessRequest) (*apppb.AppAccessResponse, error) {
	claims, err := apptoken.Claims(l.svcCtx.Config.JwtAuth.AccessSecret, in.AccessToken)
	if err != nil {
		return nil, err
	}
	if err := apptoken.MustActive(claims); err != nil {
		return nil, err
	}
	// 零查库：直接返回快照信息
	return &apppb.AppAccessResponse{User: &apppb.AppUserInfo{
		Id: claims.UserID, Status: claims.Status, Nickname: "",
	}}, nil
}
