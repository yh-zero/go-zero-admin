package authority

import (
	"context"
	"fmt"

	"go-zero-admin/application/applet/api/internal/svc"
	"go-zero-admin/application/applet/api/internal/types"
	"go-zero-admin/application/applet/rpc/pb"

	"github.com/jinzhu/copier"
	"github.com/zeromicro/go-zero/core/logx"
)

type UpdateAuthorityLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewUpdateAuthorityLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateAuthorityLogic {
	return &UpdateAuthorityLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *UpdateAuthorityLogic) UpdateAuthority(req *types.UpdateAuthorityRequest) (resp *types.UpdateAuthorityResponse, err error) {
	var pbSysAuthority pb.SysAuthority
	_ = copier.Copy(&pbSysAuthority, req)
	fmt.Println("----------- pbUpdateAuthority", pbSysAuthority)
	fmt.Println("------------ req", req)
	updateAuthority, err := l.svcCtx.AppletAuthorityRPC.UpdateAuthority(l.ctx, &pb.UpdateAuthorityRequest{SysAuthority: &pbSysAuthority})
	if err != nil {
		logx.Errorf("l.svcCtx.AppletAuthorityRPC.UpdateAuthority 错误 error: %v", err)
		return nil, err
	}
	fmt.Println("----------- updateAuthority", updateAuthority)

	var typesSysAuthority types.SysAuthority
	_ = copier.Copy(&typesSysAuthority, updateAuthority.SysAuthority)
	fmt.Println("----------- typesSysAuthority", typesSysAuthority)

	return &types.UpdateAuthorityResponse{SysAuthority: typesSysAuthority, Message: "更新成功"}, nil
}
