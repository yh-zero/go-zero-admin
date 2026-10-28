package session

import (
	"context"
	"github.com/jinzhu/copier"
	"github.com/zeromicro/go-zero/core/logx"
	"go-zero-admin/application/applet/api/internal/svc"
	"go-zero-admin/application/applet/api/internal/types"
)

type MeLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewMeLogic(ctx context.Context, svcCtx *svc.ServiceContext) *MeLogic {
	return &MeLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}
func (l *MeLogic) Me(req *types.MeRequest) (*types.UserInfo, error) {
	response, err := l.svcCtx.AppletUserRPC.GetCurrentUser(l.ctx, sessionRequest(l.ctx))
	if err != nil {
		return nil, err
	}
	out := &types.UserInfo{}
	if err := copier.Copy(out, response.UserInfo); err != nil {
		return nil, err
	}
	return out, nil
}
