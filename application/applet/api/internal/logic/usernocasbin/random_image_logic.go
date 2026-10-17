package usernocasbin

import (
	"context"

	"go-zero-admin/application/applet/api/internal/svc"
	"go-zero-admin/application/applet/api/internal/types"
	"go-zero-admin/application/applet/rpc/pb"
	"go-zero-admin/pkg/result/xerr"

	"github.com/zeromicro/go-zero/core/logx"
)

type RandomImageLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewRandomImageLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RandomImageLogic {
	return &RandomImageLogic{Logger: logx.WithContext(ctx), ctx: ctx, svcCtx: svcCtx}
}

// RandomImage is a thin pass-through to the RPC tier. The RPC owns the
// image generation, the Redis key prefix, the TTL, and the image dimensions.
func (l *RandomImageLogic) RandomImage(_ *types.RandomImageRequest) (*types.RandomImageResponse, error) {
	if l.svcCtx.AppletCaptchaRPC == nil {
		return nil, xerr.NewErrCodeMsg(900000, "验证码服务未配置")
	}
	resp, err := l.svcCtx.AppletCaptchaRPC.GenerateImage(l.ctx, &pb.NoDataResponse{})
	if err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, xerr.NewErrCodeMsg(900000, "验证码服务响应无效")
	}
	return &types.RandomImageResponse{CaptchaId: resp.Id, CaptchaImg: resp.Image}, nil
}
