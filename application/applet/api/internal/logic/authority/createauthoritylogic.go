package authority

import (
	"context"

	"go-zero-admin/application/applet/api/internal/svc"
	"go-zero-admin/application/applet/api/internal/types"
	"go-zero-admin/application/applet/rpc/pb"

	"github.com/jinzhu/copier"
	"github.com/zeromicro/go-zero/core/logx"
)

type CreateAuthorityLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewCreateAuthorityLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateAuthorityLogic {
	return &CreateAuthorityLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *CreateAuthorityLogic) CreateAuthority(req *types.CreateAuthorityRequest) (resp *types.CreateAuthorityResponse, err error) {
	var pbSysAuthority pb.SysAuthority
	_ = copier.Copy(&pbSysAuthority, req)

	createAuthority, err := l.svcCtx.AppletAuthorityRPC.CreateAuthority(l.ctx, &pb.CreateAuthorityRequest{SysAuthority: &pbSysAuthority})
	if err != nil {
		logx.Errorf("l.svcCtx.AppletAuthorityRPC.CreateAuthority 错误 error: %v", err)
		return nil, err
	}

	var typesSysAuthority types.SysAuthority
	_ = copier.Copy(&typesSysAuthority, createAuthority)

	return &types.CreateAuthorityResponse{SysAuthority: typesSysAuthority}, nil
}
