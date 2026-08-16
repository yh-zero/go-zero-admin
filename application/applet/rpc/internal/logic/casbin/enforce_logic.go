package casbinlogic

import (
	"context"

	"go-zero-admin/application/applet/rpc/internal/svc"
	"go-zero-admin/application/applet/rpc/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

type EnforceLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewEnforceLogic(ctx context.Context, svcCtx *svc.ServiceContext) *EnforceLogic {
	return &EnforceLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// casbin 鉴权 供api网关/中间件调用
func (l *EnforceLogic) Enforce(in *pb.EnforceRequest) (*pb.EnforceResponse, error) {
	ok, err := l.svcCtx.Casbin.Enforce(in.AuthorityId, in.Path, in.Method)
	if err != nil {
		logx.WithContext(l.ctx).Errorf("casbin Enforce err: %v", err)
		return nil, err
	}

	return &pb.EnforceResponse{Pass: ok}, nil
}
